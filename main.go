package main

import (
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
)

const SERVER_ADDRESS = "https://jch.irif.fr:8443"
const PEERS_PATH = "/peers/"

var LOGGING_FUNC = log.Println

func getPeers() {
	req, err := http.NewRequest("GET", SERVER_ADDRESS+PEERS_PATH, nil)
	if err != nil {
		LOGGING_FUNC("Error")
	}

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		LOGGING_FUNC("Error")
	}

	respAsByteSlice, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		LOGGING_FUNC("Error")
	}
	respBodyStr := string(respAsByteSlice)

	fmt.Println(respBodyStr)

}

func getAdressOfPeer(peerName string) {

	req, err := http.NewRequest("GET", SERVER_ADDRESS+
		PEERS_PATH+"/"+peerName+"/addresses", nil)
	if err != nil {
		LOGGING_FUNC("Error")
	}

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		LOGGING_FUNC("Error")
	}

	if resp.StatusCode == 404 {
		fmt.Println(peerName + "is not known by server!")
		return 
	}
	respAsByteSlice, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		LOGGING_FUNC("Error")
	}
	respBodyStr := string(respAsByteSlice)

	fmt.Println(respBodyStr)
}

func getRootOfPeer(peerName string)  [] byte{

	req, err := http.NewRequest("GET", SERVER_ADDRESS+PEERS_PATH+"/"+peerName+"/root", nil)
	if err != nil {
		LOGGING_FUNC("Error")
	}

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		LOGGING_FUNC("Error")
	}

	if resp.StatusCode == 204 {
		fmt.Println(peerName + " has not declared root yet!")
		return make([]byte, 0)
	} else if resp.StatusCode == 404 {
		fmt.Println(peerName + "is not known by server!")
		return make([]byte, 0)
	}
	respAsByteSlice, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		LOGGING_FUNC("Error")
	}
	respBodyStr := string(respAsByteSlice)

	fmt.Println("root of " + peerName + " is : " + respBodyStr)
    return respAsByteSlice
}

func main() {
	getPeers()
	getAdressOfPeer("jch.irif.fr")
	getAdressOfPeer("jch.irsif.fr")
	getRootOfPeer("jch.irsif.fr")
	getRootOfPeer("Slartibartfast")
}
