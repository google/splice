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

package certs

import (
	"crypto/x509"
	"testing"
	"time"
)

var (
	notBefore = time.Now().Add(-1 * time.Hour)
	notAfter  = time.Now().Add(time.Hour * 24 * 365 * 1)
)

func TestGenerateSelfSignedCert(t *testing.T) {
	// Test invalid cert common names
	invalidNames := []string{
		``,
		`.startingperiod`,
		`backs\ash`,
		`s/ashmark`,
		`c:lon`,
		`asteris*`,
		`question?mark`,
		`quotation"mark`,
		`less<than<sign`,
		`greater>than>sign`,
		`vertical|bar`,
		`waytolongforcomfort`,
	}
	c := Certificate{}
	for _, n := range invalidNames {
		if got := c.Generate(n, notBefore, notAfter); got == nil {
			t.Errorf("GenerateSelfSignedCert(%s) = %v, want err", n, got)
			continue
		}
	}

	// Test valid common names
	validNames := [...]string{`dummy.one`, `dummytwo`, `dummy-three`}
	for _, n := range validNames {
		if got := c.Generate(n, notBefore, notAfter); got != nil {
			t.Errorf("GenerateSelfSignedCert(%s) = %v, want nil", n, got)
			continue
		}
	}

	// Test self-signed certificate validity
	err := c.Generate("valid", notBefore, notAfter)
	if err != nil {
		t.Errorf("GenerateSelfSignedCert(valid) = %v", err)
	}

	cert, err := x509.ParseCertificate(c.Cert.Raw)
	if err != nil {
		t.Errorf("unable to parse the generated certificate: %v", err)
	}

	if cert.SerialNumber == nil {
		t.Error("self-signed certificate is missing serial number")
	}
}

func TestVerifyCert(t *testing.T) {
	base := "https://dummy.nowhere.com/"
	cn := "dummy"
	c := Certificate{}
	// Generate a self-signed cert in DER format to test with.
	err := c.Generate(cn, notBefore, notAfter)
	if err != nil {
		t.Errorf("GenerateSelfSignedCert(%s) = %v", cn, err)
	}

	// Test invalid cert
	if err := VerifyCert([]byte{1, 2, 3}, cn, base, "", "", "", true); err == nil {
		t.Errorf("VerifyCert(%s, %s, \"\", \"\", false) failed to catch a malformed cert", cn, base)
	}

	// test VerifyHostname
	failure := "failure"
	if err := VerifyCert(c.Cert.Raw, failure, base, "", "", "", true); err == nil {
		t.Errorf("VerifyCert(%s, %s, \"\", \"\", false) failed to catch a hostname mismatch", failure, base)
	}

	// Test bypass verification
	if err := VerifyCert(c.Cert.Raw, cn, base, "", "", "", false); err != nil {
		t.Errorf("VerifyCert(%s) failed to verify a match = %v", cn, err)
	}
}
