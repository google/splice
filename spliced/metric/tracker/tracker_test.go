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

package tracker

import (
	"testing"
)

type TestMetric struct {
	name  string
	value int64
}

func (t *TestMetric) Increment() error { return nil }
func (t *TestMetric) Name() string     { return t.name }
func (t *TestMetric) Set(int64) error  { return nil }
func (t *TestMetric) Value() int64     { return t.value }

func TestAdd(t *testing.T) {
	m := New()
	m.Add(&TestMetric{name: "test1", value: 12345})
	v := m.Get("test1").Value()
	if v != 12345 {
		t.Fatalf("unexpected result for metric test1: %q", v)
	}
}
