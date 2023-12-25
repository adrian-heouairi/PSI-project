package main

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"
)

func httpGet(url string) (*http.Response, []byte) {
	resp, err := http.Get(url)
	checkErr(err)

	bodyAsByteSlice, err := ioutil.ReadAll(resp.Body)
	checkErr(err)

	return resp, bodyAsByteSlice
}

func getPeers() {
	_, bodyAsByteSlice := httpGet(SERVER_ADDRESS + PEERS_PATH)
	fmt.Println(string(bodyAsByteSlice))
}

func getAdressesOfPeer(peerName string) []string {
	resp, bodyAsByteSlice := httpGet(SERVER_ADDRESS + PEERS_PATH + "/" + peerName + "/addresses")

	if resp.StatusCode == 404 {
		LOGGING_FUNC(peerName + " is not known by server")
		return make([]string, 0)
	}

	return strings.Split(string(bodyAsByteSlice), "\n")
}

func getRootOfPeer(peerName string) []byte {
	resp, bodyAsByteSlice := httpGet(SERVER_ADDRESS + PEERS_PATH + "/" + peerName + "/root")

	if resp.StatusCode == 204 {
		LOGGING_FUNC(peerName + " has not declared a root yet!")
		return make([]byte, 0) // TODO Fix this and other instances of returning wrong value after logging (maybe exit?)
	} else if resp.StatusCode == 404 {
		LOGGING_FUNC(peerName + "is not known by server!")
		return make([]byte, 0)
	}

	return bodyAsByteSlice
}
