package main

import (
	"crypto/sha256"
	"encoding/binary"
	"fmt"
	"io/ioutil"
	"log"
	"math/rand"
	"net"
	"net/http"
	"strings"
    "time"
    "sync"
    "strconv"
)

const SERVER_ADDRESS = "https://jch.irif.fr:8443"
const PEERS_PATH = "/peers/"
const OUR_PEER_NAME = "AS"
const SERVER_PEER_NAME = "jch.irif.fr"
var LOGGING_FUNC = log.Println

const (
    NOOP byte = 0
    ERROR byte = 1
    ERROR_REPLY byte = 128
    HELLO byte = 2
    HELLO_REPLY byte = 129
    PUBLIC_KEY byte = 3
    PUBLIC_KEY_REPLY byte = 130
    ROOT byte = 4
    ROOT_REPLY byte = 131
    GET_DATUM byte = 5
    DATUM byte = 132
    NO_DATUM byte = 133
    NAT_TRAVERSAL_REQUEST byte = 6
    NAT_TRAVERSAL byte = 7
)

type udpMsg struct {
	Id     uint32
	Type   uint8
	Length uint16
	Body   []byte
}
func udpMsgToByteSlice(toCast udpMsg) []byte {
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
func byteSliceToUdpMsg(toCast []byte) udpMsg {
	var m udpMsg
	m.Id = binary.LittleEndian.Uint32(toCast[0:4])
	m.Type = toCast[4]
	m.Length = binary.LittleEndian.Uint16(toCast[5:7])
	m.Body = toCast[7:7 + m.Length]
	return m
}

func httpGet(url string) (*http.Response, []byte) {
    resp, err := http.Get(url)
    if err != nil {
		LOGGING_FUNC(err)
	}

    bodyAsByteSlice, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		LOGGING_FUNC(err)
	}

    return resp, bodyAsByteSlice
}

func getPeers() []string{
    _, bodyAsByteSlice := httpGet(SERVER_ADDRESS + PEERS_PATH)
    listOfPeersAsString := string(bodyAsByteSlice)
    listOfPeers := strings.Split(listOfPeersAsString,"\n")
    printNumberedList(listOfPeers)
    return listOfPeers
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
    resp, bodyAsByteSlice := httpGet(SERVER_ADDRESS+PEERS_PATH+"/"+peerName+"/root")

	if resp.StatusCode == 204 {
		LOGGING_FUNC(peerName + " has not declared a root yet!")
		return make([]byte, 0)
	} else if resp.StatusCode == 404 {
		LOGGING_FUNC(peerName + "is not known by server!")
		return make([]byte, 0)
	}
	
    return bodyAsByteSlice
}

func createHello() udpMsg {
    var helloMsg udpMsg
    helloMsg.Id = rand.Uint32()
    helloMsg.Type = 2
    extensions := make([]byte, 4)
    name := OUR_PEER_NAME
    nameAsBytes := []byte(name)
    var res = append(extensions, nameAsBytes...)
    helloMsg.Body = res
    helloMsg.Length = uint16(len(res))
    return helloMsg
}

func keepConnectionAlive() {
    serverUdpAddresses := getAdressesOfPeer(SERVER_PEER_NAME)

    // Server address
	serverAddr, err := net.ResolveUDPAddr("udp", serverUdpAddresses[0])
	if err != nil {
		LOGGING_FUNC(err)
	}

	// Establish a connection
	conn, err := net.DialUDP("udp", nil, serverAddr)
	if err != nil {
		LOGGING_FUNC(err)
	}
	defer conn.Close()

	buffer := make([]byte, 1048576)
    
    helloMsg := createHello()
for {


    _, err = conn.Write(udpMsgToByteSlice(helloMsg))
    if err != nil {
        LOGGING_FUNC(err)
    }

    _, _, err = conn.ReadFromUDP(buffer)
    if err != nil {
		LOGGING_FUNC(err)
	}

    helloReplyMsg := byteSliceToUdpMsg(buffer)

    if helloMsg.Id != helloReplyMsg.Id || helloReplyMsg.Type != HELLO_REPLY {
        LOGGING_FUNC("Invalid HelloReply message")
    }

    _, _, err = conn.ReadFromUDP(buffer)
    if err != nil {
		LOGGING_FUNC(err)
	}

    publicKeyMsg := byteSliceToUdpMsg(buffer)

    if publicKeyMsg.Type != PUBLIC_KEY {
        LOGGING_FUNC("Invalid PublicKey message")
    }

    publicKeyReplyMsg := udpMsg{publicKeyMsg.Id, PUBLIC_KEY_REPLY, 0, make([]byte, 0)}

    _, err = conn.Write(udpMsgToByteSlice(publicKeyReplyMsg))
    if err != nil {
        LOGGING_FUNC(err)
    }

    _, _, err = conn.ReadFromUDP(buffer)
    if err != nil {
		LOGGING_FUNC(err)
	}

    rootMsg := byteSliceToUdpMsg(buffer)

    if rootMsg.Type != ROOT {
        LOGGING_FUNC("Invalid Root message")
    }

    hasher := sha256.New()
    rootReplyMsg := udpMsg{rootMsg.Id, ROOT_REPLY, 32, hasher.Sum(nil)}

    _, err = conn.Write(udpMsgToByteSlice(rootReplyMsg))
    if err != nil {
        LOGGING_FUNC(err)
 }
     time.Sleep(30 * time.Second) 
     fmt.Println("After waiting 30 seconds")
 }
}
func printNumberedList(list []string) {
    for i:= 0 ; i < len(list) - 1; i++{
        fmt.Println(strconv.Itoa(i + 1) + " - " + list[i])
    }
}
func UI() {
    fmt.Println("PEER CLIENT")
    fmt.Println("1 - Get peers list")
    fmt.Println("2 - Get addresses of a peer")
    fmt.Println("3 - Get root of a peer")
    var i int

    fmt.Print("Type a number[1..3]: ")
    fmt.Scan(&i)
    switch i {
    case 1:
        fmt.Println("Here is the list of peers :")
        getPeers()
    case 2:
        listOfPeers := getPeers()
        fmt.Println("Which pair are you interesseted by[1.." + strconv.Itoa(len(listOfPeers)) + "] :")
        fmt.Scan(&i)
        fmt.Println(listOfPeers[i-1] + " addresses are : ")
        printNumberedList(getAdressesOfPeer(listOfPeers[i-1]))
    }
    fmt.Println("Your number is:", i)
}

func main() {
UI()
    /*
    PAY ATTENTION THIS CODE COMES FROM CHATGPT AND NEED TO BE REFACTORED
    */
var wg sync.WaitGroup

	// Start the goroutine and increment the WaitGroup counter
	wg.Add(1)
	go func() {
		defer wg.Done()
		keepConnectionAlive()
	}()

	// Your other main function logic...

	// Wait for the goroutine to finish before exiting
	wg.Wait()
	fmt.Println("Program exited")
go keepConnectionAlive()
/*
END OF CHATGPT CODE
*/
}
