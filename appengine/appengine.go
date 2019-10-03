/*
Copyright 2018 Google LLC

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

// Package appengine is an web-application that provides a public API
// for cloud based offline Active Directory domain joins for Windows clients.
// It interacts with a client to process domain join requests and
// provides a result to the clients.
package main

import (
	http "net/http"
	"github.com/google/splice/appengine/endpoints"
)

func init() {
	http.Handle("/request", &endpoints.AttendedRequestHandler{})
	http.Handle("/result", endpoints.ResultHandler(endpoints.ProcessResult))
	http.Handle("/request-unattended", &endpoints.UnattendedRequestHandler{})
	http.Handle("/result-unattended", endpoints.ResultHandler(endpoints.ProcessResult))
}
