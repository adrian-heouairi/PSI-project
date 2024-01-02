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

	ourTree, err = pathToMerkleTreeWithoutNonChunkHashes(SHARED_FILES_DIR, nil)
	checkErr(err)
	//fmt.Println("our tree", ourTree.toString())
	ourTree.printMerkleTreeRecursively()

	checkErrPanic(initUdp())

	go listenAndRespond()
	go keepAliveMainPeer()
	mainMenu()
}
