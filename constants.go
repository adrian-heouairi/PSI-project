package main

import (
	"fmt"
	"log"
	"os"
	"time"

	"github.com/chzyer/readline"
)

// TODO Organize this and rename some and split it by .go file

const SERVER_ADDRESS = "https://jch.irif.fr:8443"
const PEERS_PATH = "/peers/"
const SERVER_PEER_NAME = "jch.irif.fr"
const DOWNLOAD_DIR = "PSI-download"
const SHARED_FILES_DIR = "../PSI-shared-files"
const UDP_LISTEN_PORT = 8444
const KEEP_ALIVE_PERIOD = 30 * time.Second
var OUR_PEER_NAME string
var OUR_OTHER_PEER_NAME string

var DEBUG bool = false

func initOurPeerName() {
	hostname, _ := os.Hostname()
	if hostname == "aetu2" {
		OUR_PEER_NAME = "AS"
		OUR_OTHER_PEER_NAME = "AS2"
	} else {
		OUR_PEER_NAME = "AS2"
		OUR_OTHER_PEER_NAME = "AS"
	}
}

const UDP_V4_SOCKET_SIZE = 6
const UDP_V6_SOCKET_SIZE = 18
const IPV4_SIZE = 4
const IPV6_SIZE = 16
const PORT_SIZE = 2

// To achieve 500 ms of waiting for a reply before reemitting the request
const MSG_QUEUE_CHECK_PERIOD = 2 * time.Millisecond
const MSG_QUEUE_CHECK_NUMBER = 250

const NUMBER_OF_REEMISSIONS = 4

const NAT_TRAVERSAL_RETRIES = 10 // We will send Hello (NUMBER_OF_REEMISSIONS + 1) * NAT_TRAVERSAL_RETRIES during our or their NAT traversal

const MSG_QUEUE_SIZE = 8192

const PRINT_MSG_BODY_TRUNCATE_SIZE = 100

func LOGGING_FUNC(v ...any) {
	if DEBUG {
		log.Println(v...)
	}
}

func LOGGING_FUNC_F(fmt string, v ...any) {
	if DEBUG {
		log.Printf(fmt, v...)
	}
}

func checkErr(err error) {
	if err != nil {
		LOGGING_FUNC(err)
	}
}

func checkErrPanic(err error) {
	if err != nil {
		panic(err)
	}
}

// HTTP response status codes
const (
	HTTP_NOT_FOUND  = 404
	HTTP_NO_CONTENT = 204
	HTTP_OK         = 200
)

const ( // UDP message types
	NOOP                  byte = 0
	ERROR                 byte = 1
	ERROR_REPLY           byte = 128
	HELLO                 byte = 2
	HELLO_REPLY           byte = 129
	PUBLIC_KEY            byte = 3
	PUBLIC_KEY_REPLY      byte = 130
	ROOT                  byte = 4
	ROOT_REPLY            byte = 131
	GET_DATUM             byte = 5
	DATUM                 byte = 132
	NO_DATUM              byte = 133
	NAT_TRAVERSAL_REQUEST byte = 6
	NAT_TRAVERSAL         byte = 7

	FIRST_RESPONSE_MSG_TYPE byte = 128
	MSG_VALID_PAIR          byte = 127
)

const ( // Datum message types
	CHUNK     byte = 0
	TREE      byte = 1
	DIRECTORY byte = 2
)

const HASH_SIZE = 32

const ( // Message and datum constants
	ID_SIZE     = 4
	TYPE_SIZE   = 1
	LENGTH_SIZE = 2

	HELLO_EXTENSIONS_SIZE = 4

	DATUM_TYPE_SIZE = 1
	CHUNK_MAX_SIZE  = 1024
	// These indices are relative to Body start
	DATUM_TYPE_INDEX     = 32
	DATUM_CONTENTS_INDEX = DATUM_TYPE_INDEX + DATUM_TYPE_SIZE
	BODY_START_INDEX     = 7

	FILENAME_MAX_SIZE    = 32
	DIRECTORY_ENTRY_SIZE = FILENAME_MAX_SIZE + HASH_SIZE

	MAX_DIRECTORY_CHILDREN = 16

	MIN_TREE_CHILDREN = 2
	MAX_TREE_CHILDREN = 32

	// Biggest message is datum chunk or bigfile with 32 children or full directory
	BODY_MAX_SIZE int = int(HASH_SIZE) + int(DATUM_TYPE_SIZE) + CHUNK_MAX_SIZE
)

const UDP_BUFFER_SIZE int = int(ID_SIZE) + int(TYPE_SIZE) + int(LENGTH_SIZE) +
	int(BODY_MAX_SIZE)

const (
	CLI_PROMPT            = "> "
	//EXIT_MESSAGE          = "Exiting gracefully"
	READLINE_HISTORY_FILE = "/tmp/readline_history"
)

type command struct {
	Name string
	Help string
	MinArgc int
	PcItem readline.PrefixCompleterInterface
}

// TODO Command name appears twice every time
// To add a new command: add it here and in the switch case of runLine(line string)
var CMD_MAP = map[string]command{
	"EXIT": {"exit", ": exits the program", 1, readline.PcItem("exit")},
	"HELP": {"help", ": shows help message", 1, readline.PcItem("help")},
	"LIST_PEERS": {"lspeers", ": shows the connected peers", 1, readline.PcItem("lspeers")}, // TODO Add --addr option
	"LIST_FILES": {"findrem", " PEER: shows the files shared by PEER", 2, readline.PcItem("findrem", readline.PcItemDynamic(peersListAutoComplete))},
	"CAT_FILE": {"curl", " PATH: downloads and shows the file at PATH", 2, readline.PcItem("curl", readline.PcItemDynamic(pathAutoComplete))},
	"DOWNLOAD_FILE": {"wget", " PATH: downloads recursively the directory or file at PATH", 2, readline.PcItem("wget", readline.PcItemDynamic(pathAutoComplete))},
	"HELLO": {"hello", " PEER: sends at least two Hellos to PEER", 2, readline.PcItem("hello", readline.PcItemDynamic(peersListAutoComplete))},
}

const CMD_TOO_FEW_ARGS = "Invalid line: too few arguments"

func byteToMsgTypeAsStr(msgType byte) (string, error) {
	var typeAsString string

	switch msgType {
	case NOOP:
		typeAsString = "NoOp"
	case ERROR:
		typeAsString = "Error"
	case ERROR_REPLY:
		typeAsString = "ErrorReply"
	case HELLO:
		typeAsString = "Hello"
	case HELLO_REPLY:
		typeAsString = "HelloReply"
	case PUBLIC_KEY:
		typeAsString = "PublicKey"
	case PUBLIC_KEY_REPLY:
		typeAsString = "PublicKeyReply"
	case ROOT:
		typeAsString = "Root"
	case ROOT_REPLY:
		typeAsString = "RootReply"
	case GET_DATUM:
		typeAsString = "GetDatum"
	case DATUM:
		typeAsString = "Datum"
	case NAT_TRAVERSAL_REQUEST:
		typeAsString = "NatTraversalRequest"
	case NO_DATUM:
		typeAsString = "NoDatum"
	case NAT_TRAVERSAL:
		typeAsString = "NatTraversal"
	default:
		typeAsString = "Unknown"
		return typeAsString, fmt.Errorf("unknown message type")
	}

	return typeAsString, nil
}

func byteToDatumTypeAsStr(datumType byte) (string, error) {
	var typeOfDatumAsString string

	switch datumType {
	case CHUNK:
		typeOfDatumAsString = "Chunk"
	case TREE:
		typeOfDatumAsString = "Tree/Big file"
	case DIRECTORY:
		typeOfDatumAsString = "Directory"
	default:
		typeOfDatumAsString = "Unknown"
		return typeOfDatumAsString, fmt.Errorf("unknown datum type")
	}

	return typeOfDatumAsString, nil
}
