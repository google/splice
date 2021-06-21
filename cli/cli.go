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

// The cli application implements the end-user client for the Splice service.
package main

import (
	"bytes"
	"crypto"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"strings"
	"time"

	appclient "github.com/google/splice/cli/appclient"
	"github.com/google/certtostore"
	"github.com/google/splice/appengine/server"
	"github.com/google/splice/models"
	"github.com/google/splice/shared/certs"
	metadata "github.com/google/splice/shared/crypto"
	"github.com/google/splice/shared/provisioning"
)

const (
	pollMaxRetries = 100
)

var (
	myName       = flag.String("name", "", "The requested hostname.")
	pollInterval = flag.Int("poll_interval", 30, "Time in seconds between server polling attempts.")
	serverAddr   = flag.String("server", "", "The address of the Splice app server.")
	reallyJoin   = flag.Bool("really_join", false, "Really join the local machine if the request succeeds.")
	unattended   = flag.Bool("unattended", false, "Runs in unattended mode. A valid certificate is required for unattended mode.")
	verbose      = flag.Bool("verbose", false, "Give more verbose output.")

	// GCE
	isGCE = flag.Bool("gce", false, "Include GCE Metadata.")

	// Auth flags
	username = flag.String("user_name", "", "User name for login.")

	// Encryption flags
	certIssuers       = flag.String("cert_issuer", "", "Comma delimited list of client certificate issuers to be looked up for metadata encryption.")
	certIntermediates = flag.String("cert_intermediate", "", "Comma delimited list of additional intermediate certificate issuers.")
	certContainer     = flag.String("cert_container", "", "The client certificate CNG container name.")
	encrypt           = flag.Bool("encrypt", true, "Encrypt all metadata in transit.")
	generateCert      = flag.Bool("generate_cert", false, "Generate a self-signed certificate for encryption.")

	issuers, intermediates []string
	decrypter              crypto.Decrypter
	cert                   []byte
)

type client interface {
	Do(*http.Request) (*http.Response, error)
}

// post posts JSON data to the splice application server
func post(c client, msg interface{}, addr string) (*models.Response, error) {
	body, err := json.Marshal(msg)
	if err != nil {
		return nil, fmt.Errorf("error marshalling message(%v): %v", msg, err)
	}

	req, err := http.NewRequest("POST", addr, bytes.NewBuffer(body))
	if err != nil {
		return nil, fmt.Errorf("error composing post request: %v", err)
	}
	req.Header.Set("Content-Type", "application/json")

	res, err := c.Do(req)
	if err != nil {
		return nil, fmt.Errorf("error executing post request: %v", err)
	}
	defer res.Body.Close()

	if res.StatusCode < http.StatusOK || res.StatusCode > http.StatusIMUsed {
		return nil, fmt.Errorf("invalid response code received for request: %d", res.StatusCode)
	}

	respBody, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return nil, fmt.Errorf("error reading response body: %v", err)
	}

	resp := &models.Response{}
	if err := json.Unmarshal(respBody, resp); err != nil {
		msg := fmt.Sprintf("json.Unmarshal returned: %v\n\nResponse Body: %s", err, respBody)
		if *verbose {
			msg = fmt.Sprintf("%s (body: %s)", msg, respBody)
		}
		return nil, fmt.Errorf(msg)
	}
	return resp, nil
}

// request posts to the splice request endpoint and returns the
// requestID if successful or an error.
func request(c client, clientID string) (string, error) {
	model := &models.ClientRequest{
		Hostname: *myName,
		ClientID: clientID,
	}
	endpoint := *serverAddr + "/request"
	if *unattended {
		endpoint = endpoint + "-unattended"
	}

	if *isGCE {
		model.GCEMetadata.Audience = endpoint
		if err := model.GCEMetadata.Read(); err != nil {
			return "", fmt.Errorf("error reading GCE metadata: %v", err)
		}
	}
	if *encrypt {
		model.ClientCert = cert
	}

	resp, err := post(c, model, endpoint)
	if err != nil {
		return "", fmt.Errorf("post(%s, %q) returned %v", model, endpoint, err)
	}
	if resp.ErrorCode != server.StatusSuccess {
		return "", fmt.Errorf("post to %s returned: %v %d %s", endpoint, resp.Status, resp.ErrorCode, resp.ResponseData)
	}

	if *verbose {
		fmt.Printf("Request ID: %s\n", resp.RequestID)
	}
	return resp.RequestID, nil
}

func resultPoll(c client, reqID string, clientID string) (*models.Response, error) {
	status := &models.StatusQuery{
		RequestID: reqID,
		ClientID:  clientID,
	}

	endpoint := *serverAddr + "/result"
	if *unattended {
		endpoint = endpoint + "-unattended"
	}

	for i := 0; i < pollMaxRetries; i++ {
		time.Sleep(time.Duration(*pollInterval) * time.Second)
		resp, err := post(c, status, endpoint)
		if err != nil {
			return nil, fmt.Errorf("post: %v", err)
		}
		fmt.Println("Checking for a result...")
		if *verbose {
			fmt.Printf("%v\n", resp)
		}
		if resp.ErrorCode == server.StatusInvalidCertError {
			// Retry lookups for Datastores to allow eventual consistency.
			fmt.Println("Result not found or invalid cert, retrying...")
			continue
		}
		if resp.ErrorCode != server.StatusSuccess {
			return resp, fmt.Errorf("server processing failed, request:%s, id:%s, status:%d %v, data: %s", reqID, clientID, resp.ErrorCode, resp.Status, resp.ResponseData)
		}
		if resp.Status == models.RequestStatusFailed {
			return resp, fmt.Errorf("domain join failed, request:%s, id:%s, status:%d %v, data: %s", reqID, clientID, resp.ErrorCode, resp.Status, resp.ResponseData)
		}
		if (resp.Status == models.RequestStatusCompleted) && (resp.Hostname != *myName) {
			fmt.Printf("Result returned is for a different host, got %s, want %s.\n", resp.Hostname, *myName)
			return resp, nil
		}
		if (resp.Status == models.RequestStatusCompleted) && resp.ResponseData != nil {
			fmt.Printf("Successfully retrieved result for host %s.\n", resp.Hostname)
			return resp, nil
		}
	}

	return nil, fmt.Errorf("retry limit (%d) exceeded", pollMaxRetries)
}

func getHostCert() error {
	var err error

	store, err := certtostore.OpenWinCertStore(certtostore.ProviderMSSoftware, *certContainer, issuers, intermediates, false)
	if err != nil {
		return fmt.Errorf("OpenWinCertStore: %v", err)
	}

	if err = getHostKey(store); err != nil {
		log.Printf("Trying '%s' instead...", certtostore.ProviderMSPlatform)
	}
	if err == nil {
		return nil
	}

	store, err = certtostore.OpenWinCertStore(certtostore.ProviderMSPlatform, *certContainer, issuers, intermediates, false)
	if err != nil {
		return fmt.Errorf("OpenWinCertStore: %v", err)
	}

	return getHostKey(store)
}

func getHostKey(store *certtostore.WinCertStore) error {
	var err error

	decrypter, err = store.Key()
	if err != nil {
		log.Printf("A private key was not found in '%s'.", store.ProvName)
		return err
	}

	c, err := store.Cert()
	if err != nil {
		return fmt.Errorf("cert: %v", err)
	}
	cert = c.Raw

	return nil
}

func checkFlags() error {
	switch {
	case *myName == "":
		return errors.New("the name flag is required")
	case *serverAddr == "":
		return errors.New("the server flag is required")
	case *encrypt && !*generateCert && *certIssuers == "":
		return errors.New("-encrypt requires either -generate_cert or -cert_issuer")
	case *encrypt && *generateCert && *certIssuers != "":
		return errors.New("-encrypt is not supported with both -generate_cert and -cert_issuer")
	}

	if !strings.HasPrefix(*serverAddr, "http") {
		*serverAddr = "https://" + *serverAddr
	}

	if *certIssuers != "" {
		issuers = strings.Split(*certIssuers, ",")
	}

	if *certIntermediates != "" {
		intermediates = strings.Split(*certIntermediates, ",")
	}

	return nil
}

func main() {
	var err error

	if err = checkFlags(); err != nil {
		log.Fatal(err.Error())
	}

	if len(issuers) >= 1 {
		err = getHostCert()
		if err != nil || cert == nil || decrypter == nil {
			log.Fatalf("error locating client certificate for issuers '%v': %v", issuers, err)
		}
	}

	if *encrypt {
		if *generateCert {
			notBefore := time.Now().Add(-1 * time.Hour)
			notAfter := time.Now().Add(time.Hour * 24 * 365 * 1)
			cert, decrypter, err = certs.GenerateSelfSignedCert(*myName, notBefore, notAfter)
			if err != nil {
				log.Fatalf("error generating self-signed certificate: %v", err)
			}
		}
		fmt.Println("Requesting encryption with public key.")
	} else {
		fmt.Println("Not requesting encryption.")
	}

	// UUID is the fallback clientID when cert lookups
	// are not enabled.
	var clientID string
	if len(issuers) >= 1 {
		// The SHA256 hash of the cert is used server side for client verification when
		// certificate verification is enabled.
		fingerprintRaw := sha256.Sum256(cert)
		clientID = strings.TrimSuffix(base64.StdEncoding.EncodeToString(fingerprintRaw[:]), "=")
	} else {
		computer, err := certtostore.CompProdInfo()
		if err != nil {
			log.Fatalf("certtostore.CompInfo returned %v", err)
		}
		clientID = computer.UUID
	}

	var c client
	if !*unattended {
		c, err = appclient.Connect(*serverAddr, *username)
		if err != nil {
			log.Fatalf("SSO error: %v", err)
		}
	} else {
		c, err = appclient.TLSClient(cert, decrypter)
		if err != nil {
			log.Fatalf("error during TLS client setup: %v", err)
		}
	}

	reqID, err := request(c, clientID)
	if err != nil {
		log.Fatalf("request: %v", err)
	}
	fmt.Println("Successfully submitted join request.")

	resp, err := resultPoll(c, reqID, clientID)
	if err != nil {
		log.Fatalf("resultPoll: %v\n", err)
	}
	meta := metadata.Metadata{
		Data:   resp.ResponseData,
		AESKey: resp.ResponseKey,
		Nonce:  resp.CipherNonce,
	}

	if *encrypt {
		meta.Data, err = meta.Decrypt(decrypter)
		if err != nil {
			log.Fatalf("error decrypting metadata: %v", err)
		}
	}

	if *reallyJoin {
		if err := provisioning.OfflineJoin(meta.Data); err != nil {
			log.Fatalf("error applying join metadata to host: %v", err)
		}
		fmt.Println("Successfully joined the domain! Reboot required to complete domain join.")
	} else {
		fmt.Println("Metadata received but skipping application without -really_join")
	}
}
