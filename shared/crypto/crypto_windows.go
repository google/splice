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

// +build windows

package crypto

import (
	"crypto"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rsa"
	"fmt"

	"github.com/google/certtostore"
)

// Decrypt decrypts a metadata blob using the host's private key.
func (m *Metadata) Decrypt(privKey crypto.Decrypter) ([]byte, error) {
	switch {
	case m.AESKey == nil:
		return nil, fmt.Errorf("AESKey missing")
	case m.Data == nil:
		return nil, fmt.Errorf("Data missing")
	case m.Nonce == nil:
		return nil, fmt.Errorf("Nonce missing")
	}

	var aesKey []byte
	var err error

	switch k := privKey.(type) {
	case *certtostore.Key:
		opts := certtostore.DecrypterOpts{
			Hashfunc: crypto.SHA256,
			Flags:    certtostore.NCryptPadOAEPFlag,
		}
		aesKey, err = k.Decrypt(rng, m.AESKey, opts)
		if err != nil {
			return nil, fmt.Errorf("certtostore.Decrypt returned %v", err)
		}
	case *rsa.PrivateKey:
		aesKey, err = rsa.DecryptOAEP(newHash(), rng, k, m.AESKey, []byte(""))
		if err != nil {
			return nil, fmt.Errorf("DecryptOAEP: %s", err)
		}
	default:
		return nil, fmt.Errorf("unsupported key type %v", k)
	}

	// Continue decryption using the now decrypted AES key.
	block, err := aes.NewCipher(aesKey)
	if err != nil {
		return nil, fmt.Errorf("aes.NewCipher: %s", err)
	}

	aesgcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("cipher.NewGCM: %s", err)
	}

	plaintext, err := aesgcm.Open(nil, m.Nonce, m.Data, nil)
	if err != nil {
		return nil, fmt.Errorf("aesgcm.Open: %s", err)
	}
	return plaintext, nil
}
