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
	// EvtMisc indicates that an uncategorized internal event
	EvtMisc = iota + 2000
	// EvtWaiting indicates the daemon has entered wait state
	EvtWaiting
	// EvtConfiguration indicates a change to process configuration
	EvtConfiguration
	// EvtStartup indicates SpliceD startup
	EvtStartup
	// EvtShutdown indicates SpliceD shutdown
	EvtShutdown
	//
	// Request-releated events
	//
	// EvtNewRequest indicates a receipt of a new request
	EvtNewRequest = iota + 100
	// EvtJoinAttempt indicates a join attempt in progress
	EvtJoinAttempt
	// EvtJoinSuccess indicates a successful join operation
	EvtJoinSuccess
	// EvtJoinFailure indicates an unsuccessful join operation
	EvtJoinFailure
	// EvtNameGeneration indicates a dynamic name generation event
	EvtNameGeneration
)

/*
 * Errors
 */
const (
	// EvtErrMisc indicates a miscellaneous internal error condition
	EvtErrMisc = iota + 4000
	// EvtErrStartup indicates an error starting the service
	EvtErrStartup
	// EvtErrSubscription indicates an error in subscription communication
	EvtErrSubscription
	// EvtErrClaim indicates an error claiming a request
	EvtErrClaim
	// EvtErrReturn indicates an error returning a request
	EvtErrReturn
	//
	// Request-releated errors
	//
	// EvtErrVerification indicates an error in request verification
	EvtErrVerification = iota + 100
	// EvtErrEncryption indicates an error encrypting a payload
	EvtErrEncryption
	// EvtErrNaming indicates a failure determining a hostname for a request. This
	// could be a problem with the request or a problem with a generator.
	EvtErrNaming
)
