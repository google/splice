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

// Package prefix provides a generator that generates names with a specific prefix.
package prefix

import (
	"fmt"

	"github.com/pkg/errors"
	"github.com/google/splice/generators"
	"github.com/google/glazier/go/registry"
)

var (
	// ErrConfig is returned if configuration fails to load the expected values
	ErrConfig = errors.New("unable to load configuration")
	// ErrInvalidLength is returned if the prefix generator is misconfigured with an invalid Length parameter
	ErrInvalidLength = errors.New("prefix generator requires a naming length greater than the specified prefix")
	// ErrInvalidPrefix is returned if the prefix generator is misconfigured with an invalid Prefix parameter
	ErrInvalidPrefix = errors.New("prefix generator requires a prefix string")
)

func init() {
	generators.Register("prefix", &pf{})
}

type pf struct {
	configured bool
	length     int
	prefix     string
}

// Configure configures this generator prior to use.
func (p *pf) Configure() error {
	pre, err := registry.GetString(generators.RootKey+`\prefix`, "Prefix")
	if err != nil {
		return fmt.Errorf("%w: %v", ErrConfig, err)
	}
	p.prefix = pre
	length, err := registry.GetInteger(generators.RootKey+`\prefix`, "Length")
	if err != nil {
		return fmt.Errorf("%w: %v", ErrConfig, err)
	}
	p.length = int(length)

	if err := p.validate(); err != nil {
		return err
	}
	p.configured = true
	return nil
}

func (p *pf) validate() error {
	switch {
	case p.length < 1:
		return ErrInvalidLength
	case p.length > 15:
		return generators.ErrLongName
	case p.length < len(p.prefix)+1:
		return ErrInvalidLength
	case p.prefix == "":
		return ErrInvalidPrefix
	}
	return nil
}

// Generate runs this generator. This is a placeholder.
func (p *pf) Generate(input []byte) (string, error) {
	if !p.configured {
		return "", generators.ErrNotConfigured
	}

	return "", fmt.Errorf("not implemented")
}
