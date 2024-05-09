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

package testing

import (
	"errors"
	"testing"

	"github.com/google/go-cmp/cmp"
)

func TestJoinWithReuse(t *testing.T) {
	nid := NewInactiveDirectory()
	tests := []struct {
		desc string
		name string
		out  []byte
		err  error
	}{
		{"successful join",
			"host1",
			SuccessBlob,
			nil,
		},
		{"successful rejoin",
			"host1",
			SuccessBlob,
			nil,
		},
	}
	for _, tt := range tests {
		out, err := nid.Join(tt.name, "domain.example.com", true)
		if diff := cmp.Diff(out, tt.out); diff != "" {
			t.Errorf("%s produced unexpected diff: %v", tt.desc, diff)
		}
		if !errors.Is(err, tt.err) {
			t.Errorf("%s produced unexpected error: got %v, want %v", tt.desc, err, tt.err)
		}
	}
}

func TestJoinWithoutReuse(t *testing.T) {
	nid := NewInactiveDirectory()
	tests := []struct {
		desc string
		name string
		out  []byte
		err  error
	}{
		{"successful join",
			"host1",
			SuccessBlob,
			nil,
		},
		{"unsuccessful rejoin",
			"host1",
			nil,
			ErrReuse,
		},
	}
	for _, tt := range tests {
		out, err := nid.Join(tt.name, "domain.example.com", false)
		if diff := cmp.Diff(out, tt.out); diff != "" {
			t.Errorf("%s produced unexpected diff: %v", tt.desc, diff)
		}
		if !errors.Is(err, tt.err) {
			t.Errorf("%s produced unexpected error: got %v, want %v", tt.desc, err, tt.err)
		}
	}
}
