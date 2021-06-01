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
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"time"

	"google.golang.org/appengine"
	"google.golang.org/appengine/log"
	"github.com/google/splice/appengine/server"
	"github.com/google/splice/models"
)

// ResultHandler is a custom http handler that services domain join
// result status queries coming in from splice clients.
type ResultHandler func(http.ResponseWriter, *http.Request) *models.Response

// ServeHTTP implements http.Handler and handles errors returned from ResultHandler.
func (rh ResultHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	ctx := appengine.NewContext(r)
	resp := rh(w, r)
	if resp.ErrorCode != server.StatusSuccess {
		// If we had a problem with the result check, log why
		// so we have a record both server and client side.
		log.Warningf(ctx, "%d %q while processing result for %q", resp.ErrorCode, resp.Status, resp.RequestID)
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
	log.Infof(ctx, "successfully processed response with requestID '%q' for host '%q'", resp.RequestID, resp.Hostname)
}

// ProcessResult requires a models.Status with a ClientID and
// a RequestID. It retrieves the current status of the request and
// provides a response to the client. If the request is ready to
// be returned, it finalizes the request and returns the join data as
// part of the response. If a request has become orphaned, it will
// release the request so that another splice joiner can claim it.
// Errors are returned in the context of models.Response.
func ProcessResult(w http.ResponseWriter, r *http.Request) *models.Response {
	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		return &models.Response{
			ErrorCode: server.StatusRequestUnreadable,
			Status:    "unable to read HTTP request body",
		}
	}

	if len(body) == 0 {
		return &models.Response{
			ErrorCode: server.StatusJSONEmpty,
			Status:    "empty HTTP json result query body",
		}
	}

	var reqStatus models.StatusQuery
	if err = json.Unmarshal(body, &reqStatus); err != nil {
		return &models.Response{
			ErrorCode: server.StatusJSONUmarshalError,
			Status:    "unable to unmarshal json request",
		}
	}

	if reqStatus.RequestID == "" || reqStatus.ClientID == "" {
		return &models.Response{
			ErrorCode: server.StatusReqProcessingError,
			Status:    "invalid result query: RequestID and ClientID are required",
		}
	}

	// Use an empty struct instead of a var to simplify testing.
	dc := &Client{Req: &models.Request{}}
	ctx := appengine.NewContext(r)
	response := &models.Response{}
	if useDatastore {
		var status server.StatusCode
		dc, status, err = NewClient(ctx, nil)
		if err != nil {
			return &models.Response{
				ErrorCode: status,
				Status:    err.Error(),
			}
		}
		defer dc.Close()

		status, err = dc.Find(ctx, reqStatus.RequestID)
		if err != nil && status != server.StatusDatastoreLookupNotFound {
			return &models.Response{
				ErrorCode: status,
				Status:    err.Error(),
			}
		}
		if status == server.StatusDatastoreLookupNotFound {
			return &models.Response{
				ErrorCode: status,
				Status:    fmt.Sprintf("result not found: %q", reqStatus.RequestID),
			}
		}

		if err := verifyCert(ctx, dc.Req.ClientID, r); err != nil {
			return &models.Response{
				ErrorCode: server.StatusInvalidCertError,
				Status:    err.Error(),
			}
		}

		if dc.Req.Status == models.RequestStatusReturned {
			return &models.Response{
				ErrorCode: server.StatusRequestResultReplay,
				Status:    fmt.Sprintf("the result for request %q has already been returned", dc.Req.RequestID),
			}
		}

		// Populating our response a little earlier allows us to
		// sanitize response data for completed requests.
		response = &models.Response{
			ErrorCode:    server.StatusSuccess,
			Status:       dc.Req.Status,
			Hostname:     dc.Req.Hostname,
			RequestID:    dc.Req.RequestID,
			ResponseData: dc.Req.ResponseData,
			ResponseKey:  dc.Req.ResponseKey,
			CipherNonce:  dc.Req.CipherNonce,
		}

		// If the request remains outstanding, check for orphans and return status info.
		// We don't start a transaction unless the request looks orphaned.
		if dc.Req.Status != models.RequestStatusCompleted {
			// Republish requests that were claimed but never completed by the joiner.
			if time.Now().Sub(dc.Req.ClaimTime) > 300*time.Second && !dc.Req.ClaimTime.IsZero() {
				log.Infof(ctx, "requestID '%q' will be released because it was claimed at %v by %s but has not been completed.", response.RequestID, dc.Req.ClaimTime, dc.Req.ClaimBy)
				return releaseRequest(ctx, dc)
			}
			// Republish requests that were never claimed by a joiner.
			if time.Now().Sub(dc.Req.AcceptTime) > 300*time.Second && dc.Req.ClaimBy == "" {
				log.Infof(ctx, "requestID '%q' will be republished because it was accepted at %v but was never claimed.", response.RequestID, dc.Req.AcceptTime)
				return releaseRequest(ctx, dc)
			}

			return response
		}

		// If we get here, we should be good to return the results.

		// Sanitize cryptographic data from the datastore.
		dc.Req.ResponseData = nil
		dc.Req.ResponseKey = nil
		dc.Req.CipherNonce = nil
		dc.Req.Status = models.RequestStatusReturned

		if err = dc.StartTx(ctx); err != nil {
			return &models.Response{
				ErrorCode: server.StatusDatastoreTxCreateError,
				Status:    err.Error(),
			}
		}
		// Defer a rollback to protect against transactions
		// that may be left open due to unexpected processing
		// errors.
		defer dc.RollbackTx()

		if status, err = dc.Save(ctx); err != nil {
			return &models.Response{
				ErrorCode: status,
				Status:    err.Error(),
			}
		}

		if err := dc.CommitTx(); err != nil {
			return &models.Response{
				ErrorCode: server.StatusDatastoreTxCommitError,
				Status:    err.Error(),
			}
		}
	}

	return response
}

// releaseRequest resets a request so that it may be claimed
// for processing by another joiner server. Released requests
// are re-published to pubsub.
func releaseRequest(ctx context.Context, dc *Client) *models.Response {
	if err := dc.StartTx(ctx); err != nil {
		return &models.Response{
			ErrorCode: server.StatusDatastoreTxCreateError,
			Status:    err.Error(),
		}
	}
	defer dc.RollbackTx()

	dc.Req.Status = models.RequestStatusAccepted
	dc.Req.ClaimBy = ""
	dc.Req.ClaimTime = time.Time{}

	// We could probably save and commit above, but leaving this here
	// to make it clearer that we're re-publishing on purpose and not just
	// because a request is in RequestStatusAccepted.
	if status, err := dc.Save(ctx); err != nil {
		return &models.Response{
			ErrorCode: status,
			Status:    err.Error(),
		}
	}

	if err := dc.CommitTx(); err != nil {
		return &models.Response{
			ErrorCode: server.StatusDatastoreTxCommitError,
			Status:    err.Error(),
		}
	}

	if err := publishRequest(ctx, dc.Req.RequestID); err != nil {
		return &models.Response{
			ErrorCode: server.StatusPubsubFailure,
			Status:    err.Error(),
		}
	}

	log.Infof(ctx, "released orphaned request %q", dc.Req.RequestID)
	return &models.Response{
		ErrorCode: server.StatusSuccess,
		Status:    fmt.Sprintf("released orphaned request: %q", dc.Req.RequestID),
	}
}
