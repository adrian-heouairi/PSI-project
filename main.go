package main

import (
	"os"
)

func main() {
	// TODO Here check that current dir is the root of the project

	var cmdToRun []string
	if len(os.Args) > 1 && os.Args[1] == "--debug" {
		DEBUG = true
		LOGGING_FUNC("Debugging")
		cmdToRun = os.Args[2:]
	} else {
		cmdToRun = os.Args[1:]
	}

	initOurPeerName()

	err := mkdirP(DOWNLOAD_DIR)
	checkErr(err)
	err = os.Chdir(DOWNLOAD_DIR)
	checkErr(err)

	// TODOSEVI Check at start that any subdirectory of SHARED_FILES_DIR has at most 16 children
	err = mkdirP(SHARED_FILES_DIR)
	checkErr(err)

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
