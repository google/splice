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

// Package crypto provides cryptographic functionality to SpliceD and CLI.
package crypto

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"fmt"
	"io"
)

var (
	newHash = sha256.New
	rng     = rand.Reader
)

// Metadata holds the join metadata along with its RSA-encrypted AES key
type Metadata struct {
	AESKey []byte
	Data   []byte
	Nonce  []byte
}

// Encrypt encrypts the metadata using an RSA-encrypted AES key.
func (m *Metadata) Encrypt(pubKey []byte) error {
	// Generate a random AES private key.
	aesKey := make([]byte, 32)
	if _, err := rand.Read(aesKey); err != nil {
		return fmt.Errorf("generate aes key: %v", err)
	}

	// RSA encrypt the key with the client's pubkey for transit.
	cert, err := x509.ParsePKIXPublicKey(pubKey)
	if err != nil {
		return fmt.Errorf("decoding public key: %v", err)
	}

	m.AESKey, err = rsa.EncryptOAEP(newHash(), rng, cert.(*rsa.PublicKey), aesKey, []byte(""))
	if err != nil {
		return fmt.Errorf("encrypting aes key: %v", err)
	}

	// AES encrypt the blob.
	block, err := aes.NewCipher(aesKey)
	if err != nil {
		return fmt.Errorf("aes cipher: %v", err)
	}
	m.Nonce = make([]byte, 12)
	if _, err := io.ReadFull(rng, m.Nonce); err != nil {
		return fmt.Errorf("aes nonce: %v", err)
	}
	aesgcm, err := cipher.NewGCM(block)
	if err != nil {
		return fmt.Errorf("aes gcm: %v", err)
	}
	m.Data = aesgcm.Seal(nil, m.Nonce, m.Data, nil)

	return nil
}
