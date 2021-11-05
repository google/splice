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

// Find attempts to populate the Certificate data from a host certificate matching the specifications.
func (c *Certificate) Find(container string, issuers, intermediates []string) (Context, error) {
	ctx := Context{}

	// Open the local cert store. Provider generally shouldn't matter, so use Software which is ubiquitous. See comments in getHostKey.
	store, err := certtostore.OpenWinCertStore(certtostore.ProviderMSSoftware, container, issuers, intermediates, false)
	if err != nil {
		return ctx, fmt.Errorf("OpenWinCertStore: %v", err)
	}
	defer store.Close()

	// Obtain the first cert matching all of container/issuers/intermediates in the store.
	// This function is indifferent to the provider the store was opened with, as the store lists certs
	// from all providers.
	crt, context, err := store.CertWithContext()
	if err != nil {
		return ctx, fmt.Errorf("cert: %v", err)
	}
	if c == nil {
		return ctx, errors.New("no certificate found")
	}
	c.Cert = crt
	ctx.Ctx = context

	// Obtain the private key from the cert. This *should* work regardless of provider because
	// the key is directly linked to the certificate.
	key, err := store.CertKey(context)
	if err != nil {
		log.Printf("A private key was not found in '%s'.", store.ProvName)
		return ctx, err
	}
	c.Decrypter = key
	c.Key = key

	return ctx, nil
}
