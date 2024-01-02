package main

import (
	"os"
)

func main() {
	if len(os.Args) > 1 {
		DEBUG = true
		LOGGING_FUNC("Debugging")
	}

	initOurPeerName()

	err := mkdir(DOWNLOAD_DIR)
	checkErr(err)
	err = os.Chdir(DOWNLOAD_DIR)
	checkErr(err)

	err = exportMerkleTree()
	checkErr(err)
	//ourTree.printMerkleTreeRecursively()

	checkErrPanic(initUdp())

	go listenAndRespond()
	go keepAliveMainPeer()
	mainMenu()
}
