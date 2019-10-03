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
	"fmt"

	"golang.org/x/sys/windows/registry"
)

const (
	rootKey = `SOFTWARE\Splice\spliced`
)

type appcfg struct {
	Domain      string
	Instance    string
	ProjectID   string
	Topic       string
	EncryptBlob bool
	VerifyCert  bool
	CaURL       string
	CaURLPath   string
	CaOrg       string
	RootsPath   string
	PermitReuse bool
}

func getConfig() (appcfg, error) {
	var conf appcfg
	k, err := registry.OpenKey(registry.LOCAL_MACHINE, rootKey, registry.ALL_ACCESS)
	if err != nil {
		return conf, fmt.Errorf("getConfig: opening root key %s failed with %v", rootKey, err)
	}

	conf.Domain, _, err = k.GetStringValue("domain")
	if err != nil {
		return conf, fmt.Errorf("getConfig: reading domain value failed with %v", err)
	}

	conf.ProjectID, _, err = k.GetStringValue("project")
	if err != nil {
		return conf, fmt.Errorf("getConfig: reading project value failed with %v", err)
	}

	conf.Instance, _, err = k.GetStringValue("instance")
	if err != nil {
		return conf, fmt.Errorf("getConfig: reading instance value failed with %v", err)
	}

	conf.Topic, _, err = k.GetStringValue("topic")
	if err != nil {
		return conf, fmt.Errorf("getConfig: reading topic value failed with %v", err)
	}

	// Optional values
	eb, _, err := k.GetIntegerValue("encrypt_blob")
	if err != nil || eb != 0 {
		conf.EncryptBlob = true
	} else {
		conf.EncryptBlob = false
	}

	vc, _, err := k.GetIntegerValue("verify_certs")
	if err != nil || vc != 0 {
		conf.VerifyCert = true
	} else {
		conf.VerifyCert = false
	}

	conf.CaURL, _, err = k.GetStringValue("ca_root_url")
	if err != nil && conf.VerifyCert {
		return conf, fmt.Errorf("getConfig: verify_certs is enabled, but ca_certs_url could not be read: %v", err)
	}

	conf.CaURLPath, _, err = k.GetStringValue("ca_cert_path")
	if err != nil {
		conf.CaURLPath = ""
	}

	conf.CaOrg, _, err = k.GetStringValue("ca_cert_org")
	if err != nil {
		conf.CaOrg = ""
	}

	conf.RootsPath, _, err = k.GetStringValue("roots_path")
	if err != nil {
		conf.RootsPath = ""
	}

	pr, _, err := k.GetIntegerValue("permit_reuse")
	if err != nil || pr != 1 {
		conf.PermitReuse = false
	} else {
		conf.PermitReuse = true
	}
	return conf, nil
}

// setDwordValue adds or updates a REG_DWORD value.
func setDWordValue(name string, value uint32) error {
	k, _, err := registry.CreateKey(registry.LOCAL_MACHINE, rootKey, registry.ALL_ACCESS)
	if err != nil {
		return fmt.Errorf("setDWordValue: creating root key %s failed with %v", rootKey, err)
	}
	if err := k.SetDWordValue(name, value); err != nil {
		return fmt.Errorf("setDWordValue: updating key %s with %d failed due to %v", name, value, err)
	}

	return nil
}

// setStringValue adds or updates a REG_SZ value.
func setStringValue(name string, value string) error {
	k, _, err := registry.CreateKey(registry.LOCAL_MACHINE, rootKey, registry.ALL_ACCESS)
	if err != nil {
		return fmt.Errorf("setStringValue: creating root key %s failed with %v", rootKey, err)
	}
	if err := k.SetStringValue(name, value); err != nil {
		return fmt.Errorf("setStringValue: updating key %s with %s failed due to %v", name, value, err)
	}

	return nil
}
