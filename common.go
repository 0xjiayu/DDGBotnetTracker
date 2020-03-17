package ddg_tracker

import (
	"strings"
	"sync"
)

/**
Some Common components
*/

// The latest cc server IPs
// store the ip list to file :ccip.list
type CCHostList struct {
	Hosts []string
	L     sync.Mutex
}

// Current samples info, include md5 and file path
type SampleInfo struct {
	MD5List  []string
	FileList []string
	L        sync.Mutex
}

// Add new cc Host to ccHostList
func (hostList *CCHostList) Append(host string) {
	hostList.L.Lock()
	defer hostList.L.Unlock()

	var cmpRes int = 1
	for _, ipStr := range hostList.Hosts {
		cmpRes = strings.Compare(host, ipStr)
		if cmpRes == 0 {
			return
		}
	}
	hostList.Hosts = append(hostList.Hosts, host)
}

// Add new md5 to sampleInfo.Md5List
func (samples *SampleInfo) AddMD5(newMd5 string) {
	samples.L.Lock()
	defer samples.L.Unlock()

	var cmpRes int = 1
	for _, md5Str := range samples.MD5List {
		cmpRes = strings.Compare(md5Str, newMd5)
		if cmpRes == 0 {
			return
		}
	}
	samples.MD5List = append(samples.MD5List, newMd5)
}

// Add new file path to sampleInfo.FileList
func (samples *SampleInfo) AddFile(newFile string) {
	samples.L.Lock()
	defer samples.L.Unlock()

	var cmpRes int = 1
	for _, filePath := range samples.FileList {
		cmpRes = strings.Compare(filePath, newFile)
		if cmpRes == 0 {
			return
		}
	}
	samples.FileList = append(samples.FileList, newFile)
}

// Check if new md5 or new file(sample download URL)
// returns:
// 		- 0: Both new
//		- 1: Only New md5
// 		- 2: Only New file
// 		- 3: Nothing new
func (samples *SampleInfo) IsNew(newMd5, newFile string) uint8 {
	var chkCode uint8 = 0
	var md5ChkCode uint8 = 0
	var fileChkCode uint8 = 0
	var cmpRes int = 1

	samples.L.Lock()
	defer samples.L.Unlock()

	for _, md5Str := range samples.MD5List {
		cmpRes = strings.Compare(md5Str, newMd5)
		if cmpRes == 0 {
			md5ChkCode = 1
			break
		}
	}

	cmpRes = 1
	for _, filePath := range samples.FileList {
		cmpRes = strings.Compare(filePath, newFile)
		if cmpRes == 0 {
			fileChkCode = 2
			break
		}
	}

	chkCode = md5ChkCode + fileChkCode
	return chkCode
}
