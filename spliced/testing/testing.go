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

// Package testing provides helpers for testing SpliceD.
package testing

import "errors"

var (
	// ErrReuse is returned if a host cannot be joined again due to reuse being disabled
	ErrReuse = errors.New("reuse disabled and host already exists")

	// SuccessBlob is returned for a successful join, in lieu of a real metadata blob.
	SuccessBlob = []byte("good job!")
)

// InactiveDirectory provides a fake AD structure for testing
type InactiveDirectory struct {
	Computers map[string]bool
}

// NewInactiveDirectory returns a new InactiveDirectory instance for testing.
func NewInactiveDirectory() *InactiveDirectory {
	return &InactiveDirectory{
		Computers: make(map[string]bool),
	}
}

// Join joins a host to the fake domain
func (id *InactiveDirectory) Join(name, domain string, reuse bool) ([]byte, error) {
	if _, ok := id.Computers[name]; ok && !reuse {
		return nil, ErrReuse
	}
	id.Computers[name] = true
	return SuccessBlob, nil
}
