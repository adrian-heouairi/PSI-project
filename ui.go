package main

import (
	"fmt"
	"os"

	"github.com/chzyer/readline"
)

var helpMessage = ""

// str is the whole current line e.g. findrem jc
func peersListAutoComplete(str string) []string {
	peers, err := restGetPeers(false)
	if err != nil {
		return []string{}
	}
	return peers
}

func pathAutoComplete(line string) []string {
	res := []string{}

	restPeers, err := restGetPeers(false)
	if err != nil {
		return []string{}
	}

	for _, p := range restPeers {
		res = append(res, p)
		res = append(res, p+"/")
	}

	splittedLine := splitLine(line)

	if len(splittedLine) < 2 {
		return res
	}

	if grep("^[^/]+", splittedLine[1]) {
		peerName := replaceAllRegexBy(splittedLine[1], "/.*", "")
		_, found := peersGet(peerName)
		if found {
			pathHashMap, err := getPeerPathHashMap(peerName)
			if err == nil {
				for _, k := range getKeys(pathHashMap) {
					if k != peerName {
						res = append(res, k)
					}
				}
			}
		}
	}

	return res
}

func mainMenu() error {
	if helpMessage == "" {
		helpMessage += "PATH is PEER_NAME[PATH2] with PATH2 = /videos for example\n"
		for _, v := range CMD_MAP {
			helpMessage += "\t" + v.Name + v.Help + "\n"
		}
		helpMessage = helpMessage[:len(helpMessage)-1]
	}

	pcItems := []readline.PrefixCompleterInterface{}
	for _, v := range CMD_MAP {
		pcItems = append(pcItems, v.PcItem)
	}

	var completer = readline.NewPrefixCompleter(pcItems...)

	rl, err := readline.NewEx(&readline.Config{
		Prompt: CLI_PROMPT,
		//EOFPrompt:    EXIT_MESSAGE,
		HistoryFile:  READLINE_HISTORY_FILE,
		AutoComplete: completer,
	})
	if err != nil {
		return err
	}

	defer rl.Close()

	for {
		line, err := rl.Readline()
		if err != nil {
			return err
		}

		// TODO Support quotes in line
		runLine(splitLine(line)) // The line passed doesn't have \n at the end
	}
}

func runLine(splittedLine []string) {
	if len(splittedLine) == 0 {
		return
	}

	var v command
	var cmd *command
	for _, v = range CMD_MAP {
		if v.Name == splittedLine[0] {
			cmd = &v
			break
		}
	}

	if cmd != nil {
		if len(splittedLine) < cmd.MinArgc {
			fmt.Fprintln(os.Stderr, helpMessage)
			fmt.Fprintln(os.Stderr, CMD_TOO_FEW_ARGS)
			return
		}
	}

	switch splittedLine[0] {
	case "test":
		m, err := ConnectAndSendAndReceive(OUR_OTHER_PEER_NAME, createHello())
		if err != nil {
			LOGGING_FUNC(err)
		} else {
			fmt.Println("Received HelloReply from teammate:", udpMsgToString(m))
		}
		rootMsg := createMsg(ROOT, ourTree.Hash)
		rootReply, err := ConnectAndSendAndReceive(OUR_OTHER_PEER_NAME, rootMsg)
		checkErr(err)
		if err == nil {
			fmt.Println(udpMsgToString(rootReply))
		}
		getDatum := createMsg(GET_DATUM, rootReply.Body)
		datum, err := ConnectAndSendAndReceive(OUR_OTHER_PEER_NAME, getDatum)
		checkErr(err)
		if err == nil {
			fmt.Println(udpMsgToString(datum))
		}

	case CMD_MAP["HELLO"].Name:
		helloReply, err := ConnectAndSendAndReceive(splittedLine[1], createHello())
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
		} else {
			fmt.Println(udpMsgToString(helloReply))
		}
	case CMD_MAP["EXIT"].Name:
		os.Exit(0)
	case CMD_MAP["LIST_PEERS"].Name:
		if len(splittedLine) == 2 {
			if grep("--addr", splittedLine[1]) {
				restDisplayAllPeersWithTheirAddresses()
			} else {
				fmt.Fprintln(os.Stderr, "Invalid argument")
			}
		} else {
			restGetPeers(true)
		}
	case CMD_MAP["LIST_FILES"].Name:
		pathHashMap, err := getPeerPathHashMap(splittedLine[1])
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
		}
		for _, elt := range getKeys(pathHashMap) {
			fmt.Println(elt)
		}
	case CMD_MAP["CAT_FILE"].Name, CMD_MAP["DOWNLOAD_FILE"].Name:
		path := removeTrailingSlash(splittedLine[1])
		// TODO Support peers whose name contains /
		peerName := replaceAllRegexBy(path, "/.*", "")
		filenamesAndHashes, err := getPeerPathHashMap(peerName)
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
		}
		val, found := filenamesAndHashes[path]
		if found {
			downloadRecursive(peerName, val, path)
		} else {
			fmt.Fprintf(os.Stderr, "File %s not found\n", path)
		}
	case "":
		return
	default: // Includes HELP
		fmt.Fprintln(os.Stderr, helpMessage)
	}

	if splittedLine[0] == CMD_MAP["CAT_FILE"].Name {
		fileContents, err := os.ReadFile(splittedLine[1])
		if err == nil {
			fmt.Println(string(fileContents))
		}
	}
}
