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

package main

import (
	"flag"
	"fmt"
)

var (
	// Flags
	cFlags = flag.NewFlagSet("configure", flag.ContinueOnError)

	fDomain   = cFlags.String("domain", "", "The fully qualified Active Directory domain (eg domain.example.com).")
	fInstance = cFlags.String("instance", "", "A unique name for this host or instance.")
	fProject  = cFlags.String("project", "", "The Google Cloud project name.")
	fTopic    = cFlags.String("topic", "", "The Pub/Sub topic name this daemon should subscribe to.")

	fEncryptBlob         = cFlags.Bool("encrypt_blob", true, "Require metadata blob encryption.")
	fVerifyCerts         = cFlags.Bool("verify_certs", false, "Require that the certificate passed with requests pass verification checks.")
	fVerifyCertsRootURL  = cFlags.String("ca_root_url", "", "The base URL for the root certificate(s) you wish to lookup during verification. Required if verify_certs=true.")
	fVerifyCertsRootPath = cFlags.String("ca_cert_path", "", "The path under ca_root_url at which the root certificate can be found for certificate validation. Optional if verify_certs=true.")
	fVerifyCertsCAOrg    = cFlags.String("ca_cert_org", "", "The expected issuing organization for the root certificate. Optional if verify_certs=true.")
	fRootsPath           = cFlags.String("roots_path", "", "The path to a pemfile containing the roots to be used for certificate verification. Optional if verify_certs=true.")
	fPermitReuse         = cFlags.Bool("permit_reuse", false, "Permit SpliceD to attempt to reuse existing domain accounts.")
)

func boolToUint32(b bool) uint32 {
	if b {
		return 1
	}
	return 0
}

// Update updates the app configuration with new settings from the command line.
func Update(args []string) error {
	cFlags.Parse(args)
	if *fVerifyCerts && *fVerifyCertsRootURL == "" && *fRootsPath == "" {
		return fmt.Errorf("ca_root_url or roots_path is required when verify_certs=%t", *fVerifyCerts)
	}

	if !*fVerifyCerts && (*fVerifyCertsRootURL != "" || *fVerifyCertsRootPath != "" || *fVerifyCertsCAOrg != "") {
		return fmt.Errorf("ca_root_url, ca_cert_path and ca_cert_org are not required when verify_certs=%t", *fVerifyCerts)
	}

	if *fDomain != "" {
		if err := setStringValue("domain", *fDomain); err != nil {
			return err
		}
	}
	if *fInstance != "" {
		if err := setStringValue("instance", *fInstance); err != nil {
			return err
		}
	}
	if *fProject != "" {
		if err := setStringValue("project", *fProject); err != nil {
			return err
		}
	}
	if *fTopic != "" {
		if err := setStringValue("topic", *fTopic); err != nil {
			return err
		}
	}

	if err := setDWordValue("encrypt_blob", boolToUint32(*fEncryptBlob)); err != nil {
		return err
	}

	if err := setDWordValue("verify_certs", boolToUint32(*fVerifyCerts)); err != nil {
		return err
	}

	if err := setStringValue("ca_root_url", *fVerifyCertsRootURL); err != nil {
		return err
	}

	if err := setStringValue("ca_cert_path", *fVerifyCertsRootPath); err != nil {
		return err
	}

	if err := setStringValue("ca_cert_org", *fVerifyCertsCAOrg); err != nil {
		return err
	}

	if *fRootsPath != "" {
		if err := setStringValue("roots_path", *fRootsPath); err != nil {
			return err
		}
	}

	return setDWordValue("permit_reuse", boolToUint32(*fPermitReuse))
}
