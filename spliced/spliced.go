// +build windows

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

/*
Package spliced processes domain join requests from the Cloud Datastore.

The core of the Splice joiner runs as a single goroutine which allows it
to function independently of the Windows service which manages it.
Under normal operation, the goroutine for spliced does not exit, unless the
parent Windows service stops.

A channel is used to enable the goroutine to signal an internal failure to
the Windows service, allowing it to shutdown cleanly. All other logging is
sent directly to EventLog.
*/
package main

import (
	"golang.org/x/net/context"
	"fmt"
	"time"

	metric "github.com/google/splice/spliced/metric"
	"cloud.google.com/go/datastore"
	"github.com/google/splice/models"
	"github.com/google/splice/shared/certs"
	"github.com/google/splice/shared/crypto"
	"github.com/google/splice/shared/provisioning"
	"github.com/google/splice/spliced/metric/tracker"
	"github.com/google/splice/spliced/pubsub"
)

var (
	conf    appcfg
	metrics *tracker.Tracker
)

// ExitEvt holds an EventLog event explaining why the goroutine had to exit.
type ExitEvt struct {
	Code    uint32
	Message string
}

// Transaction holds an in-flight datastore transaction.
type Transaction struct {
	client *datastore.Client
	keys   []*datastore.Key
	req    models.Request
	tx     *datastore.Transaction
}

// startTransaction opens a datastore transaction and attempts to locate the record with id reqID.
func startTransaction(ctx context.Context, reqID string) (Transaction, error) {
	var trans Transaction
	var err error

	trans.client, err = datastore.NewClient(ctx, conf.ProjectID)
	if err != nil {
		return trans, fmt.Errorf("startTransaction: datastore client creation failed with %v", err)
	}

	trans.tx, err = trans.client.NewTransaction(ctx)
	if err != nil {
		return trans, fmt.Errorf("startTransaction: opening a datastore transaction failed with %v", err)
	}

	var requests []models.Request
	ancestor := datastore.NameKey("RequestID", reqID, nil)
	query := datastore.NewQuery("Request").Ancestor(ancestor).Transaction(trans.tx)

	if trans.keys, err = trans.client.GetAll(ctx, query, &requests); err != nil {
		return trans, fmt.Errorf("startTransaction: obtaining request from the datastore failed with %v", err)
	}

	if len(requests) < 1 {
		return trans, fmt.Errorf("startTransaction: no request received with ID %s", reqID)
	}
	trans.req = requests[0]

	return trans, nil
}

// returnRequest passes the result of the operation to the datastore on its way to the client.
func returnRequest(ctx context.Context, reqID string, success bool, meta *crypto.Metadata) error {
	trans, err := startTransaction(ctx, reqID)
	if err != nil {
		return err
	}
	defer trans.client.Close()

	trans.req.ResponseData = meta.Data
	if success {
		trans.req.Status = models.RequestStatusCompleted
		trans.req.ResponseKey = meta.AESKey
		trans.req.CipherNonce = meta.Nonce
		metrics.Get("join-success").Increment()
	} else {
		trans.req.Status = models.RequestStatusFailed

		metrics.Get("join-fail").Increment()
	}

	trans.req.CompletionTime = time.Now().UTC()

	if _, err := trans.tx.Put(trans.keys[0], &trans.req); err != nil {
		return fmt.Errorf("returnRequest: datastore update failed with %v", err)
	}

	if _, err := trans.tx.Commit(); err != nil {
		return fmt.Errorf("returnRequest: datastore commit failed with %v", err)
	}

	return nil
}

// claimRequest attempts to claim a new join request from the datastore.
func claimRequest(ctx context.Context, reqID string) (models.Request, error) {
	trans, err := startTransaction(ctx, reqID)
	if err != nil {
		return trans.req, err
	}
	defer trans.client.Close()

	if trans.req.Status != models.RequestStatusAccepted || trans.req.ClaimBy != "" {
		return trans.req, fmt.Errorf("claimRequest: request to %s already %s and will be ignored", trans.req.ClaimBy, trans.req.Status)
	}

	trans.req.ClaimBy = conf.Instance
	trans.req.ClaimTime = time.Now().UTC()

	if _, err := trans.tx.Put(trans.keys[0], &trans.req); err != nil {
		return trans.req, fmt.Errorf("claimRequest: datastore update failed with %v", err)
	}

	if _, err := trans.tx.Commit(); err != nil {
		return trans.req, fmt.Errorf("claimRequest: datastore commit failed with %v", err)
	}

	return trans.req, nil
}

func permitReuse(req *models.Request) bool {
	// Always deny reuse if configured locally
	if !conf.PermitReuse {
		return false
	}
	// If allowed locally, do what the server wants
	return req.AttemptReuse
}

// processRequest takes a claimed request, performs any necessary
// checks, processes it and always returns a metadata object
// with the results. Errors in this func are considered non-fatal
// and are logged and returned within the metadata for display to
// the client.
func processRequest(req *models.Request) (crypto.Metadata, error) {
	meta := crypto.Metadata{}
	if err := certs.VerifyCert(req.ClientCert, req.Hostname+"."+conf.Domain, conf.CaURL, conf.CaURLPath, conf.CaOrg, conf.RootsPath, conf.VerifyCert); err != nil {
		elog.Warning(211, fmt.Sprintf("Client verification failed: %v", err))
		metrics.Get("failure-211").Increment()
		meta.Data = []byte(err.Error())
		return meta, err
	}

	var err error
	elog.Info(209, fmt.Sprintf("Attempting to join host %s to domain %s. Hostname reuse is set to %t.", req.Hostname, conf.Domain, permitReuse(req)))
	metrics.Get("join-attempt").Increment()
	meta.Data, err = provisioning.BinData(req.Hostname, conf.Domain, permitReuse(req))
	if err != nil {
		elog.Warning(207, fmt.Sprintf("Failed to join host with: %v", err))
		metrics.Get("failure-207").Increment()
		meta.Data = []byte(err.Error())
		return meta, err
	}

	if conf.EncryptBlob {
		pub, err := certs.PublicKey(req.ClientCert)
		if err != nil {
			elog.Warning(212, fmt.Sprintf("Unable to obtain certificate public key: %v", err))
			metrics.Get("failure-212").Increment()
			meta.Data = []byte(err.Error())
			return meta, err
		}

		if err := meta.Encrypt(pub); err != nil {
			elog.Warning(210, fmt.Sprintf("encryptMeta: %v", err))
			metrics.Get("failure-210").Increment()
			meta.Data = []byte(err.Error())
			return meta, err
		}
	}

	elog.Info(209, fmt.Sprintf("Attempting to create or modify %q computer object in domain %q", req.Hostname, conf.Domain))
	return meta, nil
}

// Run the splice daemon continuously, listening for new requests.
func Run(ctx context.Context) ExitEvt {
	client, err := pubsub.NewClient(ctx, conf.ProjectID)
	if err != nil {
		return ExitEvt{204, fmt.Sprintf("Failed to create client. %v", err)}
	}
	for {
		elog.Info(200, "Awaiting join requests...")
		metrics.Get("waiting").Set(1)
		reqID, err := pubsub.NewJoinRequest(ctx, client, conf.Topic)
		metrics.Get("waiting").Set(0)
		if err != nil {
			metrics.Get("failure-205").Increment()
			elog.Error(205, fmt.Sprintf("%v", err))
			time.Sleep(1 * time.Minute)
			continue
		}

		elog.Info(200, fmt.Sprintf("NewJoinRequest: pulled message for processing, %v", reqID))
		req, err := claimRequest(ctx, reqID)
		if err != nil {
			elog.Error(206, fmt.Sprintf("%v", err))
			metrics.Get("failure-206").Increment()
			continue
		}

		success := true
		meta, err := processRequest(&req)
		if err != nil {
			success = false
		}

		if err = returnRequest(ctx, reqID, success, &meta); err != nil {
			elog.Error(208, fmt.Sprintf("%v", err))
			metrics.Get("failure-208").Increment()
		}
		for i := range meta.Data {
			meta.Data[i] = 0
		}
	}
}

func initMetrics() error {
	metrics = tracker.New()

	// Counters
	for _, name := range []string{
		"failure-205",
		"failure-206",
		"failure-207",
		"failure-208",
		"failure-210",
		"failure-211",
		"failure-212",
		"join-attempt",
		"join-fail",
		"join-success",
	} {
		m, err := metric.NewCounter(name)
		if err != nil {
			return err
		}
		metrics.Add(m)
	}

	// Gauges
	for _, name := range []string{
		"waiting",
	} {
		m, err := metric.NewGauge(name)
		if err != nil {
			return err
		}
		metrics.Add(m)
	}
	return nil
}

// Init initializes the internal config and logging. Must call before Run.
func Init() error {
	var err error
	if err := initMetrics(); err != nil {
		return err
	}

	conf, err = getConfig()
	if err != nil {
		return fmt.Errorf("Could not obtain configuration from registry. %v", err)
	}
	elog.Info(201, fmt.Sprintf(
		"Application configured from registry.\n\n"+
			"Domain: %v\n"+
			"Svc name: %v\n"+
			"Project id: %v\n"+
			"Topic name: %v\n"+
			"Encrypt blob: %v\n"+
			"Verify certs: %v\n"+
			"CA URL: %v\n"+
			"CA URL Path: %v\n"+
			"CA Expected Org: %v\n"+
			"Permit reuse: %v",
		conf.Domain,
		conf.Instance,
		conf.ProjectID,
		conf.Topic,
		conf.EncryptBlob,
		conf.VerifyCert,
		conf.CaURL,
		conf.CaURLPath,
		conf.CaOrg,
		conf.PermitReuse))

	return nil
}
