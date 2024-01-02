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

	err = exportMerkleTree()
	checkErr(err)
	ourTree.printMerkleTreeRecursively()

	checkErrPanic(initUdp())

	go listenAndRespond()
	go keepAliveMainPeer()
	mainMenu()
}
