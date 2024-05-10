// Copyright 2022 Google LLC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package validators

import (
	"golang.org/x/net/context"

	"google.golang.org/appengine/v2/log"
	"github.com/google/splice/appengine/server"
	"github.com/google/splice/models"
)

var (
	allowedGenerators = []string{"prefix"}
)

// GenericGeneratorChecks provides general sanity checks for SpliceD generators
type GenericGeneratorChecks struct{}

// Check returns StatusSuccess if req is valid. It may modify req to sanitize it.
func (c GenericGeneratorChecks) Check(ctx context.Context, req *models.Request) (server.StatusCode, error) {
	if req.GeneratorID == "" {
		return server.StatusSuccess, nil
	}

	// Generator users shouldn't be providing a hostname.
	if req.Hostname != "" {
		log.Warningf(ctx, "Request %v provided both a Hostname and GeneratorID.", req.RequestID)
		return server.StatusRequestGeneratorError, nil
	}

	// Make sure a known generator was requested.
	for _, g := range allowedGenerators {
		if g == req.GeneratorID {
			return server.StatusSuccess, nil
		}
	}

	return server.StatusRequestGeneratorError, nil
}

// PrefixGeneratorCheck provides sanity checks for the SpliceD "prefix" Generator. It may modify
// the request to sanitize certain inputs for compatibility.
type PrefixGeneratorCheck struct{}

// Check returns StatusSuccess if req is valid. It may modify req to sanitize it.
func (c PrefixGeneratorCheck) Check(ctx context.Context, req *models.Request) (server.StatusCode, error) {
	if req.GeneratorID != "prefix" {
		return server.StatusSuccess, nil
	}

	// Prefix generator is prone to name collisions. Force disable reuse so in-use names aren't hijacked inadvertently.
	if req.AttemptReuse == true {
		log.Warningf(ctx, "Request %v was attempting reuse with the Prefix generator. Disabling.", req.RequestID)
		req.AttemptReuse = false
	}

	// Prefix generator doesn't use input data.
	if req.GeneratorData != nil {
		log.Warningf(ctx, "Request %v was passing unexpected input with the Prefix generator. Removing.", req.RequestID)
		req.GeneratorData = nil
	}

	return server.StatusSuccess, nil
}
