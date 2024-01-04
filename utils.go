package main

import (
	"container/list"
	"crypto/sha256"
	"encoding/binary"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"regexp"
	"strings"
	"sync"
)

// Wraps Mkdir func call
// - path: path of the directory to be created
// Returns: error if the user has not writing right in working directory
func mkdirP(path string) error {
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

func addrIsInSlice(slice []*net.UDPAddr, addr *net.UDPAddr) bool {
	for _, a := range slice {
		if compareUDPAddr(addr, a) {
			return true
		}
	}
	return false
}

func appendAddressesIfNotPresent(slice []*net.UDPAddr, addresses []*net.UDPAddr) []*net.UDPAddr {
	for _, a := range addresses {
		if !addrIsInSlice(slice, a) {
			slice = append(slice, a)
		}
	}

	return slice
}

// We assume that slice will never be modified after calling this
func byteSliceToUDPAddr(slice []byte) (*net.UDPAddr, error) {
	if len(slice) == UDP_V4_SOCKET_SIZE {
		port := binary.BigEndian.Uint16(slice[IPV4_SIZE:])
		return &net.UDPAddr{IP: slice[:IPV4_SIZE], Port: int(port)}, nil
	} else if len(slice) == UDP_V6_SOCKET_SIZE {
		panic("IPv6 not supported")
	} else {
		return nil, fmt.Errorf("invalid slice length")
	}
}

func udpAddrToByteSlice(addr *net.UDPAddr) []byte {
	slice := []byte{}

	var addrAsByteSlice []byte = addr.IP.To4()

	if addrAsByteSlice == nil {
		panic("IPv6 not supported")
	}

	slice = append(slice, addrAsByteSlice...)

	var portAsByteSlice []byte = make([]byte, 2)
	binary.BigEndian.PutUint16(portAsByteSlice, uint16(addr.Port))

	return append(slice, portAsByteSlice...)
}

// Assumes that str is no more than 32 bytes
func stringToZeroPaddedByteSlice(str string) []byte {
	res := []byte(str)
	res = append(res, make([]byte, FILENAME_MAX_SIZE-len(res))...)
	return res
}

// Removes the trailing zeroes from name.
// - name: from which to remove \0s
// - Returns: a valid string or error if data is not valid
func zeroPaddedByteSliceToString(name []byte) string {
	i := 0
	for name[i] != 0 {
		i++
	}

	if i == 0 {
		LOGGING_FUNC("empty filenames are not allowed")
		return ""
	}

	return string(name[:i])
}

func getHashOfByteSlice(slice []byte) []byte {
	hasher := sha256.New()
	hasher.Write(slice)
	return hasher.Sum(nil)
}

func grep(pattern string, content string) bool {
	res, _ := regexp.MatchString(pattern, content)
	return res
}

func splitLine(line string) []string {
	line = strings.TrimSpace(line)
	line = replaceAllRegexBy(line, " +", " ")
	// strings.Split("", " ") will return []string{""} instead of []string{}
	return strings.Split(line, " ")
}

func getNbOfChunks(path string) (int, error) {
	fi, err := os.Stat(path)
	if err != nil {
		return -1, err
	}
	res := int(fi.Size()) / CHUNK_MAX_SIZE
	if fi.Size()%CHUNK_MAX_SIZE != 0 {
		res++
	}
	return res, nil
}
