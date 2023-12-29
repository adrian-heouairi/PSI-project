package main

import (
	"fmt"
	"os"
	"strings"

	"github.com/chzyer/readline"
)

var helpMessage = 
fmt.Sprintf(`help
help 2`)

func squareText(text string) string {
    stars := ""
    for i := 0; i < len(text) + 2; i++ {
       stars += "*"
    }
    res := stars + "\n|" + text + "|\n" + stars
    return res
}

func displayMenuAndTakeChoice(title string, choices []string) (int, error) {
    fmt.Println(squareText(title))
    for i, value := range choices {
        fmt.Println(i + 1, " - ", value)
    }
    var choice int
    fmt.Print("[", 1, "..", len(choices), "] : ")
    _, err := fmt.Scanf("%d", &choice)
    if err != nil {
        fmt.Println(err)
        return -1, err
    }
    return choice,nil
}

func mainMenu() {
    restart := true
    for restart {
        choices := []string{"Show available peers", "Show addresses of peer", "Show tree of peer", "Dump full peer tree", "Exit"}
        choice ,err := displayMenuAndTakeChoice("PEER CLIENT", choices)
        fmt.Println(choice,err)
        switch choice {
        case 1:
            restGetPeers(true)
        case 2:
            peers, err := restGetPeers(false)
            if err != nil {
                return
            }
            choice, err = displayMenuAndTakeChoice("ADRESSES", peers)
            restGetAddressesOfPeer(peers[choice - 1],true)
            case 3: 

            peers, err := restGetPeers(false)
            if err != nil {
                return
            }
            choice, err = displayMenuAndTakeChoice("TREE OF PEER", peers)
            peerName := peers[choice - 1]
            rootHash,err := restGetRootOfPeer(peerName)
            datumType, datumToCast, err := downloadDatum(peers[choice - 1], rootHash)
            if err != nil {
                return 
            }

            if datumType == DIRECTORY {
                datum := datumToCast.(datumDirectory)
                fileNames := make([]string,len(datum.Children))

                for key := range datum.Children {
                    fileNames = append(fileNames, key)
                }
                choice, err = displayMenuAndTakeChoice("CHOOSE FILE TO DOWNLOAD", fileNames)
                downloadRecursive(peerName, datum.Children[fileNames[choice - 1]], fileNames[choice - 1])
            }
        case 4:
            peers, err := restGetPeers(false)
            if err != nil {
                return
            }
            choice, err = displayMenuAndTakeChoice("DOWNLOAD TREE", peers)
            downloadFullTreeOfPeer(peers[choice - 1])
        case 5:
            restart = false
        }

    }
}

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
        } else {
            downloadFullTreeOfPeer(splitLine[1])
        }
    default: // Includes HELP_CMD
        fmt.Fprintln(os.Stderr, helpMessage)
    }
}

func mainMenu2() error {
    /*reader := bufio.NewReader(os.Stdin)

    for {
        //line := ""
        fmt.Print(CLI_PROMPT)
        //_, err := fmt.Scanf("%s\n", &line)
        line, err := reader.ReadString('\n')
        if err != nil {
            return
        }
            
        line = strings.TrimSpace(line)

        fmt.Println(line)
    }*/

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
