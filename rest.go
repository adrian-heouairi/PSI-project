package main

import (
	"fmt"
	"io/ioutil"
	"net"
	"net/http"
	"strings"
)

// Wrapper of htt.Get
// - url: textual representation of the url to be visited
// Returns: - the http Response
//          - http repsonse body as byte slice
//          - error if something goes wrong nil otherwise
func httpGet(url string) (*http.Response, []byte, error) {
	resp, err := http.Get(url)
	if err != nil {
		return nil, nil, err
	};

	bodyAsByteSlice, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, nil, err
	}

	return resp, bodyAsByteSlice, nil
}

// Displays connected peers.
// Returns: -error if server is not available
func getPeers() error {
	// TODO Return something
	_, bodyAsByteSlice, err := httpGet(SERVER_ADDRESS + PEERS_PATH)
	if err != nil {
		return err
	}

	fmt.Println(string(bodyAsByteSlice))

	return nil
}

// Gives the addresses of the given peer.
// - peerName: the peer whose addresses we want
// Returns: - a slice with the peer addresses
//          - error if peer was not found
func getAdressesOfPeer(peerName string) ([]*net.UDPAddr, error) {
	resp, bodyAsByteSlice, err := httpGet(SERVER_ADDRESS + PEERS_PATH + "/" + peerName + "/addresses")
	if err != nil {
		return nil, err
	}

	if resp.StatusCode == 404 {
		return nil, fmt.Errorf(peerName + " is not known by server")
	}

    addrAsStrings := strings.Split(string(bodyAsByteSlice), "\n") 
    res := []*net.UDPAddr{}
    for _, s := range addrAsStrings {
        addr, err := net.ResolveUDPAddr("udp",s)
        if err != nil {
            return nil, err
        }
        res = append(res, addr)
    }
    return res, nil
}

// Gives the hash of the peer's root
// - peerName: the peer whose root we want
// Returns: - the root hash
//          - error if peer does not exist or the main server is not available
func getRootOfPeer(peerName string) ([]byte, error) {
    //TODO : replace /root by constant
	resp, bodyAsByteSlice, err := httpGet(SERVER_ADDRESS + PEERS_PATH + "/" + peerName + "/root")
	if err != nil {
		return nil, err
	}

	if resp.StatusCode == 204 {
		// TODO Return the hash of the empty string?
		return nil, fmt.Errorf(peerName + " has not declared a root yet")
	} else if resp.StatusCode == 404 {
		return nil, fmt.Errorf(peerName + "is not known by server")
	}

	return bodyAsByteSlice, nil
}
