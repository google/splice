/*
Copyright 2018 Google LLC

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

package gce

import (
	"bytes"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
)

var (
	instanceID = []byte("example-host")
	projectID  = []byte("google.com:example")
	shortZone  = "us-bogus1-f"
	zone       = []byte("projects/123456/zones/" + shortZone)
	armadaID   = fmt.Sprintf("%s/%s/%s", shortZone, projectID, instanceID)
	identity   = []byte("these-arent-the-droids-youre-looking-for")

	emptyIdentity []byte
)

func fakeMetadataHandler(w http.ResponseWriter, r *http.Request) {
	switch r.URL.Path {
	case "/instance/id":
		fmt.Fprintln(w, string(instanceID))
	case "/project/project-id":
		fmt.Fprintln(w, string(projectID))
	case "/instance/zone":
		fmt.Fprintln(w, string(zone))
	case "/instance/service-accounts/default/identity":
		fmt.Fprintln(w, string(identity))
	default:
		fmt.Fprintf(w, "unknown query: %s\n", r.URL.Path)
	}
}

func TestMetadata(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(fakeMetadataHandler))
	defer ts.Close()

	// Test Successful Retrieval without VMIdentity
	metadataRoot = ts.URL
	m := Metadata{}
	if err := m.Read(); err != nil {
		t.Errorf("m.Read() = %v", err)
	}
	if got, want := m.InstanceID, instanceID; !bytes.Equal(got, want) {
		t.Errorf("m.InstanceID = %s, want %s", got, want)
	}
	if got, want := m.ProjectID, projectID; !bytes.Equal(got, want) {
		t.Errorf("m.ProjectID = %s, want %s", got, want)
	}
	if got, want := m.Zone, zone; !bytes.Equal(got, want) {
		t.Errorf("m.Zone = %s, want %s", got, want)
	}
	if got, want := m.ShortZone(), shortZone; got != want {
		t.Errorf("m.ShortZone() = %v, want %v", got, want)
	}
	if got, want := m.UniqueID(), armadaID; got != want {
		t.Errorf("m.UniqueID() = %v, want %v", got, want)
	}
	if got, want := m.Identity, emptyIdentity; !bytes.Equal(got, want) {
		t.Errorf("m.Identity = %s, want %s", got, want)
	}

	// Test Retrieval of VMidentity
	v := Metadata{Audience: "https://splice-test-endpoint/request-unattended"}
	if err := v.Read(); err != nil {
		t.Errorf("v.Read() = %v", err)
	}
	if got, want := v.Identity, identity; !bytes.Equal(got, want) {
		t.Errorf("v.Identity = %s, want %s", got, want)
	}
}

func TestMetadataFailure(t *testing.T) {
	// nil Handler returns 404 for all requests
	ts := httptest.NewServer(nil)
	defer ts.Close()

	metadataRoot = ts.URL
	m := Metadata{}
	if err := m.Read(); err == nil {
		t.Error("m.Read() = nil, want error")
	}
}
