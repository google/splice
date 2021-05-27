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
	"crypto"
	"crypto/sha256"
	"crypto/x509"
	"encoding/hex"
	"encoding/json"
	"encoding/pem"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"google.golang.org/appengine/aetest"
	"google.golang.org/appengine"
	"github.com/dgrijalva/jwt-go"
	"github.com/google/splice/appengine/server"
	"github.com/google/splice/cli/gce"
	"github.com/google/splice/models"
	"github.com/google/splice/shared/certs"
)

const (
	emptyAllowlist    = ""
	invalidAllowlist  = "projects:foobar"
	folderInAllowlist = "projects/foobar, folder/foobaz"
	validAllowlist    = "projects/foobar, projects/foobaz, projects/123456"
	fakeAudience      = "https://fake-splice.google.com/request-unattended"
)

var (
	testCertJSON    = new(bytes.Buffer)
	testSigningCert []byte

	certKeyUsage  = x509.KeyUsageDigitalSignature
	validBefore   = time.Now().Add(-1 * time.Hour)
	validAfter    = time.Now().Add(1 * 365 * 24 * time.Hour)
	expiredBefore = time.Now().Add(-1 * 365 * 24 * time.Hour)
	expiredAfter  = time.Now().Add(-1 * time.Hour)
)

func TestGCEValidatorSuccess(t *testing.T) {
	if err := os.Setenv("PROJECT_ALLOWLIST", validAllowlist); err != nil {
		t.Errorf("os.Setenv(%q, %q) = %v, want nil", "PROJECT_ALLOWLIST", validAllowlist, err)
	}
	ctx, err := fakeContext()
	if err != nil {
		t.Fatalf("fakeContext: %v", err)
	}

	getSigningCert = getTestSigningCert
	// Signing cert for fake GCE VM Identities
	der, priv, err := certs.GenerateSelfSignedCert("fakehost", validBefore, validAfter)
	if err != nil {
		t.Fatalf("generating a signing cert returned %v", err)
	}
	testSigningCert = der
	// A valid test GCE VM Identity Doc
	doc, err := fakeGCEIDDoc(priv, "https:///request-unattended", expectedIssuer, "us-bogus1-a", "foobar", 1234, time.Now().Unix(), jwt.SigningMethodRS256) //NOTYPO
	if err != nil {
		t.Errorf("fakeGCEIDDoc returned %v", err)
	}

	tests := []struct {
		name string
		in   models.Request
		out  server.StatusCode
	}{
		{
			"Allowlist Project",
			models.Request{
				Hostname:    "Splice1234-W",
				GCEMetadata: gce.Metadata{ProjectID: []byte("foobar"), Identity: []byte(doc)},
			},
			server.StatusSuccess,
		},
	}

	for _, tt := range tests {
		validator, err := NewGCE(nil)
		if err != nil {
			t.Errorf("NewGCE() returned %v", err)
		}

		req := &tt.in
		want := tt.out
		if err := os.Setenv("REJOIN_ALLOWED", "true"); err != nil {
			t.Errorf("os.Setenv(%q, %q) = %v, want nil", "REJOIN_ALLOWED", "true", err)
		}

		if got, err := validator.Check(ctx, req); err != nil || got != want {
			t.Errorf("test %q; got = %d, want = %d, err = %v", tt.name, got, want, err)
		}
		if req.AttemptReuse != true {
			t.Errorf("AttemptReuse returned %t for test %q, want %t", req.AttemptReuse, tt.name, true)
		}
	}
}

func TestGCEValidatorFailure(t *testing.T) {
	ctx, err := fakeContext()
	if err != nil {
		t.Fatalf("fakeContext: %v", err)
	}

	tests := []struct {
		name      string
		allowlist string
		in        models.Request
		out       server.StatusCode
	}{
		{
			"Non-Allowlist Project",
			validAllowlist,
			models.Request{
				Hostname:    "Splice1234-W",
				GCEMetadata: gce.Metadata{ProjectID: []byte("mchammer")},
			},
			server.StatusInvalidGCEmeta,
		},
		{
			"Non-Allowlist Folder",
			validAllowlist,
			models.Request{
				Hostname:    "Splice1234-W",
				GCEMetadata: gce.Metadata{ProjectID: []byte("99999")},
			},
			server.StatusInvalidGCEmeta,
		},
	}

	for _, tt := range tests {
		if err := os.Setenv("PROJECT_ALLOWLIST", tt.allowlist); err != nil {
			t.Errorf("os.Setenv(%q, %q) = %v, want nil", "PROJECT_ALLOWLIST", tt.allowlist, err)
		}
		validator, err := NewGCE(nil)
		if err != nil {
			t.Errorf("NewGCE() returned %v", err)
		}

		req := &tt.in
		want := tt.out

		if err := os.Setenv("REJOIN_ALLOWED", "false"); err != nil {
			t.Errorf("os.Setenv(%q, %q) = %v, want nil", "REJOIN_ALLOWED", "false", err)
		}

		if got, err := validator.Check(ctx, req); err == nil || got != want {
			t.Errorf("test %q; got = %d, want = %d, err = %v", tt.name, got, want, err)
		}
		if req.AttemptReuse != false {
			t.Errorf("AttemptReuse returned %t for test %q, want %t", req.AttemptReuse, tt.name, false)
		}
	}
}

func TestGCENewFailure(t *testing.T) {
	tests := []struct {
		name      string
		allowlist string
		want      string
	}{
		{"Invalid Format", invalidAllowlist, "invalid allowlist entry"},
		{"Folder in Allowlist", folderInAllowlist, "invalid resource type"},
		{"Empty Allowlist", emptyAllowlist, ""},
	}

	for _, tt := range tests {
		if err := os.Setenv("PROJECT_ALLOWLIST", tt.allowlist); err != nil {
			t.Errorf("os.Setenv(%q, %q) = %v, want nil", "PROJECT_ALLOWLIST", tt.allowlist, err)
		}
		if _, got := NewGCE(nil); !strings.Contains(got.Error(), tt.want) {
			t.Errorf("test %q; got %s, want %s", tt.name, got.Error(), tt.want)
		}
	}
}

func TestGCENewSuccess(t *testing.T) {
	tests := []struct {
		name      string
		allowlist string
		want      int
	}{
		{"Valid Allowlist", validAllowlist, 3},
		{"Empty Allowlist", emptyAllowlist, 0},
	}

	for _, tt := range tests {
		if err := os.Setenv("PROJECT_ALLOWLIST", tt.allowlist); err != nil {
			t.Errorf("os.Setenv(%q, %q) = %v, want nil", "PROJECT_ALLOWLIST", tt.allowlist, err)
		}
		if got, err := NewGCE(nil); len(got.ProjectAllowlist) != tt.want && err == nil {
			t.Errorf("test %q: got %d, want %d", tt.name, len(got.ProjectAllowlist), tt.want)
		}
	}
}

func TestGetSigningCert(t *testing.T) {
	ctx, err := fakeContext()
	if err != nil {
		t.Fatalf("fakeContext: %v", err)
	}
	ts := httptest.NewServer(http.HandlerFunc(fakeCertificateHandler))
	defer ts.Close()

	testCertMap := make(map[string]string)
	testCertMap[testKid] = testCert
	if err := json.NewEncoder(testCertJSON).Encode(testCertMap); err != nil {
		t.Errorf("json.NewEncoder returned %v", err)
	}

	// Test successful retrieval
	getSigningCert = getPublicSigningCert
	testCertPEM, _ := pem.Decode([]byte(testCert))
	validCert, err := x509.ParseCertificate(testCertPEM.Bytes)
	if err != nil {
		t.Errorf("x509.ParseCertificate returned %v", err)
	}
	signingCertsURL = ts.URL + "/oauth2/v1/certs"
	if got, err := getSigningCert(ctx, testKid); err != nil || !got.Equal(validCert) {
		t.Errorf("test 'Successful Retrieval'; got = %v, want cert, err = %v", got, err)
	}

	// Test failure modes
	tests := []struct {
		name string
		url  string
		kid  string
		want string
	}{
		{"Invalid JSON", ts.URL + "/oauth2/v1/invalid", testKid, "does not contain valid"},
		{"Empty Response", ts.URL + "/oauth2/v1/empty", testKid, "empty response"},
		{"Missing Kid", ts.URL + "/oauth2/v1/certs", "missing", "not available"},
	}

	for _, tt := range tests {
		signingCertsURL = tt.url
		if _, got := getSigningCert(ctx, tt.kid); !strings.Contains(got.Error(), tt.want) {
			t.Errorf("test %q: got %s, want %s", tt.name, got, tt.want)
		}
	}
}

func TestCheckSigningCert(t *testing.T) {
	ctx, err := fakeContext()
	if err != nil {
		t.Fatalf("fakeContext: %v", err)
	}
	getSigningCert = getTestSigningCert

	// Signing cert for token tests
	der, priv, err := certs.GenerateSelfSignedCert("fakehost", validBefore, validAfter)
	if err != nil {
		t.Fatalf("generating a signing cert returned %v", err)
	}
	testSigningCert = der

	// Invalid Token Tests
	tokenTests := []struct {
		name      string
		doc       string
		method    *jwt.SigningMethodRSA
		headerKey string
		headerVal string
		want      string
	}{
		{"Invalid Token", "invalid", nil, "", "", "ParseSigned returned"},
		{"Invalid Signing Method", "", jwt.SigningMethodRS512, "", "", "algorithm"},
		{"Missing Key ID", "", jwt.SigningMethodRS256, "foo", "bar", "KeyID not present"},
	}

	for _, tt := range tokenTests {
		if tt.doc == "" {
			token := jwt.New(tt.method)
			token.Header[tt.headerKey] = tt.headerVal
			tt.doc, err = token.SignedString(priv)
			if err != nil {
				t.Errorf("token.SignedString returned %v", err)
			}
		}
		if _, got := checkSigningCert(ctx, tt.doc); !strings.Contains(got.Error(), tt.want) {
			t.Errorf("test '%s': got %s, want %s", tt.name, got, tt.want)
		}
	}

	// Cert Validation Tests
	certTests := []struct {
		name   string
		before time.Time
		after  time.Time
		usage  x509.KeyUsage
		want   error
	}{
		{"Valid Token and Cert", validBefore, validAfter, x509.KeyUsageDigitalSignature, nil},
		{"Invalid Cert KeyUsage", validBefore, validAfter, x509.KeyUsageCertSign, errors.New("KeyUsage")},
		{"Expired Cert", expiredBefore, expiredAfter, x509.KeyUsageDigitalSignature, errors.New("expired")},
	}

	for _, ct := range certTests {
		der, priv, err := certs.GenerateSelfSignedCert("fakehost", ct.before, ct.after)
		if err != nil {
			t.Fatalf("generating a signing cert returned %v", err)
		}
		testSigningCert = der
		certKeyUsage = ct.usage

		doc, err := fakeGCEIDDoc(priv, fakeAudience, expectedIssuer, "us-bogus1-a", "test.com:foo", 1234, time.Now().Unix(), jwt.SigningMethodRS256)
		if err != nil {
			t.Errorf("fakeGCEIDDoc returned %v", err)
		}

		if _, got := checkSigningCert(ctx, doc); got != ct.want {
			if !strings.Contains(got.Error(), ct.want.Error()) {
				t.Errorf("test %q: got %v, want %v", ct.name, got, ct.want)
			}
		}
	}
}

// fakeContext is primarily used to provide an appengine Context for test purposes.
func fakeContext() (context.Context, error) {
	inst, err := aetest.NewInstance(nil)
	if err != nil {
		return nil, fmt.Errorf("aetest.NewInstance: %v", err)
	}
	r, err := inst.NewRequest("POST", "/bogus", bytes.NewReader([]byte("test")))
	if err != nil {
		return nil, fmt.Errorf("inst.NewRequest: %v", err)
	}
	return appengine.NewContext(r), nil
}

func fakeCertificateHandler(w http.ResponseWriter, r *http.Request) {
	switch r.URL.Path {
	case "/oauth2/v1/certs":
		fmt.Fprintln(w, testCertJSON)
	case "/oauth2/v1/invalid":
		fmt.Fprintln(w, "invalid json")
	case "/oauth2/v1/empty":
		fmt.Fprintln(w, "")
	default:
		fmt.Fprintf(w, "unknown query: %s\n", r.URL.Path)
	}
}

// getTestSigningCert always returns the same signing cert,
// as all test docs are signed with the same test cert. It
// modifies the certificate KeyUsage as needed.
func getTestSigningCert(ctx context.Context, kid string) (*x509.Certificate, error) {
	cert, err := x509.ParseCertificate(testSigningCert)
	if err != nil {
		return nil, err
	}
	// Add the desired KeyUsage to the generic self-signed cert.
	cert.KeyUsage = certKeyUsage
	return cert, nil
}

// fakeGCEIDDoc creates a signed GCE VM identity document with the provided data and signer.
func fakeGCEIDDoc(signer crypto.Signer, aud, iss, zone, projectID string, instanceID, iat int64, signMethod *jwt.SigningMethodRSA) (string, error) {
	claims := jwt.MapClaims{}
	if aud != "" {
		claims["aud"] = aud
	}
	if iss != "" {
		claims["iss"] = iss
	}
	if iat != -1 {
		claims["iat"] = iat
	}

	entries := map[string]interface{}{}
	if zone != "" {
		entries["zone"] = zone
	}
	if projectID != "" {
		entries["project_id"] = projectID
	}
	if instanceID != -1 {
		entries["instance_id"] = instanceID
	}

	// Fill in the rest with dummy information.
	entries["project_number"] = -1
	entries["instance_name"] = "dummy instanceName"
	entries["creation_timestamp"] = -1

	// Put in the google.computeEngine object.
	claims["google"] = map[string]map[string]interface{}{
		"compute_engine": entries,
	}

	// Add the 'kid' (Key ID) mapping, which is the sha256 hash of the DER-encoded PKIX public key.
	der, err := x509.MarshalPKIXPublicKey(signer.Public())
	if err != nil {
		return "", err
	}
	shaBytes := sha256.Sum256(der)
	token := jwt.NewWithClaims(signMethod, claims)
	token.Header["kid"] = hex.EncodeToString(shaBytes[:])

	return token.SignedString(signer)
}
