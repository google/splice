/*
Copyright 2018 Google Inc.

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

// Package server contains shared data and structures used across splice packages
package server

// StatusCode represents a Splice Server status code, and is used to communicate
// reasons for request and result rejections, as well as internal failures.
type StatusCode int

// Server Internal Status Messages. We use static values to ensure that messages
// that may end up in the datastore do not have to be altered if a new const is added.
const (
	StatusSuccess            StatusCode = 0
	StatusRequestUnreadable  StatusCode = 101
	StatusJSONEmpty          StatusCode = 102
	StatusJSONUmarshalError  StatusCode = 103
	StatusJSONMarshalError   StatusCode = 104
	StatusReqProcessingError StatusCode = 105
	StatusInvalidCertError   StatusCode = 106
	StatusInvalidGCEmeta     StatusCode = 107

	// Default validator messages
	StatusRequestHostBlank     StatusCode = 201
	StatusRequestHostLength    StatusCode = 202
	StatusRequestClientIDBlank StatusCode = 203
	StatusRequestResultReplay  StatusCode = 204

	// Dependency validator messages
	StatusDependencyValidationError StatusCode = 301

	// Datastore status messages
	StatusDatastoreClientCreateError StatusCode = 401
	StatusDatastoreTxCreateError     StatusCode = 402
	StatusDatastoreWriteError        StatusCode = 403
	StatusDatastoreLookupError       StatusCode = 404
	StatusDatastoreLookupNotFound    StatusCode = 405
	StatusDatastoreUpdateError       StatusCode = 406
	StatusDatastoreTxCommitError     StatusCode = 407

	// Pubsub status messages
	StatusPubsubFailure StatusCode = 501
)
