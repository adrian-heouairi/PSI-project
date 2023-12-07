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

func main() {
    req, err := http.NewRequest("GET", SERVER_ADDRESS + PEERS_PATH, nil)
    if err != nil { LOGGING_FUNC("Error") }

    client := &http.Client{}
	resp, err := client.Do(req)
    if err != nil { LOGGING_FUNC("Error") }

    respAsByteSlice, err := ioutil.ReadAll(resp.Body)
    if err != nil { LOGGING_FUNC("Error") }
	respBodyStr := string(respAsByteSlice)

    fmt.Println(respBodyStr)
}
