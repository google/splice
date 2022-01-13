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

package prefix

import (
	"errors"
	"strings"
	"testing"

	"github.com/google/splice/generators"
	"github.com/google/glazier/go/registry"
)

var (
	regRoot = generators.RootKey + `\prefix`
)

func createKeys(pfx string, length int) error {
	if err := registry.Create(regRoot); err != nil {
		return err
	}
	if length > 0 {
		if err := registry.SetInteger(regRoot, "Length", length); err != nil {
			return err
		}
	}
	if pfx != "" {
		if err := registry.SetString(regRoot, "Prefix", pfx); err != nil {
			return err
		}
	}
	return nil
}

func TestConfigure(t *testing.T) {
	tests := []struct {
		desc string
		p    *pf
		want error
	}{
		{"valid", &pf{length: 12, prefix: "splice-"}, nil},
		{"missing prefix", &pf{length: 12, prefix: ""}, ErrConfig},
		{"missing length", &pf{length: -1, prefix: "splice"}, ErrConfig},
	}
	for _, tt := range tests {
		if err := createKeys(tt.p.prefix, tt.p.length); err != nil {
			t.Fatalf("createKeys: %v", err)
		}
		got := tt.p.Configure()
		if !errors.Is(got, tt.want) {
			t.Errorf("%s: got %v, want %v", tt.desc, got, tt.want)
		}
		registry.Delete(regRoot, "Length")
		registry.Delete(regRoot, "Prefix")
	}
}

func TestGenerate(t *testing.T) {
	p := &pf{length: 12, prefix: "splice-", configured: true}
	out, err := p.Generate(nil)
	if err != nil {
		t.Errorf("produced unexpected err: %v", err)
	}
	if len(out) != p.length {
		t.Errorf("generated name of invalid length: got %d, want %d", len(out), p.length)
	}
	if !strings.HasPrefix(out, p.prefix) {
		t.Errorf("produced name with invalid prefix: got %s, want %s", out, p.prefix)
	}
}

func TestGenerateUnconfigured(t *testing.T) {
	p := &pf{length: 12, prefix: "splice-"}
	_, err := p.Generate(nil)
	if !errors.Is(err, generators.ErrNotConfigured) {
		t.Errorf("produced unexpected err: got %v, want %v", err, generators.ErrNotConfigured)
	}
}

func TestValidate(t *testing.T) {
	tests := []struct {
		desc string
		p    *pf
		want error
	}{
		{"valid", &pf{length: 12, prefix: "splice-"}, nil},
		{"name longer than 15", &pf{length: 18, prefix: "splice-"}, generators.ErrLongName},
		{"prefix blank", &pf{length: 12, prefix: ""}, ErrInvalidPrefix},
		{"length too small for prefix", &pf{length: 6, prefix: "splice-"}, ErrInvalidLength},
	}
	for _, tt := range tests {
		got := tt.p.validate()
		if !errors.Is(got, tt.want) {
			t.Errorf("%s: got %v, want %v", tt.desc, got, tt.want)
		}
	}
}
