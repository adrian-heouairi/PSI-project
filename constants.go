package main

import (
	"log"
)

const SERVER_ADDRESS = "https://jch.irif.fr:8443"
const PEERS_PATH = "/peers/"
const OUR_PEER_NAME = "AS"
const SERVER_PEER_NAME = "jch.irif.fr"
const DOWNLOAD_DIR = "PSI-download"

const PRINT_MSG_BODY_TRUNCATE_SIZE = 100

var LOGGING_FUNC = log.Println

func checkErr(err error) {
	if err != nil {
		LOGGING_FUNC(err)
	}
}

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

	DATUM_TYPE_SIZE = 1
	CHUNK_MAX_SIZE  = 1024
	// These indices are relative to Body start
	DATUM_TYPE_INDEX = 32
	DATUM_CONTENTS_INDEX = DATUM_TYPE_INDEX + DATUM_TYPE_SIZE

	FILENAME_MAX_SIZE = 32
	DIRECTORY_ENTRY_SIZE = FILENAME_MAX_SIZE + HASH_SIZE

	MAX_DIRECTORY_CHILDREN = 16

	MIN_TREE_CHILDREN = 2
	MAX_TREE_CHILDREN = 32

	// Biggest message is datum chunk or bigfile with 32 children or full directory
	BODY_MAX_SIZE int = int(HASH_SIZE) + int(DATUM_TYPE_SIZE) + CHUNK_MAX_SIZE
)

const UDP_BUFFER_SIZE int = int(ID_SIZE) + int(TYPE_SIZE) + int(LENGTH_SIZE) +
	int(BODY_MAX_SIZE)

func byteToMsgTypeAsStr(msgType byte) string {
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
		LOGGING_FUNC("Unknown message type")
	}

	return typeAsString
}

func byteToDatumTypeAsStr(datumType byte) string {
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
		LOGGING_FUNC("Unknown datum type")
	}

	return typeOfDatumAsString
}
