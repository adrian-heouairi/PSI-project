package main

import (
	"encoding/binary"
	"fmt"
	"math/rand"
)

type udpMsg struct {
	Id     uint32
	Type   uint8
	Length uint16
	Body   []byte
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
	var msg udpMsg
	msg.Id = rand.Uint32()
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

	return "Id: " + fmt.Sprint(msg.Id) + "\n" +
		"Type: " + typeAsString + "\n" +
		"Length: " + fmt.Sprint(msg.Length) + "\n" +
		"Body: " + string(msg.Body[:lengthToTake])
}
