/*
Copyright 2019 Google LLC

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

package validators

import (
	"bytes"
	"golang.org/x/net/context"
	"crypto/rsa"
	"crypto/x509"
	"encoding/json"
	"encoding/pem"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"strings"
	"time"

	"google.golang.org/appengine"
	"google.golang.org/appengine/log"
	"google.golang.org/appengine/urlfetch"
	"gopkg.in/square/go-jose.v2/jwt"
	"github.com/google/splice/appengine/server"
	"github.com/google/splice/models"
)

const (
	expectedIssuer = "https://accounts.google.com"
	claimMaxAge    = 5 * time.Minute
)

var (
	// Enable swapping getSigningCert for testing.
	getSigningCert  = getPublicSigningCert
	signingCertsURL = "https://www.googleapis.com/oauth2/v1/certs"
	// Defines the resource types which are permitted for
	// the allowlist.
	defaultAllowedResources = []string{"projects"}
)

// GCEChecker implements validators.Validator and checks if the request
// includes a GCE projectID that is on the allowlist. If ExpandedCheck is
// not nil, it can be used for secondary checks.
type GCEChecker struct {
	ProjectAllowlist map[string]bool
	ExpandedCheck    func(context.Context, string, map[string]bool) error
}

// Check returns StatusSuccess if request metadata includes a GCE
// project ID that is on the allowlist. If the ExpandedCheck member
// is available, it is called to perform additional allowlist
// checks for the project.
func (g GCEChecker) Check(ctx context.Context, req *models.Request) (server.StatusCode, error) {
	// Check that the token is properly signed by a Google public cert.
	claims, err := checkSigningCert(ctx, string(req.GCEMetadata.Identity))
	if err != nil {
		return server.StatusInvalidGCEmeta, fmt.Errorf("checkSigningCert returned %v", err)
	}
	// Check that the claims comes from GCE
	if claims.Issuer != expectedIssuer {
		return server.StatusInvalidGCEmeta, fmt.Errorf("issuer got: %s, want: %s", claims.Issuer, expectedIssuer)
	}
	// Check that the audience is the unattended endpoint for this app.
	expectedAudience := `https://` + appengine.DefaultVersionHostname(ctx) + `/request-unattended`
	if len(claims.Audience) != 1 {
		return server.StatusInvalidGCEmeta, fmt.Errorf("got %d Audience(s), want: 1", claims.Audience)
	}
	if claims.Audience[0] != expectedAudience {
		return server.StatusInvalidGCEmeta, fmt.Errorf("claims.Audience got: %s, want: %s", claims.Audience[0], expectedAudience)
	}
	// Check that the claim isn't too old.
	now := time.Now()
	claimAge := now.Sub(claims.IssuedAt.Time())
	if claimAge > claimMaxAge {
		return server.StatusInvalidGCEmeta, fmt.Errorf("identity claim issued at %v is too old (%d minutes old)", claims.IssuedAt.Time(), claimAge/time.Minute)
	}
	// Check that the claim isn't from too far in the future.
	if -claimAge > claimMaxAge {
		return server.StatusInvalidGCEmeta, fmt.Errorf("identity claim issued at %v is too far (%d minutes) in the future", claims.IssuedAt, claimAge/time.Minute)
	}

	// Check that the verified claim's project ID is in the allowlist.
	p := "projects/" + fmt.Sprintf("%s", claims.Google.ComputeEngine.ProjectID)
	if _, ok := g.ProjectAllowlist[p]; ok {
		log.Infof(ctx, "Request originates from an allowlist project: %s, proceeding.", claims.Google.ComputeEngine.ProjectID)
		return server.StatusSuccess, nil
	}
	if g.ExpandedCheck == nil {
		return server.StatusInvalidGCEmeta, fmt.Errorf("requesting project(%s) is not on the allowlist(%v)", p, g.ProjectAllowlist)
	}
	err = g.ExpandedCheck(ctx, p, g.ProjectAllowlist)
	if err == nil {
		log.Infof(ctx, "Request from project %s originates from an allowlist ancestor, proceeding.", claims.Google.ComputeEngine.ProjectID)
		return server.StatusSuccess, nil
	}
	return server.StatusInvalidGCEmeta, fmt.Errorf("expanded allowlist check returned: %v", err)
}

// NewGCE returns a GCE validator initialized with a sanitized allowlist.
// The permitted resource types can be overidden by the 'allowed' parameter.
func NewGCE(allowed []string) (GCEChecker, error) {
	if allowed == nil {
		allowed = defaultAllowedResources
	}
	w, err := ParseAllowlist(allowed)
	if err != nil {
		return GCEChecker{}, err
	}
	return GCEChecker{ProjectAllowlist: w}, nil
}

// ParseAllowlist parses the allowlist from the AppEngine environment
// variable into a map. Returns an error if a disallowed type is on
// the list.
func ParseAllowlist(allowed []string) (map[string]bool, error) {
	w := make(map[string]bool)
	entries := strings.Split(os.Getenv("PROJECT_ALLOWLIST"), ",")
	for _, resource := range entries {
		r := strings.TrimSpace(resource)
		p := strings.SplitN(r, "/", 2)
		if len(p) != 2 {
			return nil, fmt.Errorf("invalid allowlist entry: %s", resource)
		}
		if !isAllowed(p[0], allowed) {
			return nil, fmt.Errorf("invalid resource type(%s), only types(%v) are supported in the allowlist", p[0], allowed)
		}
		w[r] = true
	}
	return w, nil
}

// isAllowed returns true if the string is present in the allowed list.
func isAllowed(candidate string, list []string) bool {
	for _, t := range list {
		if t == candidate {
			return true
		}
	}
	return false
}

// getPublicSigningCert obtains the public google signing certs and returns
// the cert (identified by kid) that is needed for JWT verification:
// https://cloud.google.com/compute/docs/instances/verifying-instance-identity
func getPublicSigningCert(ctx context.Context, kid string) (*x509.Certificate, error) {
	client := urlfetch.Client(ctx)
	resp, err := client.Get(signingCertsURL)
	if err != nil {
		return nil, fmt.Errorf("HTTP request for %q: %v", signingCertsURL, err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("response status != %v: %v", http.StatusOK, resp.Status)
	}
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	empty := []byte{10}
	if bytes.Equal(body, empty) {
		return nil, errors.New("received empty response")
	}
	if !json.Valid(body) {
		return nil, fmt.Errorf("response does not contain valid json, received %s", body)
	}
	var rawCerts map[string]string
	if err := json.Unmarshal(body, &rawCerts); err != nil {
		return nil, fmt.Errorf("json.Unmarshal returned %v", err)
	}

	signer, ok := rawCerts[kid]
	if !ok {
		return nil, fmt.Errorf("signing cert for %s is not available", kid)
	}
	// We're not interested in the remaining bytes after
	// the cert block, so ignore them.
	pem, _ := pem.Decode([]byte(signer))

	return x509.ParseCertificate(pem.Bytes)
}

// VMIDClaims contains the claims provided by a GCE VM ID JWT.
type VMIDClaims struct {
	jwt.Claims

	Google struct {
		ComputeEngine struct {
			ProjectID string `json:"project_id"`
		} `json:"compute_engine"`
	} `json:"google"`
}

var (
	errExpired  = errors.New("signing cert has expired")
	errKeyUsage = errors.New("invalid signing cert KeyUsage")
)

// checkSigningCert takes a GCE identity JWT document and verifies that
// it is signed by a public Google Certificate. It returns the claims
// contained within the JWT. It does not perform validation on the claim
// data, only that it comes from a trusted source.
// https://cloud.google.com/compute/docs/instances/verifying-instance-identity
func checkSigningCert(ctx context.Context, doc string) (*VMIDClaims, error) {
	token, err := jwt.ParseSigned(doc)
	if err != nil {
		return nil, fmt.Errorf("jwt.ParseSigned returned %v", err)
	}
	if len(token.Headers) != 1 {
		return nil, fmt.Errorf("signatures: got %d, want 1", len(token.Headers))
	}
	header := token.Headers[0]
	if header.Algorithm != "RS256" {
		return nil, fmt.Errorf("VMIdentity signature algorithm: got %s, want RS256", header.Algorithm)
	}
	// kid identifies which certificate signed the claims.
	kid := header.KeyID
	if kid == "" {
		return nil, errors.New("KeyID not present in VMIdentity token")
	}

	// Retrieve the signing certificate from googleapis.
	signer, err := getSigningCert(ctx, kid)
	if err != nil {
		return nil, fmt.Errorf("getSigningCert(%s): %v", kid, err)
	}

	// Per spec, check the signers key usage, public key and expiration
	if signer.KeyUsage != x509.KeyUsageDigitalSignature {
		return nil, fmt.Errorf("%w for key ID %s is %d, want %d", errKeyUsage, kid, signer.KeyUsage, x509.KeyUsageDigitalSignature)
	}
	if _, ok := signer.PublicKey.(*rsa.PublicKey); !ok {
		return nil, fmt.Errorf("signing cert for key ID %s does not contain an RSA public key", kid)
	}
	t := time.Now()
	if !t.After(signer.NotBefore) || !t.Before(signer.NotAfter) {
		return nil, fmt.Errorf("%w for key ID %s on %v", errExpired, kid, signer.NotAfter)
	}

	// Parse the claims. Signature verification occurs during claims
	// from a signed token.
	claims := VMIDClaims{}
	if err := token.Claims(signer.PublicKey, &claims); err != nil {
		return nil, err
	}

	return &claims, nil
}
