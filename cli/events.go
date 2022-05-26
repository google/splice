// Copyright 2021 Google LLC
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

package main

/*
 * Internal Events
 */

const (
	//
	// Operational events
	//

	// EvtJoinSuccess indicates a successful join operation
	EvtJoinSuccess = iota + 10000
)

/*
 * Errors
 */

const (
	//
	// Operational errors
	//

	// EvtErrStartup indicates an error starting the service
	EvtErrStartup = iota + 10200
	// EvtErrCertificate indicates an miscellaneous certificate error
	EvtErrCertificate
)

const (
	//
	// Request-related errors
	//

	// EvtErrEncryption indicates an error encrypting a payload
	EvtErrEncryption = iota + 10300
	// EvtErrConnection indicates an error connecting to appengine
	EvtErrConnection
	// EvtErrRequest indicates an error with the splice request
	EvtErrRequest
	// EvtErrPoll indicates an error with polling for the result
	EvtErrPoll
	// EvtErrJoin indicates an error with applying the join metadata
	EvtErrJoin
)
