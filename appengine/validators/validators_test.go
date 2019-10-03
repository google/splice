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
	"context"
	"testing"

	"splice/appengine/server"
	"splice/models"
)

func TestBasicValidatorSuccess(t *testing.T) {
	test := struct {
		name string
		in   models.Request
		out  server.StatusCode
	}{
		"Valid Request",
		models.Request{Hostname: "Splice1234-W", ClientID: "1"},
		server.StatusSuccess,
	}

	validator := &Basic{}
	want := test.out

	if got, err := validator.Check(context.Background(), &test.in); err != nil || got != want {
		t.Errorf("test '%s'; got = %d, want = %d, err = %v", test.name, got, want, err)
	}
}

func TestBasicValidatorFailure(t *testing.T) {
	tests := []struct {
		name string
		in   models.Request
		out  server.StatusCode
	}{
		{
			"Empty Hostname",
			models.Request{ClientID: "2"},
			server.StatusRequestHostBlank,
		},
		{
			"Hostname Too Long",
			models.Request{Hostname: "Splice1343-w34346", ClientID: "3"},
			server.StatusRequestHostLength,
		},
		{
			"Empty ClientID Field",
			models.Request{Hostname: "Splice1343-w"},
			server.StatusRequestClientIDBlank,
		},
	}

	for _, tt := range tests {
		validator := &Basic{}
		want := tt.out
		if got, err := validator.Check(context.Background(), &tt.in); err == nil || got != want {
			t.Errorf("test '%s'; got = %d, want = %d, err = %v", tt.name, got, want, err)
		}
	}
}
