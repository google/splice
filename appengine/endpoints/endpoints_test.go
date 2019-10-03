/*
Copyright 2018 Google Inc.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

	http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package endpoints

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"google.golang.org/appengine/aetest"
	"google.golang.org/appengine"
	"splice/appengine/server"
	"splice/appengine/validators"
	"splice/models"
)

const (
	// certHash is the Sha256 Hash of the Raw bytes of a dummy
	// certificate. It is equivalent of calculating the hash on
	// Cert.Raw for an x509.Certificate and removing the trailing
	// equal sign.
	certHash = `T7Da+FmlTXTQSEr+XT3kvA9NEEFOqKyVVcAH4Khqf8A`
)

func initInstance(t *testing.T) (aetest.Instance, error) {
	// Although the Datastore is eventually consistent on production, we need
	// it to be strongly consistent during testing.
	inst, err := aetest.NewInstance(&aetest.Options{StronglyConsistentDatastore: true})
	if err != nil {
		return nil, fmt.Errorf("NewInstance: %v", err)
	}
	return inst, nil
}

func newRequest(t *testing.T, inst aetest.Instance, method, url string, body io.Reader) (*http.Request, error) {
	r, err := inst.NewRequest(method, url, body)
	if err != nil {
		return nil, fmt.Errorf("NewRequest: %s, %s, error %v", method, url, err)
	}
	return r, nil
}

func TestProcessRequest(t *testing.T) {
	// Replace the standard validators with a basic validator
	// for testing.
	validatorsNewAttended = validators.New
	useDatastore = false
	usePubsub = false

	tests := []struct {
		desc string
		in   models.Request
		out  models.Response
	}{
		{
			"valid request",
			models.Request{Hostname: "Splice1234-W", ClientID: "1"},
			models.Response{ErrorCode: server.StatusSuccess, Status: "OK"},
		},
		{
			"empty Hostname field",
			models.Request{ClientID: "2"},
			models.Response{ErrorCode: server.StatusRequestHostBlank, Status: "Hostname is blank"},
		},
		{
			"hostname too long",
			models.Request{Hostname: "Splice1343-w34346", ClientID: "3"},
			models.Response{ErrorCode: server.StatusRequestHostLength, Status: "Hostname longer than 15 characters"},
		},
		{
			"empty ClientID field",
			models.Request{Hostname: "Splice1343-w"},
			models.Response{ErrorCode: server.StatusRequestClientIDBlank, Status: "ClientID is blank"},
		},
	}

	method := "POST"
	uri := "/request"
	inst, err := initInstance(t)
	if err != nil {
		t.Fatalf("initInstance(): %v", err)
	}
	defer inst.Close()

	for _, tt := range tests {
		jsonRequest, err := json.Marshal(tt.in)
		if err != nil {
			t.Errorf("%s, json.Marshal(%v) returned %v", tt.desc, tt.in, err)
			continue
		}
		req, err := newRequest(t, inst, method, uri, bytes.NewReader(jsonRequest))
		if err != nil {
			t.Errorf("%s, newRequest returned %v while processing %v", tt.desc, err, jsonRequest)
			continue
		}

		if err := os.Setenv("VERIFY_CERT", "false"); err != nil {
			t.Fatalf("os.Setenv = %v", err)
		}

		rr := httptest.NewRecorder()
		handler := &AttendedRequestHandler{}
		handler.ServeHTTP(rr, req)

		raw, err := ioutil.ReadAll(rr.Body)
		if err != nil {
			t.Errorf("%s, ioutil.ReadAll(%v) for response body returned %v", tt.desc, rr.Body, err)
		}

		var sResp models.Response
		if err := json.Unmarshal(raw, &sResp); err != nil {
			t.Errorf("%s, json.Unmarshal(%s) returned %v", tt.desc, raw, err)
		}

		if sResp.ErrorCode != tt.out.ErrorCode {
			t.Errorf("%s; got %d %v, want %d %v",
				tt.desc, sResp.ErrorCode, sResp.Status, tt.out.ErrorCode, tt.out.Status)
		}
	}
}

func TestGenerateReqID(t *testing.T) {
	invalidLength := uint(0)
	if _, got := generateReqID(invalidLength); got == nil {
		t.Errorf("generateReqID(%d) = <nil>, want: err", invalidLength)
	}

	validLengths := []uint{32, 64, 128, 256}
	for _, tlen := range validLengths {
		if _, got := generateReqID(tlen); got != nil {
			t.Errorf("generateReqID(%d) = %v, want: <nil>", got, tlen)
		}
	}
}

func TestProcessResult(t *testing.T) {
	useDatastore = false

	tests := []struct {
		desc string
		in   models.StatusQuery
		out  models.Response
	}{
		{
			"valid request",
			models.StatusQuery{RequestID: "12345", ClientID: "1"},
			models.Response{ErrorCode: server.StatusSuccess},
		},
		{
			"empty RequestID",
			models.StatusQuery{ClientID: "2"},
			models.Response{ErrorCode: server.StatusReqProcessingError},
		},
		{
			"empty ClientID",
			models.StatusQuery{RequestID: "12345"},
			models.Response{ErrorCode: server.StatusReqProcessingError},
		},
	}

	method := "POST"
	uri := "/result"
	inst, err := initInstance(t)
	if err != nil {
		t.Fatalf("initInstance(): %v", err)
	}
	defer inst.Close()

	for _, tt := range tests {
		jsonRequest, err := json.Marshal(tt.in)
		if err != nil {
			t.Errorf("%s, json.Marshal(%v) returned %v", tt.desc, tt.in, err)
			continue
		}
		req, err := newRequest(t, inst, method, uri, bytes.NewReader(jsonRequest))
		if err != nil {
			t.Errorf("%s, newRequest returned %v while processing %v", tt.desc, err, jsonRequest)
			continue
		}

		rr := httptest.NewRecorder()
		handler := ResultHandler(ProcessResult)
		handler.ServeHTTP(rr, req)

		raw, err := ioutil.ReadAll(rr.Body)
		if err != nil {
			t.Errorf("%s, ioutil.ReadAll(%v) for response body returned %v", tt.desc, rr.Body, err)
		}

		var resp models.Response
		if err := json.Unmarshal(raw, &resp); err != nil {
			t.Errorf("%s, json.Unmarshal(%s) returned %v", tt.desc, raw, err)
		}

		if resp.ErrorCode != tt.out.ErrorCode {
			t.Errorf("%s; ProcessResult = %d, want %d",
				tt.desc, resp.ErrorCode, tt.out.ErrorCode)
		}
	}
}

func TestVerifyCert(t *testing.T) {
	inst, err := initInstance(t)
	if err != nil {
		t.Fatalf("initInstance(): %v", err)
	}
	defer inst.Close()
	r, err := newRequest(t, inst, "", "", bytes.NewReader([]byte("bogus")))
	if err != nil {
		t.Fatalf("newRequest = %v", err)
	}
	ctx := appengine.NewContext(r)

	// Enable cert verification, because we intend to explicitly test it here.
	if err := os.Setenv("VERIFY_CERT", "true"); err != nil {
		t.Fatalf("os.Setenv = %v", err)
	}
	header := "header_fp"
	if err := os.Setenv("VERIFY_CERT_HEADER", header); err != nil {
		t.Fatalf("os.Setenv = %v", err)
	}

	// Test a Valid Fingerprint
	r.Header.Add(header, certHash)
	if err := verifyCert(ctx, certHash, r); err != nil {
		t.Errorf("Valid fingerprint: verifyCert = %v, want nil", err)
	}

	// Test an invalid Fingerprint
	r.Header.Set(header, "mismatch")
	if err := verifyCert(ctx, certHash, r); err == nil {
		t.Errorf("Invalid fingerprint: verifyCert = %v, want err", err)
	}

	// Test a missing fingerprint
	r.Header.Set(header, "")
	if err := verifyCert(ctx, certHash, r); err == nil {
		t.Errorf("Missing fingerprint: verifyCert = %v, want err", err)
	}

	// Test that disabling via env variable is functional
	if err := os.Setenv("VERIFY_CERT", "false"); err != nil {
		t.Fatalf("os.Setenv = %v", err)
	}
	if err := verifyCert(ctx, "", nil); err != nil {
		t.Errorf("Env variable is false: verifyCert = %v, want nil", err)
	}
}
