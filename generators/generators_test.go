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

package generators

import (
	"fmt"
	"testing"

	"github.com/pkg/errors"
)

type fakeGen struct {
	configErr  error
	configured bool
	genErr     error
}

func (f *fakeGen) Configure() error {
	f.configured = true
	fmt.Println("configured!")
	return f.configErr
}

func (f *fakeGen) Generate(in []byte) (string, error) {
	return "", f.genErr
}

func TestConfigureAll(t *testing.T) {
	tests := []struct {
		name string
		g    *fakeGen
	}{
		{"gen_a", &fakeGen{}},
		{"gen_b", &fakeGen{}},
	}
	for _, tt := range tests {
		Register(tt.name, tt.g)
	}
	err := ConfigureAll()
	if err != nil {
		t.Fatalf("ConfigureAll() returned unexpected error: %v", err)
	}
	for _, tt := range tests {
		if tt.g.configured != true {
			t.Errorf("ConfigureAll() failed to configure generator %s", tt.name)
		}
	}
}

func TestConfigureAllError(t *testing.T) {
	want := errors.New("config error")
	tests := []struct {
		name string
		g    *fakeGen
	}{
		{"gen_a", &fakeGen{}},
		{"gen_b", &fakeGen{configErr: want}},
		{"gen_c", &fakeGen{}},
	}
	for _, tt := range tests {
		Register(tt.name, tt.g)
	}
	err := ConfigureAll()
	if !errors.Is(err, want) {
		t.Fatalf("ConfigureAll() error handling failed: got %v, want %v", err, want)
	}
}

func TestList(t *testing.T) {
	Register("gen_a", &fakeGen{})
	Register("gen_b", &fakeGen{})
	out := List()
	if len(out) < 2 {
		t.Fatalf("List() length incorrect: got %d, want 2", len(out))
	}
	for _, v := range []string{"gen_a", "gen_b"} {
		found := false
		for _, ov := range out {
			if v == ov {
				found = true
			}
		}
		if !found {
			t.Errorf("List() missing expected generator %s", v)
		}
	}
}
