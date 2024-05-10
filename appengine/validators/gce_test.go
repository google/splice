/*
Copyright 2019 Google LLC

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    https://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package validators

import (
	"bytes"
	"golang.org/x/net/context"
	"fmt"

	"google.golang.org/appengine/v2/aetest"
	"google.golang.org/appengine/v2"
)

// fakeContext is primarily used to provide an appengine Context for test purposes.
func fakeContext() (context.Context, error) {
	inst, err := aetest.NewInstance(nil)
	if err != nil {
		return nil, fmt.Errorf("aetest.NewInstance: %v", err)
	}
	r, err := inst.NewRequest("POST", "/bogus", bytes.NewReader([]byte("test")))
	if err != nil {
		return nil, fmt.Errorf("inst.NewRequest: %v", err)
	}
	return appengine.NewContext(r), nil
}
