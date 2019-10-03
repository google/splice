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

// Package gce provides functionality for reading GCE instance metadata.
package gce

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"strings"
)

var (
	metadataRoot = "http://metadata/computeMetadata/v1"
)

func readGceMetadata(path string) ([]byte, error) {
	hreq, _ := http.NewRequest("GET", fmt.Sprintf("%s/%s", metadataRoot, path), nil)
	hreq.Header.Add("Metadata-Flavor", "Google")

	resp, err := http.DefaultClient.Do(hreq)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("received non-200 HTTP response: %s", resp.Status)
	}

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	return bytes.Trim(body, " \n"), nil
}

// Metadata holds GCE instance metadata.
type Metadata struct {
	InstanceID []byte
	ProjectID  []byte
	Zone       []byte

	// Unattended joins
	Audience string
	Identity []byte
}

// ShortZone returns the short version of the GCE Zone.
//
// The metadata zone value is of the form 'projects/324016238535/zones/us-west1-a';
// however, only the last part, i.e. us-west1-a, is required for Splice (b/34588301).
func (m *Metadata) ShortZone() string {
	s := strings.Split(string(m.Zone), "/")
	return s[len(s)-1]
}

// UniqueID constructs the unique version of the GCE ID, consisting of the short
// zone name, project ID and instance ID, all separated by '/' characters.
func (m *Metadata) UniqueID() string {
	return fmt.Sprintf("%s/%s/%s", m.ShortZone(), m.ProjectID, m.InstanceID)
}

// Read loads metadata from the instance.
func (m *Metadata) Read() error {
	var err error

	m.InstanceID, err = readGceMetadata("instance/id")
	if err != nil {
		return fmt.Errorf("could not retrieve GCE instance ID: %v", err)
	}

	m.ProjectID, err = readGceMetadata("project/project-id")
	if err != nil {
		return fmt.Errorf("could not retrieve GCE project ID: %v", err)
	}

	m.Zone, err = readGceMetadata("instance/zone")
	if err != nil {
		return fmt.Errorf("could not retrieve GCE zone: %v", err)
	}

	// VMIdentity isn't always necessary. Populate it only if
	// an audience was provided.
	if m.Audience != "" {
		p := fmt.Sprintf("instance/service-accounts/default/identity?audience=%s&format=full", url.QueryEscape(m.Audience))
		m.Identity, err = readGceMetadata(p)
		if err != nil {
			return fmt.Errorf("could not retrieve GCE identity document: %v", err)
		}
	}

	return nil
}
