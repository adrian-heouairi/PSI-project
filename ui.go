package main

import (
	"fmt"
	"os"

	"github.com/chzyer/readline"
)

var helpMessage = fmt.Sprintf(`CMD [OPTION]
    - %s: quit
    - %s: shows this message
    - %s: shows the connected peers
    - %s PEER: shows the files shared by PEER
    - %s PATH: cats the remote file at PATH, assumes that the path is absolute
               i.e contains peer name
    - %s PATH: downloads recursively the file or directory at PATH, assumes that
               the path is absolute i.e contains peer name`,
	stringSliceToAnySlice(CMD_LIST)...)

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
		res = append(res, p + "/")
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
	var completer = readline.NewPrefixCompleter(
		readline.PcItem(EXIT_CMD),
		readline.PcItem(HELP_CMD),
		readline.PcItem(LIST_PEERS_CMD),
		readline.PcItem(LIST_FILES_CMD, readline.PcItemDynamic(peersListAutoComplete)),
		readline.PcItem(CAT_FILE_CMD, readline.PcItemDynamic(pathAutoComplete)),
		readline.PcItem(DOWNLOAD_FILE_CMD, readline.PcItemDynamic(pathAutoComplete)))

	rl, err := readline.NewEx(&readline.Config{
		Prompt:       CLI_PROMPT,
		EOFPrompt:    EXIT_MESSAGE,
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
		parseLine(line) // The line passed doesn't have \n at the end
	}
}

func parseLine(line string) {
	splittedLine := splitLine(line)

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

	case EXIT_CMD:
		os.Exit(0)
	case LIST_PEERS_CMD:
		restGetPeers(true)
	case LIST_FILES_CMD:
		if len(splittedLine) < 2 {
			fmt.Fprintln(os.Stderr, helpMessage)
		} else {
			pathHashMap, err := getPeerPathHashMap(splittedLine[1])
			if err != nil {
				fmt.Fprintln(os.Stderr, err)
			}
			for _, elt := range getKeys(pathHashMap) {
				fmt.Println(elt)
			}
		}
	//case CAT_FILE_CMD:
	case DOWNLOAD_FILE_CMD:
		if len(splittedLine) < 2 {
			fmt.Fprintln(os.Stderr, helpMessage)
		} else if len(splittedLine) == 2 { // We assume that splitLine[1] is <peer>/<path>
			path := removeTrailingSlash(splittedLine[1])
			// TODO : support peers whose name contains /
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
		}
	case "":
		return
	default: // Includes HELP_CMD
		fmt.Fprintln(os.Stderr, helpMessage)
	}
}
