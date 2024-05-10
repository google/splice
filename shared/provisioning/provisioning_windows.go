//go:build windows
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

//Package provisioning provides Windows-specific functionality for joining hosts to a domain.
package provisioning

import (
	"errors"
	"fmt"
	"syscall"
	"unsafe"
)

// test
const (
	buffSize = 3000

	// Flags from NetProvisionComputerAccount
	netsetupProvisionReuseAccount = 0x00000002
	// Flags from NetRequestOfflineDomainJoin
	netsetupProvisionOnlineCaller = 0x40000000

	// Known error codes
	errnoERROR_ACCESS_DENIED         = 5
	errnoERROR_NOT_SUPPORTED         = 50
	errnoERROR_INVALID_PARAMETER     = 87
	errnoERROR_INVALID_DOMAIN_ROLE   = 1354
	errnoERROR_NO_SUCH_DOMAIN        = 1355
	errnoRPC_S_CALL_IN_PROGRESS      = 1791
	errnoRPC_S_PROTSEQ_NOT_SUPPORTED = 1703
	errnoNERR_DS8DCRequired          = 2720
	errnoNERR_LDAPCapableDCRequired  = 2721
	errnoNERR_UserExists             = 2224
	errnoNERR_WkstaNotStarted        = 2138
)

var (
	// ErrAccessDenied indicates an access-related error
	ErrAccessDenied = errors.New("access is denied")
	// ErrExists indicates that the specified account already exists
	ErrExists = errors.New("the account already exists in the domain and reuse is not enabled")
	// ErrInvalidParameter indicates that an invalid parameter was provided to the API
	ErrInvalidParameter = errors.New("a parameter is incorrect")
	// ErrNoSuchDomain indicates that an invalid domain was specified
	ErrNoSuchDomain = errors.New("the specified domain does not exist")
	// ErrNotSupported indicates that the request is not supported
	ErrNotSupported = errors.New("the request is not supported")
	// ErrWorkstationSvc indicates that the workstation service has not been started
	ErrWorkstationSvc = errors.New("the Workstation service has not been started")
)

// errnoErr converts errno return values from api calls into usable errors
func errnoErr(e syscall.Errno) error {
	switch e {
	case errnoERROR_ACCESS_DENIED:
		return ErrAccessDenied
	case errnoERROR_NOT_SUPPORTED:
		return ErrNotSupported
	case errnoERROR_INVALID_PARAMETER:
		return ErrInvalidParameter
	case errnoERROR_NO_SUCH_DOMAIN:
		return ErrNoSuchDomain
	case errnoNERR_UserExists:
		return ErrExists
	case errnoNERR_WkstaNotStarted:
		return ErrWorkstationSvc
	}
	return e
}

var (
	netapi32 = syscall.MustLoadDLL("Netapi32.dll")

	// Ref: https://msdn.microsoft.com/en-us/library/dd815228(v=vs.85).aspx
	netProvisionComputerAccount = netapi32.MustFindProc("NetProvisionComputerAccount")
	// Ref: https://msdn.microsoft.com/en-us/library/dd815229(v=vs.85).aspx
	netRequestOfflineDomainJoin = netapi32.MustFindProc("NetRequestOfflineDomainJoin")
)

// OfflineJoin uses provisioning metadata to conduct an offline domain join.
func OfflineJoin(metadata []byte) error {
	ptrWinders, err := syscall.UTF16PtrFromString("C:\\Windows")
	if err != nil {
		return err
	}

	dataSize := uint32(len(metadata))
	r, _, err := netRequestOfflineDomainJoin.Call(
		uintptr(unsafe.Pointer(&metadata[0])),  // _In_ BYTE *pProvisionBinData,
		uintptr(dataSize),                      // _In_ DWORD   cbProvisionBinDataSize,
		uintptr(netsetupProvisionOnlineCaller), // _In_ DWORD   dwOptions,
		uintptr(unsafe.Pointer(ptrWinders)),    // _In_ LPCWSTR lpWindowsPath
	)
	if r != 0 {
		err = fmt.Errorf("NetRequestOfflineDomainJoin failed to return successfully (%x,%v)", r, err)
		return err
	}

	return nil
}

// TextData produces provisioning data in string form for embedding in an unattended setup answer file.
func TextData(hostname, domain string, reuse, djoinCompat bool) ([]byte, error) {
	buff := make([]byte, buffSize)

	result := []byte{}

	if djoinCompat == true {
		result = append(result, 255, 254) // UTF16 BOM
	}

	ptrDomain, err := syscall.UTF16PtrFromString(domain)
	if err != nil {
		return result, err
	}
	ptrHostname, err := syscall.UTF16PtrFromString(hostname)
	if err != nil {
		return result, err
	}

	var dwOptions uintptr
	if reuse {
		dwOptions = netsetupProvisionReuseAccount
	}

	r, _, err := netProvisionComputerAccount.Call(
		uintptr(unsafe.Pointer(ptrDomain)),   //_In_      LPCWSTR lpDomain,
		uintptr(unsafe.Pointer(ptrHostname)), //_In_      LPCWSTR lpMachineName,
		0,                                    //_In_opt_  LPCWSTR lpMachineAccountOU,
		0,                                    //_In_opt_  LPCWSTR lpDcName,
		dwOptions,                            //_In_      DWORD   dwOptions,
		0,                                    //_Out_opt_ PBYTE   *pProvisionBinData,
		0,                                    //_Out_opt_ DWORD   *pdwProvisionBinDataSize,
		uintptr(unsafe.Pointer(&buff)),       //_Out_opt_ LPWSTR  *pProvisionTextData
	)
	if r != 0 {
		return result, errnoErr(syscall.Errno(r))
	}

	for i := range buff {
		if buff[i] == 0 && buff[i+1] == 0 {
			result = append(result, 0, 0)
			break
		}
		result = append(result, buff[i])
	}

	return result, nil
}

// BinData produces provisioning data in binary form for use with the NetRequestOfflineDomainJoin function.
func BinData(hostname string, domain string, reuse bool) ([]byte, error) {
	var binSize uint32
	buff := make([]byte, buffSize)
	ptrDomain, err := syscall.UTF16PtrFromString(domain)
	if err != nil {
		return buff, err
	}
	ptrHostname, err := syscall.UTF16PtrFromString(hostname)
	if err != nil {
		return buff, err
	}

	var dwOptions uintptr
	if reuse {
		dwOptions = netsetupProvisionReuseAccount
	}

	r, _, err := netProvisionComputerAccount.Call(
		uintptr(unsafe.Pointer(ptrDomain)),   //_In_      LPCWSTR lpDomain,
		uintptr(unsafe.Pointer(ptrHostname)), //_In_      LPCWSTR lpMachineName,
		0,                                    //_In_opt_  LPCWSTR lpMachineAccountOU,
		0,                                    //_In_opt_  LPCWSTR lpDcName,
		dwOptions,                            //_In_      DWORD   dwOptions,
		uintptr(unsafe.Pointer(&buff)),       //_Out_opt_ PBYTE   *pProvisionBinData,
		uintptr(unsafe.Pointer(&binSize)),    //_Out_opt_ DWORD   *pdwProvisionBinDataSize,
		0,                                    //_Out_opt_ LPWSTR  *pProvisionTextData
	)
	if r != 0 {
		return buff[:binSize], errnoErr(syscall.Errno(r))
	}

	return buff[:binSize], nil
}
