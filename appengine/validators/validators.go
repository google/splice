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

// Package validators provides basic validation for splice requests and
// exposes an interface for additional validators.
package validators

import (
	"context"
	"errors"
	"fmt"

	"splice/appengine/server"
	"splice/models"
)

// Validator performs metadata checking for requests.
type Validator interface {
	// Check returns a status code and an error if the check failed. Check
	// should perform its own cleanup (defer or otherwise) prior to returning.
	Check(context.Context, *models.Request) (server.StatusCode, error)
}

// Basic implements Validator and performs basic checking of a request.
type Basic struct {
}

// Check returns StatusSuccess(0) if a request has data in required fields and
// that data passes basic integrity checks.
func (d Basic) Check(ctx context.Context, req *models.Request) (server.StatusCode, error) {
	switch {
	case req.Hostname == "":
		return server.StatusRequestHostBlank, errors.New("hostname is blank")

	case len(req.Hostname) > 15:
		return server.StatusRequestHostLength, fmt.Errorf("hostname %s too long (got: %d, want: <15)", req.Hostname, len(req.Hostname))

	case len(req.ClientID) == 0:
		return server.StatusRequestClientIDBlank, errors.New("clientID is blank")
	}
	return server.StatusSuccess, nil
}

// New returns a slice containing all basic validators for
// interactive requests.
func New() ([]Validator, error) {
	return []Validator{Basic{}}, nil
}

// NewUnattended returns a slice containing all validators required
// for unattended requests.
func NewUnattended() ([]Validator, error) {
	v, err := New()
	if err != nil {
		return nil, fmt.Errorf("New() returned: %v", err)
	}

	g, err := NewGCE(nil)
	if err != nil {
		return nil, fmt.Errorf("NewGCE() returned %v", err)
	}
	return append(v, g), nil
}
