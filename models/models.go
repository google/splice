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

// Package models provides models for data storage and transfer.
package models

import (
	"time"

	"github.com/google/splice/appengine/server"
	"github.com/google/splice/cli/gce"
)

// Request Datastore Field Values
const (
	RequestStatusAccepted   = "Accepted"
	RequestStatusProcessing = "Processing"
	RequestStatusCompleted  = "Completed"
	RequestStatusFailed     = "Failed"
	RequestStatusReturned   = "Returned"
)

// ClientRequest models the allowable data that a client can
// submit as part of a request.
type ClientRequest struct {
	Hostname   string
	ClientID   string
	ClientCert []byte

	// Unattended validation
	GCEMetadata gce.Metadata
}

// Request models a new request to join a machine to the domain.
type Request struct {
	RequestID      string
	ClientID       string
	ClientCert     []byte
	Hostname       string
	AcceptTime     time.Time
	ClaimBy        string
	ClaimTime      time.Time
	Status         string
	CompletionTime time.Time
	ResponseData   []byte

	// Unattended validation
	GCEMetadata gce.Metadata

	// Encryption
	ResponseKey []byte
	CipherNonce []byte

	AttemptReuse bool
}

// StatusQuery models a request for the status of a join.
type StatusQuery struct {
	RequestID string
	ClientID  string

	GCEMetadata gce.Metadata
}

// Response models the response to a client request
type Response struct {
	RequestID    string
	Status       string
	ErrorCode    server.StatusCode
	Hostname     string
	ResponseData []byte

	// Encryption
	ResponseKey []byte
	CipherNonce []byte
}
