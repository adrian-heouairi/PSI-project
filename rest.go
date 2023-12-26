package main

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"
)

func httpGet(url string) (*http.Response, []byte, error) {
	resp, err := http.Get(url)
	if err != nil {
		return nil, nil, err
	}

	bodyAsByteSlice, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, nil, err
	}

	return resp, bodyAsByteSlice, nil
}

func getPeers() error {
	// TODO Return something
	_, bodyAsByteSlice, err := httpGet(SERVER_ADDRESS + PEERS_PATH)
	if err != nil {
		return err
	}

	fmt.Println(string(bodyAsByteSlice))

	return nil
}

func getAdressesOfPeer(peerName string) ([]string, error) {
	resp, bodyAsByteSlice, err := httpGet(SERVER_ADDRESS + PEERS_PATH + "/" + peerName + "/addresses")
	if err != nil {
		return nil, err
	}

	if resp.StatusCode == 404 {
		return nil, fmt.Errorf(peerName + " is not known by server")
	}

	return strings.Split(string(bodyAsByteSlice), "\n"), nil
}

func getRootOfPeer(peerName string) ([]byte, error) {
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
