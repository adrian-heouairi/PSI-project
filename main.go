package main

import (
	"os"
)

func main() {
	initOurPeerName()

	err := mkdir(DOWNLOAD_DIR)
	checkErr(err)

	err = os.Chdir(DOWNLOAD_DIR)
	checkErr(err)

	checkErrPanic(initUdp())

	go listenAndRespond()
	go keepAliveMainPeer()
	mainMenu() 
}
