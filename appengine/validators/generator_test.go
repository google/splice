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
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/google/splice/appengine/server"
	"github.com/google/splice/models"
)

func TestGenericGeneratorCheck(t *testing.T) {
	tests := []struct {
		name string
		req  *models.Request
		out  server.StatusCode
	}{
		{"No Generator",
			&models.Request{Hostname: "test"},
			server.StatusSuccess,
		},
		{"Invalid Generator",
			&models.Request{GeneratorID: "invalid"},
			server.StatusRequestGeneratorError,
		},
		{"Hostname Plus Generator",
			&models.Request{GeneratorID: "prefix", Hostname: "test"},
			server.StatusRequestGeneratorError,
		},
	}
	ctx, err := fakeContext()
	if err != nil {
		t.Fatalf("fakeContext: %v", err)
	}
	validator := GenericGeneratorChecks{}
	for _, tt := range tests {
		out, _ := validator.Check(ctx, tt.req)
		if out != tt.out {
			t.Errorf("%s produced unexpected status code: got %d, want %d", tt.name, out, tt.out)
		}
	}
}

func TestPrefixGeneratorCheck(t *testing.T) {
	tests := []struct {
		name string
		req  *models.Request
		out  *models.Request
	}{
		{"No Generator",
			&models.Request{Hostname: "test"},
			&models.Request{Hostname: "test"},
		},
		{"Different Generator",
			&models.Request{GeneratorID: "other"},
			&models.Request{GeneratorID: "other"},
		},
		{"Reuse Enabled",
			&models.Request{GeneratorID: "prefix", AttemptReuse: true},
			&models.Request{GeneratorID: "prefix", AttemptReuse: false}},
		{"Data Present",
			&models.Request{GeneratorID: "prefix", GeneratorData: []byte("unexpected")},
			&models.Request{GeneratorID: "prefix", GeneratorData: nil}},
	}
	ctx, err := fakeContext()
	if err != nil {
		t.Fatalf("fakeContext: %v", err)
	}
	validator := PrefixGeneratorCheck{}
	for _, tt := range tests {
		validator.Check(ctx, tt.req)
		if diff := cmp.Diff(tt.req, tt.out); diff != "" {
			t.Errorf("%s produced unexpected diff: %v", tt.name, diff)
		}
	}
}
