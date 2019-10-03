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

/*
Package metric implements a very simple internal metric with Set and Increment
abilities.

These metrics can be exported via a web service, or can simply serve as an
example for building more complex metric handling.
*/
package metric

import "sync"

// Metric tracks a basic int64 type value metric.
type Metric struct {
	name string

	mu    sync.Mutex
	value int64
}

// NewCounter allocates a new metric for incremental values.
func NewCounter(name string) (*Metric, error) {
	return &Metric{name: name}, nil
}

// NewGauge allocates a new metric for arbitrary values.
func NewGauge(name string) (*Metric, error) {
	return &Metric{name: name}, nil
}

// Increment adds one to the current value.
func (c *Metric) Increment() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.value++
	return nil
}

// Name retrieves the name of the metric.
func (c *Metric) Name() string {
	return c.name
}

// Value retrieves the current value of the metric.
func (c *Metric) Value() int64 {
	c.mu.Lock()
	defer c.mu.Unlock()

	return c.value
}

// Set updates the internal metric value.
func (c *Metric) Set(val int64) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.value = val
	return nil
}
