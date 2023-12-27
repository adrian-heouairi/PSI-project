package main

import (
	"fmt"
	"net"
	"os"
)

func main() {
	err := mkdir(DOWNLOAD_DIR)
	checkErr(err)

	err = os.Chdir(DOWNLOAD_DIR)
	checkErr(err)

	checkErrPanic(initUdp())

    go listenAndRespond()
	serverUdpAddresses, err := getAdressesOfPeer(SERVER_PEER_NAME)
	checkErr(err)
	
	jchAddr, err := net.ResolveUDPAddr("udp4", serverUdpAddresses[0])
	checkErr(err)
    fmt.Println("before sending hello to jch")
	helloReply, _ := sendAndReceiveMsg(addrUdpMsg{jchAddr, createHello()})

	fmt.Println(udpMsgToString(helloReply.Msg))

	err = listAllFilesOfPeer("jch.irif.fr")
	checkErr(err)
	/*_, err = sendAndReceiveMsg(createHello()) // TODO Check that it is a HelloReply
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
	checkErr(err)*/


	//err = downloadFullTreeOfPeer("jch.irif.fr")
	//checkErr(err)
}
