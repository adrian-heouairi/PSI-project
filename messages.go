package main

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"math/rand"
	"net"
)

// TODO Transform some of these functions into methods

// It is assumed that len(Body) == Length
// Does not support cryptographic footer
type udpMsg struct {
	Id     uint32
	Type   uint8
	Length uint16
	Body   []byte
    Signature []byte // May be nil if peer doesn't sign messages
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
    if toCast.Signature != nil {
        res = append(res, toCast.Signature...)
    }
	return res
}

// Casts a byte slice to a udp message.
//   - toCast: the byte slice to be casted
//     it has the max length of a message
//   - bytesRead: the number of bytes that was received for this message
//   - Returns: a valid udpMsg and nil or an empty udpMsg and err
func byteSliceToUdpMsg(toCast []byte, bytesRead int) (udpMsg, error) {
	if len(toCast) < ID_SIZE + TYPE_SIZE + LENGTH_SIZE {
		return udpMsg{}, fmt.Errorf("Message too small")
	}

	var m udpMsg
	m.Id = binary.BigEndian.Uint32(toCast[:ID_SIZE])
	m.Type = toCast[ID_SIZE]

	_, err := byteToMsgTypeAsStr(m.Type)
	if err != nil {
		return udpMsg{}, err
	}

	m.Length = binary.BigEndian.Uint16(toCast[ID_SIZE+1 : ID_SIZE+1+LENGTH_SIZE])

	expectedSizeWithoutSig := ID_SIZE + TYPE_SIZE + LENGTH_SIZE + m.Length
	if bytesRead < int(expectedSizeWithoutSig) {
		return udpMsg{}, fmt.Errorf("UDP message too small: stated length %d but received %d bytes", m.Length, bytesRead)
	}

	m.Body = append([]byte{}, toCast[BODY_START_INDEX:BODY_START_INDEX+m.Length]...)

    if bytesRead == int(m.Length) + ID_SIZE + TYPE_SIZE + LENGTH_SIZE + SIGNATURE_SIZE {
        LOGGING_FUNC("MSG CONTAINS SIGNATURE")
        m.Signature = append(m.Signature, toCast[ID_SIZE + TYPE_SIZE + LENGTH_SIZE + m.Length:ID_SIZE + TYPE_SIZE + LENGTH_SIZE + m.Length + SIGNATURE_SIZE]...)
    } else if bytesRead != int(m.Length) + ID_SIZE + TYPE_SIZE + LENGTH_SIZE {
       return udpMsg{}, fmt.Errorf("Invalid message size")
    }

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
	return signMsg(msg)
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
				name := zeroPaddedByteSliceToString(msg.Body[startOffset+i*DIRECTORY_ENTRY_SIZE : startOffset+i*DIRECTORY_ENTRY_SIZE+FILENAME_MAX_SIZE])
				childrenNames += name + "\n\t"
			}
		}
	}

	return "Id: " + fmt.Sprint(msg.Id) + "\n" +
		"Type: " + typeAsString + "\n" +
		"Length: " + fmt.Sprint(msg.Length) + "\n" +
		"len(Body): " + fmt.Sprint(len(msg.Body)) + "\n" +
		"Abbreviated body as string: " + string(msg.Body[:lengthToTake]) + "\n" +
		"Abbreviated body bytes: " + fmt.Sprint(msg.Body[:lengthToTake]) +
		childrenNames
}

func udpMsgToStringShort(msg udpMsg) string {
	lengthToTake := len(msg.Body)
	if lengthToTake > PRINT_MSG_BODY_TRUNCATE_SIZE {
		lengthToTake = PRINT_MSG_BODY_TRUNCATE_SIZE
	}

	typeAsString, _ := byteToMsgTypeAsStr(msg.Type)

	return "Id: " + fmt.Sprint(msg.Id) + " " +
		"Type: " + typeAsString + " " +
		"Length: " + fmt.Sprint(msg.Length) + " " +
		"len(Body): " + fmt.Sprint(len(msg.Body)) + " " +
		"Abbreviated body as string: " + string(msg.Body[:lengthToTake]) + " " +
		"Abbreviated body bytes: " + fmt.Sprint(msg.Body[:lengthToTake])
}

// Parses a byte slice represneting a directory.
// - body: directory to be parsed
// - Returns: - a map containing the names and the hashes of the directory datum message
//   - nil in case of error
func parseDirectory(body []byte) (map[string][]byte, error) {
	if body[DATUM_TYPE_INDEX] != DIRECTORY {
		return nil, fmt.Errorf("not a directory")
	}

	res := make(map[string][]byte)

	nbEntry := (len(body) - int(DATUM_CONTENTS_INDEX)) / int(DIRECTORY_ENTRY_SIZE)

	if nbEntry < 0 || nbEntry > MAX_DIRECTORY_CHILDREN {
		return nil, fmt.Errorf("wrong number %d of children for directory", nbEntry)
	}

	for i := 0; i < int(nbEntry); i++ {
		keyStart := int(DATUM_CONTENTS_INDEX) + i*int(DIRECTORY_ENTRY_SIZE)
		valueStart := keyStart + FILENAME_MAX_SIZE
		filename := zeroPaddedByteSliceToString(body[keyStart:valueStart])

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
		return nil, fmt.Errorf("not a tree/big file")
	}

	res := [][]byte{}

	nbEntry := (len(body) - int(DATUM_CONTENTS_INDEX)) / int(HASH_SIZE)

	if nbEntry < MIN_TREE_CHILDREN || nbEntry > MAX_TREE_CHILDREN {
		return nil, fmt.Errorf("invalid number %d of children for tree/big file", nbEntry)
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
		return 0, nil, fmt.Errorf("invalid datum type")
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
	extensions := make([]byte, HELLO_EXTENSIONS_SIZE)
	nameAsBytes := []byte(OUR_PEER_NAME)
	var res = append(extensions, nameAsBytes...)
    helloMsg = createMsg(HELLO,res)
	return helloMsg
}

func createComplexHello(msgId uint32, msgType byte) (udpMsg, error) {
	if msgType != HELLO && msgType != HELLO_REPLY {
		msgTypeStr, _ := byteToMsgTypeAsStr(msgType)
		return udpMsg{}, fmt.Errorf("invalid message type %s (%d) when creating Hello/HelloReply", msgTypeStr, msgType)
	}

	ourHelloBody := hello{OUR_EXTENSIONS, OUR_PEER_NAME}

    ourHello := createMsgWithId(msgId, msgType, helloToByteSlice(ourHelloBody))
    return ourHello, nil
}

// TODO Use htons/htonl instead of BigEndian

// We never send NatTraversal, it is the main server who does it
func createNatTraversalRequestMsg(addr *net.UDPAddr) udpMsg {
	msg := createMsg(NAT_TRAVERSAL_REQUEST, udpAddrToByteSlice(addr))
	return msg
}

func checkMsgTypePair(sent uint8, received uint8) bool {
	//return (received == NO_DATUM && sent == GET_DATUM)
	return received-sent == MSG_VALID_PAIR
}

// Checks datum integrity.
// - body: message to be checked
// - Returns: error if data is not valid
// TODO Check that filenames in directory are valid UTF-8
func checkDatumIntegrity(body []byte) error {
	statedHash := body[:HASH_SIZE]

	computedHash := getHashOfByteSlice(body[DATUM_TYPE_INDEX:])

	if !bytes.Equal(statedHash, computedHash) {
		return fmt.Errorf("corrupted datum")
	}

	_, _, err := parseDatum(body)

	return err
}

func checkMsgIntegrity(msg udpMsg) error {
	switch msg.Type {
	case HELLO, HELLO_REPLY:
		_, err := parseHello(msg.Body)
		return err
	case DATUM:
		return checkDatumIntegrity(msg.Body)
	case NAT_TRAVERSAL_REQUEST, NAT_TRAVERSAL:
		if msg.Length != UDP_V4_SOCKET_SIZE && msg.Length != UDP_V6_SOCKET_SIZE {
			return fmt.Errorf("invalid NatTraversal[Request] size")
		}
	}

	return nil
}
