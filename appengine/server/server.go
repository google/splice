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

const (
	// StatusSuccess indicates a successful operation.
	StatusSuccess StatusCode = 0
)

// Server Internal Status Messages.
const (
	StatusRequestUnreadable StatusCode = iota + 101
	StatusJSONEmpty
	StatusJSONUmarshalError
	StatusJSONMarshalError
	StatusReqProcessingError
	StatusInvalidCertError
	StatusInvalidGCEmeta
)

// Default validator messages
const (
	StatusRequestHostBlank StatusCode = iota + 201
	StatusRequestHostLength
	StatusRequestClientIDBlank
	StatusRequestResultReplay
	StatusRequestGeneratorError
)

// Dependency validator messages
const (
	StatusDependencyValidationError StatusCode = iota + 301
)

// Datastore status messages
const (
	StatusDatastoreClientCreateError StatusCode = iota + 401
	StatusDatastoreTxCreateError
	StatusDatastoreWriteError
	StatusDatastoreLookupError
	StatusDatastoreLookupNotFound
	StatusDatastoreUpdateError
	StatusDatastoreTxCommitError
)

// Pubsub status messages
const (
	StatusPubsubFailure StatusCode = iota + 501
)
