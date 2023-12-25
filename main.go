package main

import (
	"crypto/sha256"
	"fmt"
	"log"
	"net"
	"os"
)

var jchConn *net.UDPConn

func createDownloadDirAndCd() {
	_, err := os.Stat(DOWNLOAD_DIR)
	if os.IsNotExist(err) {
		err = os.Mkdir(DOWNLOAD_DIR, 0755)
		if err != nil {
			LOGGING_FUNC(err)
		}
	}
	err = os.Chdir(DOWNLOAD_DIR)
	if err != nil {
		LOGGING_FUNC(err)
	}
}

func main() {
	createDownloadDirAndCd()

    serverUdpAddresses := getAdressesOfPeer(SERVER_PEER_NAME)

    // Server address
    serverAddr, err := net.ResolveUDPAddr("udp", serverUdpAddresses[0])
    if err != nil {
        LOGGING_FUNC(err)
    }

    // Establish a connection
    jchConn, err = net.DialUDP("udp", nil, serverAddr)
    if err != nil {
        LOGGING_FUNC(err)
    }
    defer jchConn.Close()

    buffer := make([]byte, UDP_BUFFER_SIZE)

    sendAndReceiveMsg := func (toSend udpMsg) udpMsg {
        _, err = jchConn.Write(udpMsgToByteSlice(toSend))
        if err != nil {
            LOGGING_FUNC(err)
        }

        _, _, err = jchConn.ReadFromUDP(buffer)
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

    _, err = jchConn.Write(udpMsgToByteSlice(helloMsg))
    if err != nil {
        LOGGING_FUNC(err)
    }

    _, _, err = jchConn.ReadFromUDP(buffer)
    if err != nil {
        LOGGING_FUNC(err)
    }

    helloReplyMsg := byteSliceToUdpMsg(buffer)

    if helloMsg.Id != helloReplyMsg.Id || helloReplyMsg.Type != HELLO_REPLY {
        LOGGING_FUNC("Invalid HelloReply message")
    }

    _, _, err = jchConn.ReadFromUDP(buffer)
    if err != nil {
        LOGGING_FUNC(err)
    }

    publicKeyMsg := byteSliceToUdpMsg(buffer)

    if publicKeyMsg.Type != PUBLIC_KEY {
        LOGGING_FUNC("Invalid PublicKey message")
    }

    publicKeyReplyMsg := udpMsg{publicKeyMsg.Id, PUBLIC_KEY_REPLY, 0, make([]byte, 0)}

    _, err = jchConn.Write(udpMsgToByteSlice(publicKeyReplyMsg))
    if err != nil {
        LOGGING_FUNC(err)
    }

    _, _, err = jchConn.ReadFromUDP(buffer)
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

    _, err = jchConn.Write(udpMsgToByteSlice(rootReplyMsg))
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
			fmt.Println("CONTENT before write : " + string(content))
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
                fk = key
                if i == 4 {
                    break
                }
				i++
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
