package main

import (
    "fmt"
	"os"
)

func main() {
	initOurPeerName()
	err := mkdir(DOWNLOAD_DIR)
	checkErr(err)

	err = os.Chdir(DOWNLOAD_DIR)
	checkErr(err)
    ourTree, err = pathToMerkleTreeWithoutHashComputation(SHARED_FILES_DIR, nil)
    fmt.Println("our tree == nil ", ourTree == nil)
    fmt.Println("our tree", ourTree.toString())

	checkErrPanic(initUdp())

	go listenAndRespond()
	go keepAliveMainPeer()
	mainMenu()
}
