package main

import (
	"container/list"
	"crypto/sha256"
	"fmt"
	"net"
	"sync"
	"time"
)

var connIPv4 *net.UDPConn
var connIPv6 *net.UDPConn

// Protected by a RWMutex
// If we received a message we assume the address is valid and add it to this map
// If we have sent a request and received a reply we consider the address valid and add it to this map
// If we don't receive a reply to a request after NUMBER_OF_REEMISSIONS we consider the address invalid and remove it from this map
// We remove addresses from the slice value but the key remains if the slice is empty
var peers map[string][]*net.UDPAddr
var peersMutex *sync.RWMutex

type addrUdpMsg struct {
	Addr *net.UDPAddr
	Msg  udpMsg
}

// TODO Rename msgQueue
// TODO Replace msgQueue with a map whose keys are struct {*net.UDPAddr, uint32 (Msg.Id)} and value is the udpMsg
var msgQueue *list.List
var msgQueueMutex *sync.RWMutex

func addAddrToPeers(peerName string, addr *net.UDPAddr) {
	createKeyValuePairInPeers(peerName)
	
	_ = removeAddrFromPeers(peerName, addr)

	peersMutex.Lock()
	peers[peerName] = append(peers[peerName], addr)
	peersMutex.Unlock()
}

// TODO Complete Javadoc comments everywhere

// Removes an element from a slice and returns the new slice
// From https://stackoverflow.com/a/37335777
// Assumes that peers is mutex locked
func removeFromAddrSlice(slice []*net.UDPAddr, index int) []*net.UDPAddr {
	slice[index] = slice[len(slice)-1]
    return slice[:len(slice)-1]
}

func removeAddrFromPeers(peerName string, addrToRemove *net.UDPAddr) error {
	peersMutex.Lock()
	defer peersMutex.Unlock()

	addresses, valueFound := peers[peerName]
	if ! valueFound {
		return fmt.Errorf("peer not found when trying to remove one of its addresses")
	}

	indexToRemove := -1
	for i, addr := range addresses {
		if compareUDPAddr(addrToRemove, addr) {
			indexToRemove = i
			break
		}
	}

	if indexToRemove == -1 {
		return fmt.Errorf("address to remove not found")
	}

	peers[peerName] = removeFromAddrSlice(addresses, indexToRemove)

	return nil
}

func createKeyValuePairInPeers(peerName string) {
	peersMutex.Lock()
	_, found := peers[peerName]
	if ! found {
		peers[peerName] = []*net.UDPAddr{}
	}
	peersMutex.Unlock()
}

func initUdp() error {
	msgQueue = list.New()
	msgQueueMutex = &sync.RWMutex{}

	peers = make(map[string][]*net.UDPAddr)
	peersMutex = &sync.RWMutex{}

	v4ListenAddr, err := net.ResolveUDPAddr("udp4", ":"+fmt.Sprint(UDP_LISTEN_PORT))
	if err != nil {
		return err
	}

	v6ListenAddr, err := net.ResolveUDPAddr("udp6", ":"+fmt.Sprint(UDP_LISTEN_PORT))
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

// This function is internal to udp.go
func receiveMsg() (addrUdpMsg, error) {
	buffer := make([]byte, UDP_BUFFER_SIZE)

	bytesRead, peerAddr, err := connIPv4.ReadFromUDP(buffer)
	if err != nil {
		return addrUdpMsg{}, err
	}

	receivedMsg, err := byteSliceToUdpMsg(buffer, bytesRead)
	if err != nil {
		return addrUdpMsg{}, err
	}

	if receivedMsg.Type == DATUM {
		err = checkDatumIntegrity(receivedMsg.Body)
		if err != nil {
			return addrUdpMsg{}, err
		}
	}

	return addrUdpMsg{peerAddr, receivedMsg}, nil
}

// Use only if a reply is not expected e.g. NoOp
func sendMsgToPeer(peerName string, toSend udpMsg) error {
	peerAddr, err := getAddressOfPeer(peerName)
	if err != nil {
		return err
	}

	return sendMsgToAddr(peerAddr, toSend)
}

// If using manually, use only when a reply is not expected e.g. NoOp
func sendMsgToAddr(peerAddr *net.UDPAddr, toSend udpMsg) error {
	// TODO Verify number of bytes written and underscores everywhere in the code
	_, err := connIPv4.WriteToUDP(udpMsgToByteSlice(toSend), peerAddr)
	return err
}

// Internal to udp.go
func handleMsg(receivedMsg addrUdpMsg) {
	shouldReply := true
	var replyMsg udpMsg

	if receivedMsg.Msg.Type >= FIRST_RESPONSE_MSG_TYPE {
		// TODO After some time remove messages that have not been retrieved from the message queue and log them
		threadSafeAppendToList(msgQueue, msgQueueMutex, receivedMsg)
		return
	}

	// The received message is a request
	switch receivedMsg.Msg.Type {
	case HELLO: // TODO Implement others
		hello, _ := parseHello(receivedMsg.Msg.Body)
		addAddrToPeers(hello.PeerName, receivedMsg.Addr)
		replyMsg, _ = createComplexHello(receivedMsg.Msg.Id, HELLO_REPLY)
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
		_ = sendMsgToAddr(receivedMsg.Addr, replyMsg)
	}
}

func listenAndRespond() {
	for {
		addrMsg, _ := receiveMsg()
		go handleMsg(addrMsg)
	}
}

func retrieveInMsgQueue(sentMsg addrUdpMsg) addrUdpMsg { // TODO Return error?

	var foundMsg *list.Element
	for {
		msgFound := false
		msgQueueMutex.RLock()
		for m := msgQueue.Front(); m != nil; m = m.Next() {
			mCasted := m.Value.(addrUdpMsg)
			if compareUDPAddr(mCasted.Addr, sentMsg.Addr) && mCasted.Msg.Id == sentMsg.Msg.Id {
				msgFound = true
				foundMsg = m
				break
			}
		}
		msgQueueMutex.RUnlock()
		if msgFound {
			break
		}
	}

	msgQueueMutex.Lock()
	msgQueue.Remove(foundMsg)
	msgQueueMutex.Unlock()

	return foundMsg.Value.(addrUdpMsg)
}

// TODO Reemissions here? -> return err after multiple retries
// TODO Verify that we verify that Length is sufficient or respect protocol generally
// TODO If we receive a datum, its integrity is verified?
// TODO No need to return addrUdpMsg, just return udpMsg?
// TODO Check that we don't send replies or requests without a reply e.g. NoOp
// Call this manually
// TODO Checks that a correct reply is indeed received
func sendAndReceiveMsg(peerName string, toSend udpMsg) (addrUdpMsg, error) {
	// TODO If we have never talked to peerName, send Hello, receive HelloReply, receive PublicKey and respond, receive Root and respond, only after this we can send toSend and receive the reply
	peerAddr, err := getAddressOfPeer(peerName)
	if err != nil {
		return addrUdpMsg{}, err
	}

	err = sendMsgToAddr(peerAddr, toSend)
	if err != nil {
		return addrUdpMsg{}, err
	}

	replyMsg := retrieveInMsgQueue(addrUdpMsg{peerAddr, toSend})

	// TODO We should verify that the type of the response corresponds to the request
	//reply - request = 127
	// TODO Check for NoDatum

	// TODO Print ErrorReply messages

	return replyMsg, nil
}

func downloadDatum(peerName string, hash []byte) (byte, interface{}, error) {
	getDatumMsg := createMsg(GET_DATUM, hash)
	datumReply, err := sendAndReceiveMsg(peerName, getDatumMsg)
	if err != nil {
		return 0, nil, err
	}

	return parseDatum(datumReply.Msg.Body)
}

// Must not stop e.g. internet connection stops and comes back 10 minutes after...
func keepAliveMainServer() {
    for {
        _, err := sendAndReceiveMsg(SERVER_PEER_NAME, createHello())
		if err != nil {
			LOGGING_FUNC(err)
		}
        
        time.Sleep(KEEP_ALIVE_PERIOD)
	}
}
