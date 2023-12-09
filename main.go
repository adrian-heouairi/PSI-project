package main

import (
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
    "math/rand"
	"encoding/binary"
	"net"
)

type udpSocket struct {
	Host string
	Port int
}

type msg struct {
	Id     uint32
	Type   uint8
	Length uint16
	Body   []byte
}

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
func toByteSlice(toCast msg) []byte {
	var idToByteSlice []byte = make([]byte, 4)
	binary.LittleEndian.PutUint32(idToByteSlice, toCast.Id)
	var typeToByteSlice []byte = make([]byte, 1)
	typeToByteSlice[0] = toCast.Type
	var lengthToByteSlice []byte = make([]byte, 2)
	binary.BigEndian.PutUint16(lengthToByteSlice, toCast.Length)
	var res = append(idToByteSlice, typeToByteSlice...)
	res = append(res, lengthToByteSlice...)
	res = append(res, toCast.Body...)
	return res
}

func createHello() []byte {
    var helloMsg msg
    helloMsg.Id = rand.Uint32()
    helloMsg.Type = 2
    extention := make([] byte, 4)
    name := "tc"
    nameAsBytes := [] byte(name)
    var res = append(extention, nameAsBytes...)
    helloMsg.Length = uint16(len(res))
    return toByteSlice(helloMsg)
}

func main() {
	/*getPeers()
	getAdressOfPeer("jch.irif.fr")
	getAdressOfPeer("jch.irsif.fr")
	getRootOfPeer("jch.irsif.fr")
	getRootOfPeer("Slartibartfast")
*/
	var sock udpSocket
    sock.Host = SERVER_ADDRESS[8:19]
    fmt.Println(sock.Host)
    sock.Port =  8443
	// Server address
	serverAddr, err := net.ResolveUDPAddr("udp", sock.Host+":"+fmt.Sprint(sock.Port))
	if err != nil {
		fmt.Println("Error resolving address:", err)
		return
	}

	// Establish a connection
	conn, err := net.DialUDP("udp", nil, serverAddr)
	if err != nil {
		fmt.Println("Error connecting:", err)
		return
	}
	defer conn.Close()


	// Send the UDP packet
	_, err = conn.Write(createHello())
	if err != nil {
		fmt.Println("Error sending UDP packet:", err)
		return
	}
    
}
