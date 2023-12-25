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

func byteSliceToUdpMsg(toCast []byte) udpMsg {
	var m udpMsg
	m.Id = binary.BigEndian.Uint32(toCast[0:4])
	m.Type = toCast[4]
	m.Length = binary.BigEndian.Uint16(toCast[5:7])
	m.Body = append([]byte{}, toCast[7:7+m.Length]...)
	return m
}

func createHello() udpMsg {
	var helloMsg udpMsg
	helloMsg.Id = rand.Uint32()
	helloMsg.Type = 2
	extensions := make([]byte, 4)
	name := OUR_PEER_NAME
	nameAsBytes := []byte(name)
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

func udpMsgToString(msg udpMsg) string {
	lengthToTake := len(msg.Body)
	if lengthToTake > PRINT_MSG_BODY_TRUNCATE_SIZE {
		lengthToTake = PRINT_MSG_BODY_TRUNCATE_SIZE
	}

	typeAsString := byteToMsgTypeAsStr(msg.Type)

	if msg.Type == DATUM {
		typeAsString += " " + byteToDatumTypeAsStr(msg.Body[DATUM_TYPE_INDEX])
	}

    // TODO If datum directory, print the names inside

	return "Id: " + fmt.Sprint(msg.Id) + "\n" +
		"Type: " + typeAsString + "\n" +
		"Length: " + fmt.Sprint(msg.Length) + "\n" +
		"Body: " + string(msg.Body[:lengthToTake])
}

func checkDatumIntegrity(body []byte) {
    statedHash := body[:HASH_SIZE]

	hasher := sha256.New()
	hasher.Write(body[DATUM_TYPE_INDEX:])
	computedHash := hasher.Sum(nil)

    if !bytes.Equal(statedHash, computedHash) {
        LOGGING_FUNC("Corrupted datum")
    }
}

func byteSliceToStringWithoutTrailingZeroes(name []byte) string {
	i := 0
	for name[i] != 0 {
		i++
	}
	return string(name[:i])
}

// Returns a map containing the names and the hashes of the directory datum message
// Returns nil in case of error
func parseDirectory(body []byte) map[string][]byte {
    if body[DATUM_TYPE_INDEX] != DIRECTORY {
        LOGGING_FUNC("Not a directory")
        return nil
    }

	res := make(map[string][]byte)
	
    nbEntry := (len(body) - int(DATUM_CONTENTS_INDEX)) / int(DIRECTORY_ENTRY_SIZE)

    if nbEntry < 0 || nbEntry > MAX_DIRECTORY_CHILDREN {
        LOGGING_FUNC("Invalid directory")
        return nil
    }

    for i := 0; i < int(nbEntry); i++ {
        keyStart := int(DATUM_CONTENTS_INDEX) + i * int(DIRECTORY_ENTRY_SIZE)
        valueStart := keyStart + FILENAME_MAX_SIZE
        res[byteSliceToStringWithoutTrailingZeroes(body[keyStart:valueStart])] = body[valueStart:valueStart + HASH_SIZE]
    }

	return res
}

func parseTree(body []byte) [][]byte {
    if body[DATUM_TYPE_INDEX] != TREE {
        LOGGING_FUNC("Not a tree/big file")
        return nil
    }

	res := [][]byte{}
	
    nbEntry := (len(body) - int(DATUM_CONTENTS_INDEX)) / int(HASH_SIZE)

    if nbEntry < MIN_TREE_CHILDREN || nbEntry > MAX_TREE_CHILDREN {
        LOGGING_FUNC("Invalid tree/big file")
        return nil
    }

    for i := 0; i < int(nbEntry); i++ {
        hashStart := int(DATUM_CONTENTS_INDEX) + i * int(HASH_SIZE)
        res = append(res, body[hashStart:hashStart + HASH_SIZE])
    }

	return res
}

// We assume that the udpMsg that is parsed will not be modified
func parseDatum(body []byte) interface{} {
    datumType := body[DATUM_TYPE_INDEX]
    statedHash := body[:HASH_SIZE]

    switch(datumType) {
        case CHUNK:
            return datumChunk{statedHash, datumType, body[DATUM_CONTENTS_INDEX:]}
        case TREE:
            return datumTree{statedHash, datumType, parseTree(body)}
        case DIRECTORY:
            return datumDirectory{statedHash, datumType, parseDirectory(body)}
        default:
            LOGGING_FUNC("Invalid datum type")
            return nil
    }
}

func sendAndReceiveMsg(toSend udpMsg) udpMsg {
    sendMsg(toSend)
    replyMsg := receiveMsg()

    // TODO We should verify that the type of the response corresponds to the request

    if toSend.Id != replyMsg.Id {
        LOGGING_FUNC("Query and reply IDs don't match")
    }

    return replyMsg
}

func sendMsg(toSend udpMsg) {
    _, err := jchConn.Write(udpMsgToByteSlice(toSend))
    checkErr(err)
}

func receiveMsg() udpMsg {
    buffer := make([]byte, UDP_BUFFER_SIZE)

    _, err := jchConn.Read(buffer)
    checkErr(err)

    replyMsg := byteSliceToUdpMsg(buffer)

    if replyMsg.Type == DATUM {
        checkDatumIntegrity(replyMsg.Body)
    }

    return replyMsg
}
