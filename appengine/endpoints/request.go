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

package endpoints

import (
	"golang.org/x/net/context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"math/rand"
	"net/http"
	"os"
	"time"

	"google.golang.org/appengine"
	"google.golang.org/appengine/log"
	"cloud.google.com/go/pubsub"
	"github.com/google/splice/appengine/server"
	basic "github.com/google/splice/appengine/validators"
	"github.com/google/splice/models"
)

// AttendedRequestHandler implements http.Handler for user interactive joins.
type AttendedRequestHandler struct{}

func (ah AttendedRequestHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	ctx := appengine.NewContext(r)
	checks, err := validatorsNewAttended()
	if err != nil {
		log.Errorf(ctx, "Validator setup returned %v", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	requestResponse(ctx, w, r, checks)
}

// UnattendedRequestHandler implements http.Handler for unattended joins.
type UnattendedRequestHandler struct{}

func (uh UnattendedRequestHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	ctx := appengine.NewContext(r)
	checks, err := validatorsNewUnattended()
	if err != nil {
		log.Errorf(ctx, "Validator setup returned %v", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	requestResponse(ctx, w, r, checks)
}

// requestResponse performs necessary cleanup and provides a response to the client.
func requestResponse(ctx context.Context, w http.ResponseWriter, r *http.Request, checks []basic.Validator) {
	resp := ProcessRequest(ctx, w, r, checks)
	if resp.ErrorCode != server.StatusSuccess {
		log.Warningf(ctx, "could not process request %v", resp)
	}

	jsonResponse, err := json.Marshal(resp)
	if err != nil {
		log.Errorf(ctx, "json.Marshal(%v) failed: %v", resp, err)
		http.Error(
			w,
			fmt.Sprintf("json.Marshal(%v) failed: %v", resp, err),
			http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write(jsonResponse)

	if resp.ErrorCode == server.StatusSuccess {
		log.Infof(ctx, "successfully processed requestID '%q'", resp.RequestID)
	}
}

// ProcessRequest takes a models.Request that is provided by the client,
// and validates it. A response is provided using models.Response.
func ProcessRequest(ctx context.Context, w http.ResponseWriter, r *http.Request, checks []basic.Validator) models.Response {
	request, code, err := unmarshalRequest(r)
	if err != nil {
		return models.Response{
			ErrorCode: code,
			Status:    err.Error(),
		}
	}

	if err := verifyCert(ctx, request.ClientID, r); err != nil {
		return models.Response{
			ErrorCode: server.StatusInvalidCertError,
			Status:    err.Error(),
		}
	}

	// Run this request through all validators
	for _, c := range checks {
		status, err := c.Check(ctx, &request)
		if err != nil {
			return models.Response{
				ErrorCode: status,
				Status:    err.Error(),
			}
		}
	}

	// New requests require a cryptographically secure requestID
	// of a sufficient length. Existing requests re-use their
	// existing requestID.
	//
	request.RequestID, err = generateReqID(reqIDLen)
	if err != nil {
		return models.Response{
			ErrorCode: server.StatusReqProcessingError,
			Status:    fmt.Sprintf("generateReqID(%d) returned %v", reqIDLen, err),
		}
	}

	request.AcceptTime = time.Now()
	request.Status = models.RequestStatusAccepted

	// Initialize an empty datastore client at the appropriate scope.
	dc := &Client{Req: &models.Request{}}
	log.Infof(ctx, "AttemptReuse returned %t in the App Engine request", request.AttemptReuse)
	if useDatastore {
		var status server.StatusCode
		dc, status, err = NewClient(ctx, &request)
		if err != nil {
			return models.Response{
				ErrorCode: status,
				Status:    err.Error(),
			}
		}
		defer dc.Close()

		if err = dc.StartTx(ctx); err != nil {
			return models.Response{
				ErrorCode: server.StatusDatastoreTxCreateError,
				Status:    err.Error(),
			}
		}
		defer dc.RollbackTx()

		if status, err := dc.Save(ctx); err != nil {
			return models.Response{
				ErrorCode: status,
				Status:    err.Error(),
			}
		}

		if err := dc.CommitTx(); err != nil {
			return models.Response{
				ErrorCode: server.StatusDatastoreTxCommitError,
				Status:    err.Error(),
			}
		}

		// Failure to cleanup orphans should be reported
		// but should not stop processing.
		if err := cleanupOrphans(ctx, dc); err != nil {
			log.Warningf(ctx, "cleanupOrphans failed with %v ", err)
		}
	}

	if usePubsub {
		if err := publishRequest(ctx, request.RequestID); err != nil {
			return models.Response{
				ErrorCode: server.StatusPubsubFailure,
				Status:    err.Error(),
			}
		}
	}

	return models.Response{
		Status:    models.RequestStatusAccepted,
		RequestID: request.RequestID,
		ErrorCode: server.StatusSuccess,
	}
}

// GenerateReqID returns a URL-safe, base64 encoded
// securely generated random requestID to identify a request.
// It takes as input the length in bytes of the token to be
// generated. E.G. 64 bytes (512 bit), or 128 bytes (1024 bit).
func generateReqID(n uint) (string, error) {
	if n <= 0 {
		return "", fmt.Errorf("invalid length %d requested", n)
	}
	rand.Seed(time.Now().UnixNano())
	b := make([]byte, n)
	_, err := rand.Read(b)
	if err != nil {
		return "", fmt.Errorf("rand.Read returned %v", err)
	}
	return base64.URLEncoding.EncodeToString(b), nil
}

// unmarshalRequest takes a raw inbound request and returns
// a models.Request for processing
func unmarshalRequest(r *http.Request) (models.Request, server.StatusCode, error) {
	var clientRequest models.ClientRequest

	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		return models.Request{},
			server.StatusRequestUnreadable,
			errors.New("unable to read HTTP request body")
	}

	if len(body) == 0 {
		return models.Request{},
			server.StatusJSONEmpty,
			errors.New("empty HTTP JSON request body")
	}

	if err = json.Unmarshal(body, &clientRequest); err != nil {
		return models.Request{},
			server.StatusJSONUmarshalError,
			errors.New("unable to unmarshal JSON request")
	}

	return models.Request{
			Hostname:      clientRequest.Hostname,
			ClientID:      clientRequest.ClientID,
			ClientCert:    clientRequest.ClientCert,
			GCEMetadata:   clientRequest.GCEMetadata,
			GeneratorID:   clientRequest.GeneratorID,
			GeneratorData: clientRequest.GeneratorData,
		},
		server.StatusSuccess,
		nil
}

// cleanupOrphans looks for requests that are too old or were
// tried once and not re-tried and makes sure that they are not
// left in an indeterminate state.
func cleanupOrphans(ctx context.Context, dc *Client) error {
	// Orphans older than 24 hours are cleaned up.
	olderThan := 24 * time.Hour

	// Check the following statuses for orphaned entries.
	s := []string{
		models.RequestStatusAccepted,
		models.RequestStatusProcessing,
		models.RequestStatusCompleted,
	}

	if err := dc.StartTx(ctx); err != nil {
		return fmt.Errorf("dc.StartTx returned %v", err)
	}
	defer dc.RollbackTx()

	// Cleanup requests by type because the datastore GetAll
	// func contcatenates filters using AND.
	for _, kind := range s {
		keys, requests, err := dc.FindOrphans(ctx, olderThan, kind)
		if err != nil {
			return fmt.Errorf("FindOrphans(%d, %s) = %v", olderThan, kind, err)
		}

		if len(requests) > 0 {
			for i, orphan := range requests {
				orphan.Status = models.RequestStatusFailed
				dc.Req = nil
				dc.Req = &orphan
				dc.Keys[0] = keys[i]
				if _, err = dc.Save(ctx); err != nil {
					return err
				}
				log.Infof(ctx, "cleaned up orphan with reqID = %q ", orphan.RequestID)
			}
		}
	}

	if err := dc.CommitTx(); err != nil {
		return fmt.Errorf("dc.CommitTx returned %v", err)
	}

	return nil
}

// Publishes a request to the pubsub channel.
func publishRequest(ctx context.Context, reqID string) error {
	envProject := appengine.AppID(ctx)
	envTopic := os.Getenv("PUBSUB_TOPIC")
	if envTopic == "" {
		return errors.New("PUBSUB_TOPIC environment variable not set")
	}

	ps, err := pubsub.NewClient(ctx, envProject)
	if err != nil {
		return fmt.Errorf("pubsub.NewClient(%q) returned: %v", envProject, err)
	}

	topic := ps.Topic(envTopic)
	defer topic.Stop()
	res := topic.Publish(ctx, &pubsub.Message{Data: []byte(reqID)})

	msgID, err := res.Get(ctx)
	if err != nil {
		return fmt.Errorf("topic.Publish error publishing to topic %s: %v, %v", envTopic, res, err)
	}

	log.Infof(ctx, "request id %q published with msg id %q", reqID, msgID)
	return nil
}
