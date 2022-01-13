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

// Package generators provides custom hostname generation for use with Splice.
package generators

import (
	"sync"

	"go.uber.org/atomic"
	"github.com/pkg/errors"
)

var (
	// RootKey is the root registry key for generator configuration
	RootKey = `SOFTWARE\Splice\Generators`

	// ErrLongName is returned in cases where a name may exceed Active Directory limits
	ErrLongName = errors.New("names greater than 15 characters may fail to join")
	// ErrNotConfigured indicates that a generator has not yet been configured.
	// Try calling ConfigureAll() first.
	ErrNotConfigured = errors.New("generator is not configured")
)

type generator interface {
	Configure() error
	Generate([]byte) (string, error)
}

var (
	mu    sync.Mutex
	atoms atomic.Value
)

func init() {
	atoms.Store(make(map[string]generator))
}

// ConfigureAll configures all generators for use. This should be called
// once at setup, but can be called subsequently to reconfigure.
func ConfigureAll() error {
	mu.Lock()
	defer mu.Unlock()
	gens, _ := atoms.Load().(map[string]generator)
	for _, v := range gens {
		if err := v.Configure(); err != nil {
			return err
		}
	}
	return nil
}

// List returns a list of the available generators by name.
func List() []string {
	mu.Lock()
	defer mu.Unlock()
	m := []string{}
	gens, _ := atoms.Load().(map[string]generator)
	for k := range gens {
		m = append(m, k)
	}
	return m
}

// Register registers a new generator.
func Register(name string, g generator) {
	mu.Lock()
	defer mu.Unlock()
	gens, _ := atoms.Load().(map[string]generator)
	gens[name] = g
	atoms.Store(gens)
}
