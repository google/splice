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
	"context"
	"errors"
	"fmt"
	"time"

	"google.golang.org/appengine"
	"cloud.google.com/go/datastore"
	"splice/appengine/server"
	"splice/models"
)

// Client is a datastore client that includes transaction
// relevant metadata.
type Client struct {
	client *datastore.Client
	Keys   []*datastore.Key
	Req    *models.Request
	tx     *datastore.Transaction
}

// Close closes the datastore client.
func (c *Client) Close() error {
	if c.client == nil {
		return fmt.Errorf("no active client")
	}

	return c.client.Close()
}

// StartTx starts a new transaction in an existing datastore client.
// Other functions in this package require that a transaction exist
// prior to being used. Transactions started should be closed by the
// caller explicitly through CommitTx or at least through a defer
// statement.
func (c *Client) StartTx(ctx context.Context) error {
	if c.tx != nil {
		return fmt.Errorf("cannot start a new tx until the prior tx is committed: c.tx(%v)", c.tx)
	}

	var err error
	c.tx, err = c.client.NewTransaction(ctx)
	if err != nil {
		return fmt.Errorf("client.NewTransaction(%v) returned %v", ctx, err)
	}

	return nil
}

// CommitTx finalizes and commits an existing transaction to the
// datastore. It must be called after StartTx. Other datastore
// actions such as Save and Find must take place prior to calling
// CommitTx.
func (c *Client) CommitTx() error {
	if c.tx == nil {
		return errors.New("no transaction to commit")
	}

	if _, err := c.tx.Commit(); err != nil {
		return fmt.Errorf("transaction.Commit() returned %v", err)
	}

	// The committed transaction must be cleared from the
	// datastore client to allow a new transaction to be created.
	c.tx = nil
	return nil
}

// RollbackTx rolls back an in-flight transaction. It is typically
// used to ensure transaction cleanups when a processing error
// has occurred during processing.
func (c *Client) RollbackTx() error {
	if c.tx == nil {
		return nil
	}
	return c.tx.Rollback()
}

// Save commits a request to the datastore. An int identifying
// the status is always returned to be passed to the Splice
// client.
func (c *Client) Save(ctx context.Context) (server.StatusCode, error) {
	var err error
	if c.Req == nil || c.client == nil {
		return server.StatusDatastoreWriteError,
			errors.New("client must contain a valid models.Request")
	}
	if c.tx == nil {
		return server.StatusDatastoreWriteError,
			errors.New("client does not have an active transaction")
	}

	// Create a new row if we are processing a new request.
	if len(c.Keys) == 0 {
		ancestor := datastore.NameKey("RequestID", c.Req.RequestID, nil)
		c.Keys = []*datastore.Key{datastore.IncompleteKey("Request", ancestor)}
	}

	if _, err = c.tx.Put(c.Keys[0], c.Req); err != nil {
		return server.StatusDatastoreWriteError,
			fmt.Errorf("transaction.Put(%v, %v) returned %v", c.Keys[0], c.Req, err)
	}

	return server.StatusSuccess, nil
}

// Find searches for a previously committed request using the Request ID.
// Not Found is returned as a status code with a nil error.
func (c *Client) Find(ctx context.Context, reqID string) (server.StatusCode, error) {
	if reqID == "" || c.client == nil {
		return server.StatusDatastoreWriteError,
			fmt.Errorf("missing requestID(%q)", reqID)
	}

	var err error
	var requests []models.Request
	ancestor := datastore.NameKey("RequestID", reqID, nil)
	query := datastore.NewQuery("Request").Ancestor(ancestor)

	// Associate the current transaction if one is specified.
	if c.tx != nil {
		query.Transaction(c.tx)
	}

	c.Keys, err = c.client.GetAll(ctx, query, &requests)
	if err != nil {
		return server.StatusDatastoreLookupError,
			fmt.Errorf("client.GetAll(%v, %v) returned %v", ctx, query, err)
	}

	if len(requests) < 1 {
		return server.StatusDatastoreLookupNotFound, nil
	}

	c.Req = &requests[0]
	return server.StatusSuccess, nil
}

// FindOrphans searches for requests that need to be cleaned up or were
// never processed.
func (c *Client) FindOrphans(ctx context.Context, olderThan time.Duration, kind string) ([]*datastore.Key, []models.Request, error) {
	if olderThan <= 0 {
		return nil, nil, fmt.Errorf("olderThan: got(%d), want(>0)", olderThan)
	}
	if c.client == nil {
		return nil, nil, errors.New("missing datastore client")
	}
	if c.tx == nil {
		return nil, nil, errors.New("client does not have an active transaction")
	}

	var requestsRaw []models.Request
	query := datastore.NewQuery("Request").Filter("Status =", kind)
	keysRaw, err := c.client.GetAll(ctx, query, &requestsRaw)
	if err != nil {
		return nil, nil, fmt.Errorf("client.GetAll(%v, %v) for returned %v", ctx, query, err)
	}

	// Filter out current requests here to workaround
	// query filter comparisons and time.Time not
	// playing nice.
	var keys []*datastore.Key
	var requests []models.Request
	for i, req := range requestsRaw {
		if time.Since(req.AcceptTime) > olderThan {
			keys = append(keys, keysRaw[i])
			requests = append(requests, req)
		}
	}

	return keys, requests, nil
}

// NewClient returns a splice datastore client to the caller.
func NewClient(ctx context.Context, req *models.Request) (*Client, server.StatusCode, error) {
	client, err := datastore.NewClient(ctx, appengine.AppID(ctx))
	if err != nil {
		return nil,
			server.StatusDatastoreClientCreateError,
			fmt.Errorf("datastore.NewClient(%v) returned: %v", ctx, err)
	}

	return &Client{
		client: client,
		Req:    req,
	}, server.StatusSuccess, nil
}
