package main

import (
	"container/list"
	"crypto/sha256"
	"fmt"
	"net"
	"reflect"
	"sync"
)

var connIPv4 *net.UDPConn
var connIPv6 *net.UDPConn

type addrUdpMsg struct {
	Addr *net.UDPAddr
	Msg udpMsg
}

var msgQueue *list.List
var msgQueueMutex *sync.RWMutex

func initUdp() error {
	msgQueue = list.New()
	msgQueueMutex = &sync.RWMutex{}

	v4ListenAddr, err := net.ResolveUDPAddr("udp4", ":" + fmt.Sprint(UDP_LISTEN_PORT))
	if err != nil {
		return err
	}

	v6ListenAddr, err := net.ResolveUDPAddr("udp6", ":" + fmt.Sprint(UDP_LISTEN_PORT))
	if err != nil {
		return err
	}

	connIPv4, err = net.ListenUDP("udp4", v4ListenAddr)
	if err != nil {
		return err
	}

	connIPv6, err = net.ListenUDP("udp6", v6ListenAddr)
	if err != nil {
		return err
	}
	
	// TODO defer conn.Close()

	return nil
}

func receiveMsg() (addrUdpMsg, error) {
    buffer := make([]byte, UDP_BUFFER_SIZE)

    bytesRead, peerAddr, err := connIPv4.ReadFromUDP(buffer)
    if err != nil {
		return addrUdpMsg{}, err
	}

    replyMsg, err := byteSliceToUdpMsg(buffer, bytesRead)
	if err != nil {
		return addrUdpMsg{}, err
	}

    if replyMsg.Type == DATUM {
        err = checkDatumIntegrity(replyMsg.Body)
		if err != nil {
			return addrUdpMsg{}, err
		}
    }

    return addrUdpMsg{peerAddr, replyMsg}, nil
}

func sendMsg(toSend addrUdpMsg) error {
	// TODO Verify number of bytes written and underscores everywhere in the code
    _, err := connIPv4.WriteToUDP(udpMsgToByteSlice(toSend.Msg), toSend.Addr)
    return err
}

func handleMsg(receivedMsg addrUdpMsg) {
	shouldReply := true
	var replyMsg udpMsg

	if receivedMsg.Msg.Type >= FIRST_RESPONSE_MSG_TYPE {
		threadSafeAppendToList(msgQueue, msgQueueMutex, receivedMsg)
		return
	}

	// The received message is a request
	switch receivedMsg.Msg.Type {
	//case HELLO: // TODO Implement this and others
		//sendMsg()
	case PUBLIC_KEY:
		replyMsg = createMsgWithId(receivedMsg.Msg.Id, PUBLIC_KEY_REPLY, []byte{})
	case ROOT:
		hasher := sha256.New()
		replyMsg = createMsgWithId(receivedMsg.Msg.Id, ROOT_REPLY, hasher.Sum(nil))
	default:
		shouldReply = false
		LOGGING_FUNC(udpMsgToString(receivedMsg.Msg))
	}

	if shouldReply {
		_ = sendMsg(addrUdpMsg{receivedMsg.Addr, replyMsg})
	}
}

func listenAndRespond() {
	for {
		addrMsg, _ := receiveMsg()
		go handleMsg(addrMsg)
	}
}

func retrieveInMsgQueue(sentMsg addrUdpMsg) addrUdpMsg { // TODO Return error?
	msgQueueMutex.RLock()
	
	var foundMsg *list.Element
	for {
		msgFound := false
		for m := msgQueue.Front(); m != nil; m = m.Next() {
			mCasted := m.Value.(addrUdpMsg)
			if reflect.DeepEqual(mCasted.Addr, sentMsg.Addr) && mCasted.Msg.Id == sentMsg.Msg.Id {
				msgFound = true
				foundMsg = m
				break
			}
		}
		if msgFound {
			break
		}
	}
	msgQueueMutex.RUnlock()

	msgQueueMutex.Lock()
	msgQueue.Remove(foundMsg)
	msgQueueMutex.Unlock()

	return foundMsg.Value.(addrUdpMsg)
}

// TODO Improve this comment
// Returns error if peer does not respond after multiple retries or if peer
// does not respect the protocol e.g. Length field doesn't match Body length
func sendAndReceiveMsg(toSend addrUdpMsg) (addrUdpMsg, error) {
    err := sendMsg(toSend)
	if err != nil {
		return addrUdpMsg{}, err
	}

    replyMsg := retrieveInMsgQueue(toSend)

    // TODO We should verify that the type of the response corresponds to the request
	// TODO Check for NoDatum

	// TODO Print ErrorReply messages

    return replyMsg, nil
}

func downloadDatum(peerAddr *net.UDPAddr, hash []byte) (byte, interface{}, error) {
	getDatumMsg := createMsg(GET_DATUM, hash)
	datumReply, err := sendAndReceiveMsg(addrUdpMsg{peerAddr, getDatumMsg})
    if err != nil {
        return 0, nil, err
    }

    return parseDatum(datumReply.Msg.Body)
}
