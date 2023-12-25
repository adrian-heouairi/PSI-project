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
	"os"
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
    CHUNK                 byte = 0
    TREE                  byte = 1
    DIRECTORY             byte = 2
    DATUM_TYPE_INDEX      byte = 32
    HASH_SIZE             byte = 32
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
    var typeAsString string
    var typeOfDatumAsString string
    switch msg.Type {
    case 0:
        typeAsString = "NOOP"
    case 1:
        typeAsString = "ERROR"
    case 128:
        typeAsString = "ERROR REPLY"
    case 2:
        typeAsString = "HELLO"
    case 129:
        typeAsString = "HELLO REPLY"
    case 3:
        typeAsString = "PUBLIC KEY"
    case 130:
        typeAsString = "PUBLIC KEY REPLY"
    case 4:
        typeAsString = "ROOT"
    case 131:
        typeAsString = "ROOT REPLY"
    case 5:
        typeAsString = "GET DATUM"
    case 132:
        typeAsString = "DATUM"
        switch msg.Body[DATUM_TYPE_INDEX] {
        case CHUNK:
            typeOfDatumAsString = "CHUNK"
        case TREE:
            typeOfDatumAsString = "TREE"
        case DIRECTORY:
            typeOfDatumAsString = "DIRECTORY"
        }
    case 6:
        typeAsString = "NAT TRAVERSAL REQUEST"
    case 133:
        typeAsString = "NO DATUM"
    case 7:
        typeAsString = "NAT TRAVERSAL"

    }
    if msg.Type == DATUM {
        typeAsString += " " + typeOfDatumAsString
    }
    return "Id: " + fmt.Sprint(msg.Id) + "\n" +
    "Type: " + typeAsString + "\n" +
    "Length: " + fmt.Sprint(msg.Length) + "\n" +
    "Body: " + string(msg.Body[:lengthToTake])
}

func castNameFromBytesSliceToString(name []byte) string {
    i := 0;
    for name[i] != 0 {
       i++ 
    }
    return string(name[:i])
}

// Returns a map containing the names and the hashes of the Datum message
// Assumes that msg is datum of type Directory
// Otherwise empty map is returned
func getDataHashes(msg udpMsg) map[string][]byte {
    res := make(map[string][]byte)
    if msg.Body[DATUM_TYPE_INDEX] == DIRECTORY {
        nbEntry := (msg.Length - 33) / 64
        for i := 0; i < int(nbEntry); i++{
            res[castNameFromBytesSliceToString(msg.Body[33 + i * 64 : 33 + i * 64 + 32])] = msg.Body[65 + i * 64 : 65 + i * 64 + 32]

        }
    }
    return res
}

func check_data_integrity(hash []byte, content []byte) bool {
    computed_hash := sha256.New()
    computed_hash.Write(content)
    hash_sum := computed_hash.Sum(nil)
    if len(hash_sum) != len(hash){

        return false
    }
    for i := 0; i < len(hash); i++ {
        if hash_sum[i] != hash[i] {
            return false
        }
    }
    return true
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
    download_and_save_file := func (file_name string, hash []byte, content[] byte){}
    download_and_save_file = func (file_name string,hash []byte, content []byte) {
        f, err := os.OpenFile(file_name,os.O_WRONLY|os.O_CREATE,0644)
        if err != nil {
            fmt.Println("Failed to create")
            log.Fatal(err)
        }
        defer f.Close()
        datumMsg := sendAndReceiveMsg(createMsg(GET_DATUM,hash))
        fmt.Println(udpMsgToString(datumMsg))
        if datumMsg.Type != DATUM {
            LOGGING_FUNC("Not a datum msg")
            //return
        }
        hashToCheck := datumMsg.Body[:32]
        if(datumMsg.Body[DATUM_TYPE_INDEX] == CHUNK) { 
            if check_data_integrity(hashToCheck,datumMsg.Body[32:datumMsg.Length]) {
                content = append(content,datumMsg.Body[33:datumMsg.Length]...)
           } else {
                LOGGING_FUNC("CORRUPTED CHUNK")
            }
        } else if (datumMsg.Body[DATUM_TYPE_INDEX] == TREE) {
            // bigFile/Tree : 1 + (32 *[2:32])
            nbElt := (datumMsg.Length - 33) / uint16(HASH_SIZE)
            for i := 0; i < int(nbElt); i++ {
                download_and_save_file(file_name, datumMsg.Body[33 + i * 32:65 + i * 32],content) 
            }
            bytesWritten, err := f.Write(content)
            if err != nil {
                LOGGING_FUNC("WRINTING GOES WRONG")
            }
            fmt.Println("wrote : " + fmt.Sprint(bytesWritten) + "bytes")

        } else if datumMsg.Body[DATUM_TYPE_INDEX] == DIRECTORY {
            hashes := getDataHashes(datumMsg) 
            var fk string
            i := 0
            for key := range hashes {
                i++
                fk = key
                if i == 3 {
                    break
                }
            }
            download_and_save_file("bidule",hashes[fk],content)
        }
    }



    mr := sendAndReceiveMsg(createMsg(ROOT, hasher.Sum(nil)))
    fmt.Println(udpMsgToString(mr))
    rootJuliuszUDP := mr.Body

    //rootJuliuszREST := getRootOfPeer("jch.irif.fr")

    rootDatumReply := sendAndReceiveMsg(createMsg(GET_DATUM, rootJuliuszUDP))
    // hash puis contenu
    if !check_data_integrity(
        rootDatumReply.Body[:32],
        rootDatumReply.Body[32:rootDatumReply.Length]) {
            LOGGING_FUNC("DATUM IS CORRUPTED")
        }
        hashes := getDataHashes(rootDatumReply)
        fmt.Println(hashes)
        download_and_save_file("images",rootDatumReply.Body[:32],[]byte{})
    }
