package main

import (
	"crypto/sha256"
	//"fmt"
	//"log"
	"net"
	"os"
)

var jchConn *net.UDPConn

func createDownloadDirAndCd() error {
	_, err := os.Stat(DOWNLOAD_DIR)
	if os.IsNotExist(err) {
		err = os.Mkdir(DOWNLOAD_DIR, 0755)
		if err != nil {
			return err
		}
	} else {
		return err
	}
	err = os.Chdir(DOWNLOAD_DIR)
	return err
}

func main() {
	err := createDownloadDirAndCd()
	checkErr(err)

	serverUdpAddresses, err := getAdressesOfPeer(SERVER_PEER_NAME)
	checkErr(err)

	// Server address
	serverAddr, err := net.ResolveUDPAddr("udp", serverUdpAddresses[0])
	checkErr(err)

	// Establish a connection
	jchConn, err = net.DialUDP("udp", nil, serverAddr)
	checkErr(err)
	defer jchConn.Close()

	_, err = sendAndReceiveMsg(createHello()) // TODO Check that it is a HelloReply
	checkErr(err)
	publicKeyMsg, err := receiveMsg()
	checkErr(err)
	publicKeyReplyMsg := createMsgWithId(publicKeyMsg.Id, PUBLIC_KEY_REPLY, []byte{})
	err = sendMsg(publicKeyReplyMsg)
	checkErr(err)
	rootMsg, err := receiveMsg()
	checkErr(err)
	hasher := sha256.New()
	rootReplyMsg := createMsgWithId(rootMsg.Id, ROOT_REPLY, hasher.Sum(nil))
	err = sendMsg(rootReplyMsg)
	checkErr(err)

	err = listAllFilesOfPeer("jch.irif.fr")
	checkErr(err)
}
