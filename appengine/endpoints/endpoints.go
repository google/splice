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

// Package endpoints contains all request handler functions for Splice.
// Individual handlers are separated into their own files for readability.
package endpoints

import (
	"golang.org/x/net/context"
	"errors"
	"fmt"
	"net/http"
	"os"

	"github.com/google/splice/appengine/validators"
	"google.golang.org/appengine/log"
)

// reqIDLength represents the length in bytes of a requestID
// e.g. 128 bytes = 1024 bits.
const reqIDLen = 128

// For easier testing, use a vars for validators and
// datastore usage
var (
	validatorsNewAttended   = validators.New
	validatorsNewUnattended = validators.NewUnattended
	useDatastore            = true
	usePubsub               = true
)

// verifyCert returns an error if there is a discrepancy between
// ClientID (the hash of Client's certificate) and the fingerprint of the certificate
// used for TLS, or VERIFY_CERT_HEADER. verifyCert is enabled by default and can be
// disabled by setting the VERIFY_CERT environment variable to false. If verification
// is set to false or there was no error during verification, nil is returned.
func verifyCert(ctx context.Context, fp string, r *http.Request) error {
	enabled := os.Getenv("VERIFY_CERT")
	if enabled == "false" {
		log.Infof(ctx, "VERIFY_CERT=%s, skipping cert fingerprint verification.", enabled)
		return nil
	}
	headerName := os.Getenv("VERIFY_CERT_HEADER")
	if headerName == "" {
		return errors.New("VERIFY_CERT_HEADER must not be empty")
	}

	header := r.Header.Get(headerName)
	if header == "" {
		return fmt.Errorf("VERIFY_CERT='%s', but no %s header was present", enabled, headerName)
	}
	if fp == "" {
		return fmt.Errorf("VERIFY_CERT='%s', a fingerprint(%s) is required", enabled, fp)
	}

	if fp != header {
		log.Warningf(ctx, "Cert fingerprint(%s) mismatch with header('%s' = %s), aborting.", fp, headerName, header)
		return fmt.Errorf("fingerprint(%s) did not match header('%s' = %s)", fp, headerName, header)
	}

	log.Infof(ctx, "Cert fingerprint(%s) matched header('%s' = %s), processing request", fp, headerName, header)
	return nil
}
