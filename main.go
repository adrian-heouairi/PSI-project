package main

import (
	"crypto/sha256"
	"fmt"
	"os"
	"time"
)

func main() {
	err := mkdir(DOWNLOAD_DIR)
	checkErr(err)

	err = os.Chdir(DOWNLOAD_DIR)
	checkErr(err)

	checkErrPanic(initUdp())

	go listenAndRespond()
	go keepAliveMainPeer()

	for {
		hasher := sha256.New()
		rootMsg := createMsg(ROOT, hasher.Sum(nil))
		rootReply, err := ConnectAndSendAndReceive("AS2", rootMsg)
		checkErr(err)
		if err == nil {
			fmt.Println(udpMsgToString(rootReply))
		}

		time.Sleep(2 * time.Second)
	}

	//mainMenu()
}
