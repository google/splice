// Copyright 2021 Google LLC
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
	"os"

	"google.golang.org/appengine/log"
	"github.com/google/splice/appengine/server"
	"github.com/google/splice/models"
)

// Reuse implements validators.Validator and checks if the request is permitted to enable name reuse.
type Reuse struct{}

// NewReuse returns a reuse validator.
func NewReuse() *Reuse {
	return &Reuse{}
}

func (r Reuse) Check(ctx context.Context, req *models.Request) (server.StatusCode, error) {
	req.AttemptReuse = false

	// Determine if reuse is permissible in this environment.
	if os.Getenv("REJOIN_ALLOWED") == "true" {
		log.Infof(ctx, "Rejoin allowed; AttemptReuse enabled on request %s", req.RequestID)
		req.AttemptReuse = true
	}

	return server.StatusSuccess, nil
}
