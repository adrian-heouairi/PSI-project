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

	err = createMerkleTree(SHARED_FILES_DIR)
	checkErr(err)
	ourTree.printMerkleTreeRecursively()

	checkErrPanic(initUdp())

	go listenAndRespond()
	go keepAliveMainPeer()
	mainMenu()
}
