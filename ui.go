package main

import (
	"fmt"
	"os"
	"strings"

	"github.com/chzyer/readline"
)

var helpMessage = 
fmt.Sprintf(
`CMD [OPTION]
    - lspeers : shows the connected peers
    - wget PATH : downloads the datum at the PATH asumes that the path is absolute i.e contains peer name
    - lsrem PEER : shows the files shared by PEER`)
func parseLine(line string) {
    line = strings.TrimSpace(line)
    line = replaceAllRegexBy(line, " +", " ")
    splitLine := strings.Split(line, " ")
    
    if len(splitLine) == 0 {
        return
    }

    switch splitLine[0] {
	case LIST_PEERS_CMD:
        restGetPeers(true)
	case LIST_FILES_CMD:
        if len(splitLine) < 2 {
            fmt.Fprintln(os.Stderr, helpMessage)
        } else {
            listAllFilesOfPeer(splitLine[1])
        }
	//case CAT_FILE_CMD:
	case DOWNLOAD_FILE_CMD:
        if len(splitLine) < 2 {
            fmt.Fprintln(os.Stderr, helpMessage)
        } else if len(splitLine) == 2{// We assume that splitLine[1] is <peer>/<path>
            path := removeTrailingSlash(splitLine[1])
            // TODO : support peers whose name contains /
            peerName := replaceAllRegexBy(path, "/.*", "")
            filenamesAndHashes, err := getPeerAllDataHashes(peerName)
            if err != nil {
                fmt.Println(err)
            }
            val, found := filenamesAndHashes[path]
            if found {
                downloadRecursive(peerName, val, path)

            } else {
                fmt.Println("NOT FOUND")
                fmt.Println(path)
            }
        }

        default: // Includes HELP_CMD
        fmt.Fprintln(os.Stderr, helpMessage)
    }
}

func mainMenu2() error {
    rl, err := readline.New(CLI_PROMPT)
    if err != nil {
        return err
    }
    defer rl.Close()

    for {
        line, err := rl.Readline()
        if err != nil { // io.EOF
            return err
        }
        
        parseLine(line)
    }
}
