package main

import (
	"bytes"
	"crypto/sha256"
	"encoding/binary"
	"fmt"
	"math/rand"
)

// It is assumed that len(Body) == Length
// Does not support cryptographic footer
type udpMsg struct {
	Id     uint32
	Type   uint8
	Length uint16
	Body   []byte
}

// Represent a datum message containing a chunk.
// - StatedHash: represents the received hash that should be the same as the one computed using Contents
type datumChunk struct {
	StatedHash []byte
	Type       byte
	Contents   []byte
}

// Represent a datum message containing a Tree/BigFile.
// - StatedHash: represents the received hash that should be the same as the one computed using ChildrenHashes
type datumTree struct {
	StatedHash     []byte
	Type           byte
	ChildrenHashes [][]byte
}

// Represent a datum message containing a Directory.
// - StatedHash: represents the received hash that should be the same as the one computed using Children
type datumDirectory struct {
	StatedHash []byte
	Type       byte
	Children   map[string][]byte // Map of filename (no \0 padding) -> hash
}

type hello struct {
	Extensions uint32
	PeerName   string
}

// Castes a udpMsg to byte slice ready to be sent.
// - toCast: the message to be casted.
// - Returns: a byte slice that can be sent through network.
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

// Casts a byte slice to a udp message.
//   - toCast: the byte slice to be casted
//     it has the max length of a message
//   - bytesRead: the number of bytes that was received for this message
//   - Returns: a valid udpMsg and nil or an empty udpMsg and err
func byteSliceToUdpMsg(toCast []byte, bytesRead int) (udpMsg, error) {
	var m udpMsg
	m.Id = binary.BigEndian.Uint32(toCast[:ID_SIZE])
	m.Type = toCast[ID_SIZE]

	_, err := byteToMsgTypeAsStr(m.Type)
	if err != nil {
		return udpMsg{}, err
	}

	m.Length = binary.BigEndian.Uint16(toCast[ID_SIZE+1 : ID_SIZE+1+LENGTH_SIZE])
	expectedSize := ID_SIZE + TYPE_SIZE + LENGTH_SIZE + m.Length
	if bytesRead < int(expectedSize) {
		return udpMsg{}, fmt.Errorf("UDP message too small: stated length %d but received %d bytes", m.Length, bytesRead)
	}

	m.Body = append([]byte{}, toCast[BODY_START_INDEX:BODY_START_INDEX+m.Length]...)
	return m, nil
}

// Creates a valid udp message with random id and the given type and body.
// - msgType: valid type of message
// - msgBody: valid body of message
// - Returns: a valid udp message
func createMsg(msgType byte, msgBody []byte) udpMsg {
	return createMsgWithId(rand.Uint32(), msgType, msgBody)
}

// Like createMsg but specifying id.
// - msgId: the id of the message
// - msgType: valid type of message
// - msgBody: valid body of message
// - Returns: a valid udp message
func createMsgWithId(msgId uint32, msgType byte, msgBody []byte) udpMsg {
	var msg udpMsg
	msg.Id = msgId
	msg.Type = msgType
	msg.Body = msgBody
	msg.Length = uint16(len(msgBody))

	return msg
}

// Human readable representation of a udp message.
// - msg: fully valid udp message
// - Returns: a string describing msg
func udpMsgToString(msg udpMsg) string {
	lengthToTake := len(msg.Body)
	if lengthToTake > PRINT_MSG_BODY_TRUNCATE_SIZE {
		lengthToTake = PRINT_MSG_BODY_TRUNCATE_SIZE
	}

	typeAsString, _ := byteToMsgTypeAsStr(msg.Type)

	childrenNames := ""
	if msg.Type == DATUM {
		datumType, _ := byteToDatumTypeAsStr(msg.Body[DATUM_TYPE_INDEX])
		typeAsString += " " + datumType
		if msg.Body[DATUM_TYPE_INDEX] == DIRECTORY {
			childrenNames = "\nChildren:\n\t"
			nbEntry := (len(msg.Body) - int(DATUM_CONTENTS_INDEX)) / int(DIRECTORY_ENTRY_SIZE)
			startOffset := DATUM_CONTENTS_INDEX
			for i := 0; i < nbEntry; i++ {
				name, err := byteSliceToStringWithoutTrailingZeroes(msg.Body[startOffset+i*DIRECTORY_ENTRY_SIZE : startOffset+i*DIRECTORY_ENTRY_SIZE+FILENAME_MAX_SIZE])
				if err != nil {
					LOGGING_FUNC(err)
					return ""
				}
				childrenNames += name + "\n\t"
			}
		}
	}

	return "Id: " + fmt.Sprint(msg.Id) + "\n" +
		"Type: " + typeAsString + "\n" +
		"Length: " + fmt.Sprint(msg.Length) + "\n" +
		"Body: " + string(msg.Body[:lengthToTake]) +
		childrenNames
}

// Checks datum integrity.
// - body: message to be checked
// - Returns: error if data is not valid
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

// Removes the trailing zeroes from name.
// - name: from which to remove \0s
// - Returns: a valid string or error if data is not valid
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

// Parses a byte slice represneting a directory.
// - body: directory to be parsed
// - Returns: - a map containing the names and the hashes of the directory datum message
//   - nil in case of error
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
		keyStart := int(DATUM_CONTENTS_INDEX) + i*int(DIRECTORY_ENTRY_SIZE)
		valueStart := keyStart + FILENAME_MAX_SIZE
		filename, err := byteSliceToStringWithoutTrailingZeroes(body[keyStart:valueStart])

		if err != nil {
			return nil, err
		}

		res[filename] = body[valueStart : valueStart+HASH_SIZE]
	}

	return res, nil
}

// Parses a byte slice represneting a big file.
// - body: big file to be parsed
// - Returns: - a slice of slices of byte containing the hashes of children
//   - nil in case of error
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
		hashStart := int(DATUM_CONTENTS_INDEX) + i*int(HASH_SIZE)
		res = append(res, body[hashStart:hashStart+HASH_SIZE])
	}

	return res, nil
}

// Parses a byte slice represneting a datum.
// - body: datum to be parsed
// - Returns: - the type of the datum message
//   - a slice of slices of byte containing the hashes of children
//   - nil in case of error
//
// We assume that the udpMsg that is parsed will not be modified
func parseDatum(body []byte) (byte, interface{}, error) {
	datumType := body[DATUM_TYPE_INDEX]
	statedHash := body[:HASH_SIZE]

	switch datumType {
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

func helloToByteSlice(h hello) []byte {
	res := make([]byte, HELLO_EXTENSIONS_SIZE)

	binary.BigEndian.PutUint32(res, h.Extensions)

	res = append(res, []byte(h.PeerName)...)

	return res
}

func parseHello(body []byte) (hello, error) {
	length := len(body)

	if length < HELLO_EXTENSIONS_SIZE+1 {
		return hello{}, fmt.Errorf("Hello[Reply] is too short")
	}

	extensions := binary.BigEndian.Uint32(body[:HELLO_EXTENSIONS_SIZE])
	peerName := string(body[HELLO_EXTENSIONS_SIZE:])

	return hello{extensions, peerName}, nil
}

// Creates a valid hello message containing our peer name.
// - Returns: a valid hello udpMsg
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

func createComplexHello(msgId uint32, msgType byte) (udpMsg, error) {
	if msgType != HELLO && msgType != HELLO_REPLY {
		msgTypeStr, _ := byteToMsgTypeAsStr(msgType)
		return udpMsg{}, fmt.Errorf("invalid message type %s (%d) when creating Hello/HelloReply", msgTypeStr, msgType)
	}

	// TODO 0 is our extensions, replace it with constant
	ourHelloBody := hello{0, OUR_PEER_NAME}

	return createMsgWithId(msgId, msgType, helloToByteSlice(ourHelloBody)), nil
}
