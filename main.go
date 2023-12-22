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
)

const SERVER_ADDRESS = "https://jch.irif.fr:8443"
const PEERS_PATH = "/peers/"
const OUR_PEER_NAME = "AS"
const SERVER_PEER_NAME = "jch.irif.fr"

var LOGGING_FUNC = log.Println

const (
	NOOP                  byte = 0
	ERROR                 byte = 1
	ERROR_REPLY           byte = 128
	HELLO                 byte = 2
	HELLO_REPLY           byte = 129
	PUBLIC_KEY            byte = 3
	PUBLIC_KEY_REPLY      byte = 130
	ROOT                  byte = 4
	ROOT_REPLY            byte = 131
	GET_DATUM             byte = 5
	DATUM                 byte = 132
	NO_DATUM              byte = 133
	NAT_TRAVERSAL_REQUEST byte = 6
	NAT_TRAVERSAL         byte = 7
)

type udpMsg struct {
	Id     uint32
	Type   uint8
	Length uint16
	Body   []byte
}

func udpMsgToByteSlice(toCast udpMsg) []byte {
	var idToByteSlice []byte = make([]byte, 4)
	binary.BigEndian.PutUint32(idToByteSlice, toCast.Id)
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
	m.Id = binary.BigEndian.Uint32(toCast[0:4])
	m.Type = toCast[4]
	m.Length = binary.BigEndian.Uint16(toCast[5:7])
	m.Body = append([]byte{}, toCast[7 : 7+m.Length]...)
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

func createMsg(msgType byte, msgBody []byte) udpMsg {
    var msg udpMsg
    msg.Id = rand.Uint32()
    msg.Type = msgType
    msg.Body = msgBody
    msg.Length = uint16(len(msgBody))

    return msg
}

func udpMsgToString(msg udpMsg) string {
    lengthToTake := len(msg.Body)
    if lengthToTake > 32 {
        lengthToTake = 32
    }

    return "Id: " + fmt.Sprint(msg.Id) + "\n" +
        "Type: " + fmt.Sprint(msg.Type) + "\n" +
        "Length: " + fmt.Sprint(msg.Length) + "\n" +
        "Body: " + string(msg.Body[:lengthToTake])
}

func main() {
	/*getPeers()
	getAdressOfPeer("jch.irif.fr")
	getAdressOfPeer("jch.irsif.fr")
	getRootOfPeer("jch.irsif.fr")
	getRootOfPeer("Slartibartfast")
	*/

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

    sendAndReceiveMsg := func (toSend udpMsg) udpMsg {
        _, err = conn.Write(udpMsgToByteSlice(toSend))
        if err != nil {
            LOGGING_FUNC(err)
        }

        _, _, err = conn.ReadFromUDP(buffer)
        if err != nil {
            LOGGING_FUNC(err)
        }

        replyMsg := byteSliceToUdpMsg(buffer)

        // TODO We should verify that the type of the response corresponds to the request
        if toSend.Id != replyMsg.Id {
            LOGGING_FUNC("Query and reply IDs don't match")
        }

        return replyMsg
    }

	helloMsg := createHello()

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
	//h.Write([]byte(""))
	rootReplyMsg := udpMsg{rootMsg.Id, ROOT_REPLY, 32, hasher.Sum(nil)}

	_, err = conn.Write(udpMsgToByteSlice(rootReplyMsg))
	if err != nil {
		LOGGING_FUNC(err)
	}




    mr := sendAndReceiveMsg(createMsg(ROOT, hasher.Sum(nil)))
    fmt.Println(udpMsgToString(mr))
    rootJuliuszUDP := mr.Body

    //rootJuliuszREST := getRootOfPeer("jch.irif.fr")

    rootDatumReply := sendAndReceiveMsg(createMsg(GET_DATUM, rootJuliuszUDP))
    // Parser racine (directory)
    fmt.Println(udpMsgToString(rootDatumReply))
}
