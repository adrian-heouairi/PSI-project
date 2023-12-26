package main

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"math/rand"
    "crypto/sha256"
)

// It is assumed that len(Body) == Length
// Does not support cryptographic footer
type udpMsg struct {
	Id     uint32
	Type   uint8
	Length uint16
	Body   []byte
}

type datumChunk struct {
	StatedHash []byte
	Type byte
    Contents []byte
}

type datumTree struct {
	StatedHash []byte
	Type byte
    ChildrenHashes [][]byte
}

type datumDirectory struct {
	StatedHash []byte
	Type byte
    Children map[string][]byte // Map of filename (no \0 padding) -> hash
}

func udpMsgToByteSlice(toCast udpMsg) []byte {
	var idToByteSlice []byte = make([]byte, 4)
	binary.BigEndian.PutUint32(idToByteSlice, toCast.Id)
	var typeToByteSlice []byte = make([]byte, 1)
	typeToByteSlice[0] = toCast.Type
	var lengthToByteSlice []byte = make([]byte, 2)
	binary.BigEndian.PutUint16(lengthToByteSlice, toCast.Length)
	var res = append(idToByteSlice, typeToByteSlice...)
	res = append(res, lengthToByteSlice...)
	res = append(res, toCast.Body...)
	return res
}

// toCast is the buffer, it has the max length of a message
func byteSliceToUdpMsg(toCast []byte, bytesRead int) (udpMsg, error) {
	// TODO Replace indices by constants
	var m udpMsg
	m.Id = binary.BigEndian.Uint32(toCast[0:4])
	m.Type = toCast[4]

	_, err := byteToMsgTypeAsStr(m.Type)
	if err != nil {
		return udpMsg{}, err
	}

	m.Length = binary.BigEndian.Uint16(toCast[5:7])
	expectedSize := ID_SIZE + TYPE_SIZE + LENGTH_SIZE + m.Length
	if bytesRead < int(expectedSize) {
		return udpMsg{}, fmt.Errorf("UDP message too small: stated length %d but received %d bytes", m.Length, bytesRead)
	}

	m.Body = append([]byte{}, toCast[7:7+m.Length]...)
	return m, nil
}

func createHello() udpMsg {
	var helloMsg udpMsg
	helloMsg.Id = rand.Uint32()
	helloMsg.Type = HELLO
	extensions := make([]byte, HELLO_EXTENSIONS_SIZE)
	nameAsBytes := []byte(OUR_PEER_NAME)
	var res = append(extensions, nameAsBytes...)
	helloMsg.Body = res
	helloMsg.Length = uint16(len(res))
	return helloMsg
}

func createMsg(msgType byte, msgBody []byte) udpMsg {
    return createMsgWithId(rand.Uint32(), msgType, msgBody)
}

func createMsgWithId(msgId uint32, msgType byte, msgBody []byte) udpMsg {
	var msg udpMsg
	msg.Id = msgId
	msg.Type = msgType
	msg.Body = msgBody
	msg.Length = uint16(len(msgBody))

	return msg
}

// Assumes that msg is fully valid
func udpMsgToString(msg udpMsg) string {
	lengthToTake := len(msg.Body)
	if lengthToTake > PRINT_MSG_BODY_TRUNCATE_SIZE {
		lengthToTake = PRINT_MSG_BODY_TRUNCATE_SIZE
	}

	typeAsString, _ := byteToMsgTypeAsStr(msg.Type)

	if msg.Type == DATUM {
		datumType, _ := byteToDatumTypeAsStr(msg.Body[DATUM_TYPE_INDEX])
		typeAsString += " " + datumType
	}

    // TODO If datum directory, print the names inside

	return "Id: " + fmt.Sprint(msg.Id) + "\n" +
		"Type: " + typeAsString + "\n" +
		"Length: " + fmt.Sprint(msg.Length) + "\n" +
		"Body: " + string(msg.Body[:lengthToTake])
}

func checkDatumIntegrity(body []byte) error {
    statedHash := body[:HASH_SIZE]

	hasher := sha256.New()
	hasher.Write(body[DATUM_TYPE_INDEX:])
	computedHash := hasher.Sum(nil)

    if !bytes.Equal(statedHash, computedHash) {
        return fmt.Errorf("Corrupted datum")
    }

	return nil
}

func byteSliceToStringWithoutTrailingZeroes(name []byte) (string, error) {
	i := 0
	for name[i] != 0 {
		i++
	}

	if i == 0 {
		return "", fmt.Errorf("Empty filenames are not allowed")
	}

	return string(name[:i]), nil
}

// Returns a map containing the names and the hashes of the directory datum message
// Returns nil in case of error
func parseDirectory(body []byte) (map[string][]byte, error) {
    if body[DATUM_TYPE_INDEX] != DIRECTORY {
        return nil, fmt.Errorf("Not a directory")
    }

	res := make(map[string][]byte)
	
    nbEntry := (len(body) - int(DATUM_CONTENTS_INDEX)) / int(DIRECTORY_ENTRY_SIZE)

    if nbEntry < 0 || nbEntry > MAX_DIRECTORY_CHILDREN {
        return nil, fmt.Errorf("Wrong number %d of children for directory", nbEntry)
    }

    for i := 0; i < int(nbEntry); i++ {
        keyStart := int(DATUM_CONTENTS_INDEX) + i * int(DIRECTORY_ENTRY_SIZE)
        valueStart := keyStart + FILENAME_MAX_SIZE
		filename, err := byteSliceToStringWithoutTrailingZeroes(body[keyStart:valueStart])

		if err != nil {
			return nil, err
		}

        res[filename] = body[valueStart:valueStart + HASH_SIZE]
    }

	return res, nil
}

func parseTree(body []byte) ([][]byte, error) {
    if body[DATUM_TYPE_INDEX] != TREE {
        return nil, fmt.Errorf("Not a tree/big file")
    }

	res := [][]byte{}
	
    nbEntry := (len(body) - int(DATUM_CONTENTS_INDEX)) / int(HASH_SIZE)

    if nbEntry < MIN_TREE_CHILDREN || nbEntry > MAX_TREE_CHILDREN {
        return nil, fmt.Errorf("Invalid number %d of children for tree/big file", nbEntry)
    }

    for i := 0; i < int(nbEntry); i++ {
        hashStart := int(DATUM_CONTENTS_INDEX) + i * int(HASH_SIZE)
        res = append(res, body[hashStart:hashStart + HASH_SIZE])
    }

	return res, nil
}

// We assume that the udpMsg that is parsed will not be modified
func parseDatum(body []byte) (byte, interface{}, error) {
    datumType := body[DATUM_TYPE_INDEX]
    statedHash := body[:HASH_SIZE]

    switch(datumType) {
        case CHUNK:
            return datumType, datumChunk{statedHash, datumType, body[DATUM_CONTENTS_INDEX:]}, nil
        case TREE:
			hashList, err := parseTree(body)

			if err != nil {
				return 0, nil, err
			}

            return datumType, datumTree{statedHash, datumType, hashList}, nil
        case DIRECTORY:
			filenameHashMap, err := parseDirectory(body)

			if err != nil {
				return 0, nil, err
			}

            return datumType, datumDirectory{statedHash, datumType, filenameHashMap}, nil
        default:
            return 0, nil, fmt.Errorf("Invalid datum type")
    }
}

// Returns error if peer does not respond after multiple retries or if peer
// does not respect the protocol e.g. Length field doesn't match Body length
func sendAndReceiveMsg(toSend udpMsg) (udpMsg, error) {
    err := sendMsg(toSend)
	if err != nil {
		return udpMsg{}, err
	}

    replyMsg, err := receiveMsg()
	if err != nil {
		return udpMsg{}, err
	}

    // TODO We should verify that the type of the response corresponds to the request

	// TODO Print ErrorReply messages

    if toSend.Id != replyMsg.Id {
		return replyMsg, fmt.Errorf("Query and reply IDs don't match")
    }

    return replyMsg, nil
}

func sendMsg(toSend udpMsg) error {
    _, err := jchConn.Write(udpMsgToByteSlice(toSend))
    return err
}

func receiveMsg() (udpMsg, error) {
    buffer := make([]byte, UDP_BUFFER_SIZE)

    bytesRead, err := jchConn.Read(buffer)
    if err != nil {
		return udpMsg{}, err
	}

    replyMsg, err := byteSliceToUdpMsg(buffer, bytesRead)
	if err != nil {
		return udpMsg{}, err
	}

    if replyMsg.Type == DATUM {
        err = checkDatumIntegrity(replyMsg.Body)
		if err != nil {
			return udpMsg{}, err
		}
    }

    return replyMsg, nil
}
