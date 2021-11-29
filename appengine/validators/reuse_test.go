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
	"os"
	"strconv"
	"testing"

	"github.com/google/splice/models"
)

func TestReuse(t *testing.T) {
	tests := []struct {
		name    string
		enabled bool
	}{
		{"Reuse Allowed", true},
		{"Reuse Disabled", false},
	}
	ctx, err := fakeContext()
	if err != nil {
		t.Fatalf("fakeContext: %v", err)
	}
	for _, tt := range tests {
		req := models.Request{}
		if err := os.Setenv("REJOIN_ALLOWED", strconv.FormatBool(tt.enabled)); err != nil {
			t.Errorf("os.Setenv(%s, %t) = %v, want nil", "REJOIN_ALLOWED", tt.enabled, err)
		}
		validator := NewReuse()
		if _, err := validator.Check(ctx, &req); req.AttemptReuse != tt.enabled && err == nil {
			t.Errorf("test %q: got %t, want %t", tt.name, req.AttemptReuse, tt.enabled)
		}
	}
}
