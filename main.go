package main

import (
	"os"
)

func main() {
	err := mkdir(DOWNLOAD_DIR)
	checkErr(err)

	err = os.Chdir(DOWNLOAD_DIR)
	checkErr(err)

	checkErrPanic(initUdp())

	go listenAndRespond() 
    go keepAliveMainServer()
    mainMenu()
}
