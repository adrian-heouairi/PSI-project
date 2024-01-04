package main

import (
	"fmt"
	"os"
)

func main() {
	var cmdToRun []string
	if len(os.Args) > 1 && os.Args[1] == "--debug" {
		DEBUG = true
		LOGGING_FUNC("Debugging")
		cmdToRun = os.Args[2:]
	} else {
		cmdToRun = os.Args[1:]
	}

	initOurPeerName()

	err := mkdirP(SHARED_FILES_DIR)
	checkErr(err)

	err = mkdirP(DOWNLOAD_DIR)
	checkErr(err)
	err = os.Chdir(DOWNLOAD_DIR)
	checkErr(err)
    if !checkNbChildrenExportedFilTree(SHARED_FILES_DIR, MAX_DIRECTORY_CHILDREN) {
        fmt.Fprintf(os.Stderr, "Your tree is not valid")
    }

	err = exportMerkleTree()
	checkErr(err)

	checkErrPanic(initUdp())

	go listenAndRespond()
	go keepAliveMainPeer()

	if len(cmdToRun) > 0 {
		runLine(cmdToRun)
	} else {
		mainMenu()
	}
}
