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

//go:build windows
// +build windows

package certs

import (
	"errors"
	"fmt"
	"log"

	"github.com/google/certtostore"
	"golang.org/x/sys/windows"
)

// Store holds an open connection to the certificate store.
type Store struct {
	handle *certtostore.WinCertStore
}

// NewStore opens a handle to the certificate store.
func NewStore(container string, issuers, intermediates []string) (*Store, error) {
	var s Store
	// Open the local cert store. Provider generally shouldn't matter, so use Software which is ubiquitous. See comments in getHostKey.
	store, err := certtostore.OpenWinCertStore(certtostore.ProviderMSSoftware, container, issuers, intermediates)
	if err != nil {
		return nil, fmt.Errorf("OpenWinCertStore: %v", err)
	}
	s.handle = store
	return &s, nil
}

// Close closes the certificate store.
func (s *Store) Close() error {
	if s.handle != nil {
		return s.handle.Close()
	}
	return nil
}

// Find attempts to populate the Certificate data from a host certificate matching the specifications.
func (s *Store) Find() (Certificate, Context, error) {
	ctx := Context{}
	c := Certificate{}

	// Obtain the first cert matching all of container/issuers/intermediates in the store.
	// This function is indifferent to the provider the store was opened with, as the store lists certs
	// from all providers.
	crt, context, err := s.handle.CertWithContext()
	if err != nil {
		return c, ctx, fmt.Errorf("cert: %v", err)
	}
	if crt == nil {
		return c, ctx, errors.New("no certificate found")
	}
	c.Cert = crt
	ctx.Ctx = context

	// Obtain the private key from the cert. This *should* work regardless of provider because
	// the key is directly linked to the certificate.
	key, err := s.handle.CertKey(context)
	if err != nil {
		log.Printf("A private key was not found in '%s'.", s.handle.ProvName)
		return c, ctx, err
	}
	c.Decrypter = key
	c.Key = key

	return c, ctx, nil
}

// Context holds a context for a certificate in the Windows certificate store. It must
// be closed once all references to the certificate are complete.
type Context struct {
	Ctx *windows.CertContext
}

// Close the context for the host certificate. It's important that the context
// remain open while in use to avoid Windows releasing the corresponding handles
// to the objects in the cert store.
func (c *Context) Close() {
	if c.Ctx != nil {
		certtostore.FreeCertContext(c.Ctx)
	}
}
