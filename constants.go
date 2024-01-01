package main

import (
	"fmt"
	"log"
	"time"
)

const SERVER_ADDRESS = "https://jch.irif.fr:8443"
const PEERS_PATH = "/peers/"
const OUR_PEER_NAME = "AS"
const SERVER_PEER_NAME = "jch.irif.fr"
const DOWNLOAD_DIR = "PSI-download"
const UDP_LISTEN_PORT = 8444
const KEEP_ALIVE_PERIOD = 30 * time.Second

// To achieve 500 ms of waiting for a reply before reemitting the request
const MSG_QUEUE_CHECK_PERIOD = 2 * time.Millisecond
const MSG_QUEUE_CHECK_NUMBER = 250

const NUMBER_OF_REEMISSIONS = 4

const MSG_QUEUE_SIZE = 1024

const PRINT_MSG_BODY_TRUNCATE_SIZE = 100

var LOGGING_FUNC = log.Println

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
	EXIT_CMD          = "exit"
	HELP_CMD          = "help"
	LIST_PEERS_CMD    = "lspeers" // TODO Add --addr option
	LIST_FILES_CMD    = "findrem"
	CAT_FILE_CMD      = "curl"
	DOWNLOAD_FILE_CMD = "wget"

	CLI_PROMPT            = "> "
	EXIT_MESSAGE          = "Exiting gracefully"
	READLINE_HISTORY_FILE = "/tmp/readline_history"
)

var CMD_LIST = []string{EXIT_CMD, HELP_CMD, LIST_PEERS_CMD, LIST_FILES_CMD, CAT_FILE_CMD, DOWNLOAD_FILE_CMD}

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
		return typeAsString, fmt.Errorf("Unknown message type")
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
		return typeOfDatumAsString, fmt.Errorf("Unknown datum type")
	}

	return typeOfDatumAsString, nil
}
