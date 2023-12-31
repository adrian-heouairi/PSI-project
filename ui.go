package main

import (
	"fmt"
	"os"
	"strings"

	"github.com/chzyer/readline"
)

var helpMessage = fmt.Sprintf(
	`CMD [OPTION]
    - %s : shows the connected peers
    - %s PATH : downloads the datum at the PATH asumes that the path is absolute i.e contains peer name
    - %s PEER : shows the files shared by PEER
    - %s quit`, LIST_PEERS_CMD, DOWNLOAD_FILE_CMD, LIST_FILES_CMD, EXIT_CMD)
    

    func peersList() []string {
        peers, _ := restGetPeers(false)
        return peers
    }
    func parseLine(line string) {
        line = strings.TrimSpace(line)
        line = replaceAllRegexBy(line, " +", " ")
        splitLine := strings.Split(line, " ")

        if len(splitLine) == 0 {
            fmt.Println(helpMessage)
        }

        switch splitLine[0] {
        case EXIT_CMD:
            os.Exit(0)
        case LIST_PEERS_CMD:
            restGetPeers(true)
        case LIST_FILES_CMD:
            if len(splitLine) < 2 {
                fmt.Fprintln(os.Stderr, helpMessage)
            } else {
                filenamesAndHashes, err := getPeerAllDataHashes(splitLine[1])
                if err != nil {
                    fmt.Println(err)
                }
                availableFiles := getKeys(filenamesAndHashes)
                for _, elt := range availableFiles {
                    fmt.Println(elt)

                }
            }
            //case CAT_FILE_CMD:
        case DOWNLOAD_FILE_CMD:
            if len(splitLine) < 2 {
                fmt.Fprintln(os.Stderr, helpMessage)
            } else if len(splitLine) == 2 { // We assume that splitLine[1] is <peer>/<path>
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

    func mainMenu() error {
        var completer = readline.NewPrefixCompleter(
            //readline.PcItem(LIST_PEERS_CMD, readline.PcItemDynamic(peersList)),
            readline.PcItem(LIST_FILES_CMD),
            readline.PcItem(DOWNLOAD_FILE_CMD),
            readline.PcItem(EXIT_CMD))
            rl, err := readline.NewEx(&readline.Config{
                UniqueEditLine: true,
                Prompt: CLI_PROMPT,
                InterruptPrompt: "^C",
                EOFPrompt: "exit",
                HistoryFile: "/tmp/readlinehistory.tmp",
                AutoComplete: completer,
            })

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
