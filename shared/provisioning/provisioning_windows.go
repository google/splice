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
	"fmt"
	"syscall"
	"unsafe"
)

const (
	buffSize = 3000

	// Flags from NetProvisionComputerAccount
	netsetupProvisionReuseAccount = 0x00000002
	// Flags from NetRequestOfflineDomainJoin
	netsetupProvisionOnlineCaller = 0x40000000
)

var (
	netapi32 = syscall.MustLoadDLL("Netapi32.dll")

	// Ref: https://msdn.microsoft.com/en-us/library/dd815228(v=vs.85).aspx
	netProvisionComputerAccount = netapi32.MustFindProc("NetProvisionComputerAccount")
	// Ref: https://msdn.microsoft.com/en-us/library/dd815229(v=vs.85).aspx
	netRequestOfflineDomainJoin = netapi32.MustFindProc("NetRequestOfflineDomainJoin")

	// Return codes for netRequestOfflineDomainjoin
	netRequestCodes = map[syscall.Errno]string{
		5:    "ERROR_ACCESS_DENIED",
		50:   "ERROR_NOT_SUPPORTED",
		87:   "ERROR_INVALID_PARAMETER",
		1354: "ERROR_INVALID_DOMAIN_ROLE",
		1355: "ERROR_NO_SUCH_DOMAIN",
		1791: "RPC_S_CALL_IN_PROGRESS",
		1703: "RPC_S_PROTSEQ_NOT_SUPPORTED",
		2720: "NERR_DS8DCRequired",
		2721: "NERR_LDAPCapableDCRequired",
		2224: "NERR_UserExists",
		2138: "NERR_WkstaNotStarted",
	}
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
		// The detailed error code is found only in syscall.Errno
		if errno, ok := err.(syscall.Errno); ok {
			err = fmt.Errorf("netProvisionComputerAccount failed with %d(%s)", errno, netRequestCodes[errno])
			return result, err
		}
		err = fmt.Errorf("netProvisionComputerAccount failed with unknown error (%x,%v)", r, err)
		return result, err
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
		// The detailed error code is found only in syscall.Errno
		if errno, ok := err.(syscall.Errno); ok {
			err = fmt.Errorf("netProvisionComputerAccount failed with %s(%d)", netRequestCodes[errno], errno)
			return buff[:binSize], err
		}
		err = fmt.Errorf("netProvisionComputerAccount failed with unknown error (%x,%v)", r, err)
		return buff[:binSize], err
	}

	return buff[:binSize], nil
}
