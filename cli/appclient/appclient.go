/*
Copyright 2019 Google LLC

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    https://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

// Package appclient provides a TLS enabled HTTP client for use with splice
// requests.
package appclient

import (
	"crypto"
	"crypto/tls"
	"errors"
	"net/http"
)

// Connect performs an OAuth 2.0 authorization for the user, and returns
// an OAuth enabled http.Client for use in subsequent API calls.
func Connect(server string, username string) (*http.Client, error) {
	// Connect is not yet implemented.
	return nil, errors.New("user-based authorization is not implemented")
}

// TLSClient returns a TLS enabled http client without SSO credentials for use
// in unauthenticated requests. It requires a raw x509 certificate and its
// associated crypto.Decrypter in order to generate the required TLS configuration.
func TLSClient(rawCert []byte, decrypter crypto.Decrypter) (*http.Client, error) {
	tlsCerts := []tls.Certificate{{Certificate: [][]byte{rawCert}, PrivateKey: decrypter}}
	tlsCfg := &tls.Config{Certificates: tlsCerts, MinVersion: tls.VersionTLS12}

	return &http.Client{Transport: &http.Transport{TLSClientConfig: tlsCfg}}, nil
}
