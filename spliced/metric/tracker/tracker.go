/*
Copyright 2016 Google Inc.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

	http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

// Package tracker manages all internal state metrics for the SpliceD application.
package tracker

import (
	"sync"
)

// Metric models a metric tracking the internal state of the SpliceD application.
type Metric interface {
	Increment() error
	Set(int64) error
}

// Tracker maintains a map of all internal metrics.
type Tracker struct {
	mu       sync.Mutex
	counters map[string]Metric
}

// New allocates a new metric Tracker object.
func New() *Tracker {
	return &Tracker{
		counters: make(map[string]Metric),
	}
}

// Get retrieves a metric by name.
func (t *Tracker) Get(name string) Metric {
	t.mu.Lock()
	defer t.mu.Unlock()

	return t.counters[name]
}

// Add adds a new metric to the tracker.
func (t *Tracker) Add(name string, m Metric) {
	t.mu.Lock()
	defer t.mu.Unlock()

	t.counters[name] = m
}
