/*
Copyright 2016 Google LLC

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    https://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

// Package certs provides splice provisioning support for certificate generation,
// lookup and verification during the provisioning process.
package certs

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"errors"
	"fmt"
	"io/ioutil"
	"math/big"
	"net/http"
	"regexp"
	"time"
)

// GenerateSelfSignedCert generates a self-signed certificate using
// a template and returns the certificate in DER format and its key.
func GenerateSelfSignedCert(cn string, notBefore, notAfter time.Time) ([]byte, *rsa.PrivateKey, error) {
	// A proposed computer name must always satisfy MS naming conventions.
	// https://support.microsoft.com/en-us/help/909264/naming-conventions-in-active-directory-for-computers-domains-sites-and
	invalidName, err := regexp.MatchString(`^$|^\.|[\\/:*?"<>|]|.{15,}$`, cn)
	if invalidName || err != nil {
		return nil, nil, fmt.Errorf("cn(%s) is invalid or empty, regexp.MatchString returned %v", cn, err)
	}

	priv, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return nil, nil, fmt.Errorf("rsa.GenerateKey returned %v", err)
	}

	serialNumberLimit := new(big.Int).Lsh(big.NewInt(1), 128)
	serialNumber, err := rand.Int(rand.Reader, serialNumberLimit)
	if err != nil {
		return nil, nil, fmt.Errorf("unable to generate certificate serial number: %v", err)
	}

	template := x509.Certificate{
		SerialNumber: serialNumber,
		Subject: pkix.Name{
			CommonName: cn,
		},
		Issuer: pkix.Name{
			OrganizationalUnit: []string{"SelfSigned"},
		},
		KeyUsage:  x509.KeyUsageCertSign,
		NotBefore: notBefore,
		NotAfter:  notAfter,
	}

	derBytes, err := x509.CreateCertificate(rand.Reader, &template, &template, &priv.PublicKey, priv)
	if err != nil {
		return nil, nil, fmt.Errorf("unable to generate a self-signed certificate, x509.CreateCertificate returned %v", err)
	}

	return derBytes, priv, nil
}

// VerifyCert takes a raw DER encoded cert, verifies that it is valid
// and optionally attempts to verify its certificate chain. It returns
// the DER encoded public key of the certificate.
func VerifyCert(c []byte, hostname, base, path, caOrg, roots string, verify bool) error {
	if !verify {
		return nil
	}
	if len(c) < 1 {
		return fmt.Errorf("verify_certs is set to %t, but %s did not provide a certificate", verify, hostname)
	}

	cert, err := x509.ParseCertificate(c)
	if err != nil {
		return fmt.Errorf("x509.ParseCertificate(c) for %s returned %v", hostname, err)
	}

	//Check that the cert presented is for the same host being joined.
	if err := cert.VerifyHostname(hostname); err != nil {
		return fmt.Errorf("cert.VerifyHostname(%s): %v", hostname, err)
	}
	opts := x509.VerifyOptions{
		Intermediates: x509.NewCertPool(),
		Roots:         x509.NewCertPool(),
		DNSName:       hostname,
		CurrentTime:   time.Now(),
		KeyUsages:     []x509.ExtKeyUsage{x509.ExtKeyUsageAny},
	}

	// Build chain for validation.
	// If issuer fetching is configured, fetch the roots.
	var intermediate, root *x509.Certificate
	if path != "" {
		intermediate, err = fetchIssuer(cert, base, path)
		if err != nil {
			return fmt.Errorf("fetchIssuer of intermediate cert for %q returned %v", hostname, err)
		}
		opts.Intermediates.AddCert(intermediate)

		root, err = fetchIssuer(intermediate, base, path)
		if err != nil {
			return fmt.Errorf("fetchIssuer of root cert for %q returned %v", hostname, err)
		}
		opts.Roots.AddCert(root)
	}

	// If file roots are configured, fetch more roots from the file.
	if roots != "" {
		pem, err := ioutil.ReadFile(roots)
		if err != nil {
			return fmt.Errorf("error reading %q: %v", roots, err)
		}
		if ok := opts.Intermediates.AppendCertsFromPEM(pem); !ok {
			return fmt.Errorf("no certificates found in intermediate bundle at %q", roots)
		}
		if ok := opts.Roots.AppendCertsFromPEM(pem); !ok {
			return fmt.Errorf("no certificates found in root bundle at %q", roots)
		}
	}

	// Validate using prepared cert pools.
	chains, err := cert.Verify(opts)
	if err != nil {
		return fmt.Errorf("x509.Verify of cert for %s returned %v", hostname, err)
	}
	if len(chains) < 1 {
		return fmt.Errorf("cert chain validation for %s failed (chain: %v)", hostname, chains)
	}

	// Check for expected issuing organization if configured.
	if caOrg != "" {
		if !contains(intermediate.Issuer.Organization, caOrg) {
			return fmt.Errorf("expected issuer(%s) not found in intermediate cert issuers (%v)", caOrg, intermediate.Issuer.Organization)
		}
		if !contains(root.Issuer.Organization, caOrg) {
			return fmt.Errorf("expected issuer (%s) not found in root cert issuers (%v)", caOrg, root.Issuer.Organization)
		}
	}

	return nil
}

// PublicKey takes a raw DER encoded cert, and returns only the public
// key portion of the certificate in DER format.
func PublicKey(c []byte) ([]byte, error) {
	cert, err := x509.ParseCertificate(c)
	if err != nil {
		return nil, fmt.Errorf("x509.ParseCertificate(c): %v", err)
	}

	rsa, ok := cert.PublicKey.(*rsa.PublicKey)
	if !ok {
		return nil, fmt.Errorf("unsupported key type, not an RSA public key")
	}

	public, err := x509.MarshalPKIXPublicKey(rsa)
	if err != nil {
		return nil, fmt.Errorf("error marshalling public key: %v", err)
	}

	return public, nil
}

// fetchIssuer returns the public certificate for the issuer of the
// provided leaf certificate located at the base url. If path is not provided
// FetchIssuingCert attempts to calculate it automatically. If no certificate
// is found, an error is returned.
func fetchIssuer(c *x509.Certificate, base string, path string) (*x509.Certificate, error) {
	caURL, err := certPath(base, path, c)
	if err != nil {
		return nil, err
	}
	resp, err := http.Get(caURL)
	if err != nil {
		return nil, fmt.Errorf("HTTP request for %q returned: %v", caURL, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("response status != %v: %v", http.StatusOK, resp.Status)
	}

	raw, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	return x509.ParseCertificate(raw)
}

// certPath returns the path to an issuing CA's public certificate.
// If path is provided, it is appended to base and returned.
// Otherwise, path is calculated using information available
// in the certificate.
func certPath(base string, path string, c *x509.Certificate) (string, error) {
	if path != "" {
		return base + path, nil
	}

	aki := fmt.Sprintf("%x", c.AuthorityKeyId)
	if len(c.Issuer.Organization) < 1 {
		return "", errors.New("certificate issuer lacks organization")
	}
	iorg := c.Issuer.Organization[len(c.Issuer.Organization)-1]

	return base + iorg + "/" + aki + ".cert", nil
}

// contains searches a slice of strings for a value
// and returns true if the value is found.
func contains(list []string, str string) bool {
	for _, item := range list {
		if item == str {
			return true
		}
	}
	return false
}
