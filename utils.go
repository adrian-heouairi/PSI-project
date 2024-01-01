package main

import (
	"container/list"
	"io"
	"net"
	"net/http"
	"os"
	"regexp"
	"sync"
)

// Wraps Mkdir func call
// -path: path of the directory to be created
// Returns: error if the user has not writing right in working directory
func mkdir(path string) error {
	_, err := os.Stat(path)
	if os.IsNotExist(err) {
		err = os.MkdirAll(path, 0755)
		if err != nil {
			return err
		}
	} else {
		return err
	}

	return nil
}

// Writes the given chunk to the specified path.
// -path: represnts the file to write in
// Retruns: error if file does not exists or we can not write in
func writeChunk(path string, chunk []byte) error {
	file, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_APPEND, 0644)
	if err != nil {
		return err
	}
	defer file.Close()

	_, err = file.Write(chunk)
	if err != nil {
		return err
	}

	return nil
}

// Appends elem to list concurrency safe.
// -list: the list in which to add
// -mutex: to protect critical section
// -elem: to be added
func threadSafeAppendToList(list *list.List, mutex *sync.RWMutex, elem any) {
	mutex.Lock()
	defer mutex.Unlock()

	list.PushBack(elem)
}

// Compares to UDP addresses.
// -first: the first address
// -second: the second address
// Returns: true if addresses are equal false otherwise
func compareUDPAddr(first *net.UDPAddr, second *net.UDPAddr) bool {
	return first.String() == second.String()
}

// Wrapper of http.Get
// - url: textual representation of the url to be visited
// Returns: - the http Response
//   - http repsonse body as byte slice
//   - error if something goes wrong nil otherwise
func httpGet(url string) (*http.Response, []byte, error) {
	resp, err := http.Get(url)
	if err != nil {
		return nil, nil, err
	}

	bodyAsByteSlice, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, nil, err
	}

	return resp, bodyAsByteSlice, nil
}

// TODO Addresses will never be removed but if none work we call this,
// if size after call is bigger test the new addresses
// If none work, give up as peer doesn't exist
func getAddressOfPeer(peerName string) (*net.UDPAddr, error) {
	peersMutex.RLock()
	_, found := peers[peerName]
	peersMutex.RUnlock()
	if !found {
		createKeyValuePairInPeers(peerName)
	}

	var addrToReturn *net.UDPAddr

	peersMutex.RLock()
	peerAddresses, _ := peers[peerName]
	if len(peerAddresses) > 0 {
		addrToReturn = peerAddresses[0]
	}
	peersMutex.RUnlock()

	if addrToReturn != nil {
		return addrToReturn, nil
	}

	restPeerAddresses, err := restGetAddressesOfPeer(peerName, false)
	if err != nil {
		return nil, err
	}

	for _, a := range restPeerAddresses {
		addAddrToPeers(peerName, a)
	}

	return restPeerAddresses[0], nil
}

func replaceAllRegexBy(src, regex, replacement string) string {
	pattern := regexp.MustCompile(regex)
	return pattern.ReplaceAllString(src, replacement)
}

func removeTrailingSlash(path string) string {
	if path[len(path)-1] == '/' {
		return path[:len(path)-1]
	}
	return path
}

func getKeys(m map[string][]byte) []string {
	res := make([]string, 0)
	for key := range m {
		res = append(res, key)
	}
	return res
}

func stringSliceToAnySlice(slc []string) []any {
	var res []any = make([]any, 0)
	for _, elt := range slc {
		res = append(res, elt)
	}
	return res
}
