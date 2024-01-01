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
var connIPv6 *net.UDPConn // TODO Implement IPv6 e.g. use the right conn according to the IP address (v4 or v6 of the query)

// TODO Send ErrorReply
// TODO Vérifier qu'on peut envoyer des messages à AS
// TODO Enlever une IP à laquelle on ne peut pas parler lors des réémissions

// Protected by a RWMutex
// If we received a Hello we send HelloReply and we assume the address is valid and add it to this map
// If we have sent a Hello and received a HelloReply we consider the address valid and add it to this map
// TODO If we don't receive a reply to a request after NUMBER_OF_REEMISSIONS we consider the address invalid and remove it from this map (and the key if the slice becomes empty)
// We don't remove an address from this map after 180 s with nothing sent or received
var peers map[string][]*net.UDPAddr
var peersMutex *sync.RWMutex

type addrUdpMsg struct {
	Addr *net.UDPAddr
	Msg  udpMsg
}

// TODO Replace msgQueue with a map whose keys are struct {*net.UDPAddr, uint32 (Msg.Id)} and value is *udpMsg. When we sent a request we add the corresponding key with a nil value and wait for the value to become not nil, we then retrieve the value and remove the key
var msgQueue *list.List
var msgQueueMutex *sync.RWMutex

func peersAddAddr(peerName string, addr *net.UDPAddr) {
	peersCreateKeyValuePairIfNotExist(peerName)

	_ = peersRemoveAddr(peerName, addr)

	peersMutex.Lock()
	peers[peerName] = append(peers[peerName], addr)
	peersMutex.Unlock()
}

func peersGet(key string) ([]*net.UDPAddr, bool) {
	peersMutex.RLock()
	defer peersMutex.RUnlock()

	value, found := peers[key]
	return value, found
}

// TODO Complete Javadoc comments everywhere

// Removes an element from a slice and returns the new slice, order after is arbitrary
// From https://stackoverflow.com/a/37335777
// Assumes that peers is mutex locked
func removeFromAddrSlice(slice []*net.UDPAddr, index int) []*net.UDPAddr {
	slice[index] = slice[len(slice)-1]
	return slice[:len(slice)-1]
}

// Removes the key if the value slice becomes empty
func peersRemoveAddr(peerName string, addrToRemove *net.UDPAddr) error {
	peersMutex.Lock()
	defer peersMutex.Unlock()

	addresses, valueFound := peers[peerName]
	if !valueFound {
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

	if len(peers[peerName]) == 0 {
		delete(peers, peerName)
	}

	return nil
}

func peersCreateKeyValuePairIfNotExist(peerName string) {
	peersMutex.Lock()
	_, found := peers[peerName]
	if !found {
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

// This function is internal to udp.go, used to receive all messages
// Called only by listenAndRespond
// Will return the message normally even for invalid messages e.g. Hello with empty body
func receiveAnyMsg() (addrUdpMsg, error) {
	buffer := make([]byte, UDP_BUFFER_SIZE)

	bytesRead, peerAddr, err := connIPv4.ReadFromUDP(buffer)
	if err != nil {
		return addrUdpMsg{}, err
	}

	receivedMsg, err := byteSliceToUdpMsg(buffer, bytesRead)
	if err != nil {
		return addrUdpMsg{}, err
	}

	return addrUdpMsg{peerAddr, receivedMsg}, nil
}

// Send a message and do not wait for a reply
func simpleSendMsgToAddr(peerAddr *net.UDPAddr, toSend udpMsg) error {
	// TODO Verify number of bytes written and underscores everywhere in the code
	_, err := connIPv4.WriteToUDP(udpMsgToByteSlice(toSend), peerAddr)
	return err
}

// Internal to udp.go
func handleMsg(receivedMsg addrUdpMsg) {
	if receivedMsg.Msg.Type >= FIRST_RESPONSE_MSG_TYPE {
		// TODO Remove from msgQueue after some time
		threadSafeAppendToList(msgQueue, msgQueueMutex, receivedMsg)
		return
	}

	// The received message is a request
	err := checkMsgIntegrity(receivedMsg.Msg)
	if err != nil {
		LOGGING_FUNC("not replying to invalid request received: " + udpMsgToString(receivedMsg.Msg))
		return
	}

	var replyMsg udpMsg
	switch receivedMsg.Msg.Type {
	case HELLO: // TODO Implement others
		hello, _ := parseHello(receivedMsg.Msg.Body)
		peersAddAddr(hello.PeerName, receivedMsg.Addr)
		replyMsg, _ = createComplexHello(receivedMsg.Msg.Id, HELLO_REPLY)
	case PUBLIC_KEY:
		replyMsg = createMsgWithId(receivedMsg.Msg.Id, PUBLIC_KEY_REPLY, []byte{})
	case ROOT:
		hasher := sha256.New()
		replyMsg = createMsgWithId(receivedMsg.Msg.Id, ROOT_REPLY, hasher.Sum(nil))
	case NAT_TRAVERSAL:
		peerAddr, _ := byteSliceToUDPAddr(receivedMsg.Msg.Body)
		_, err = sendToAddrAndReceiveMsgWithReemissions(peerAddr, createHello())
		if err == nil {
			LOGGING_FUNC("Received HelloReply from peer behind a NAT", peerAddr.String())
		}
		return
	default:
		LOGGING_FUNC("received request that we don't handle: " + udpMsgToString(receivedMsg.Msg))
		return
	}

	// Note that we reply to peers even if they have never sent Hello
	simpleSendMsgToAddr(receivedMsg.Addr, replyMsg)
}

func listenAndRespond() {
	for {
		addrMsg, err := receiveAnyMsg()
		if err == nil {
			go handleMsg(addrMsg)
		}
	}
}

func retrieveInMsgQueue(sentMsg addrUdpMsg) (addrUdpMsg,error) {
	var foundMsg *list.Element
    for i := 0; i < MSG_QUEUE_CHECK_NUMBER; i++ {
		msgQueueMutex.RLock()
		for m := msgQueue.Front(); m != nil; m = m.Next() {
			mCasted := m.Value.(addrUdpMsg)
			if compareUDPAddr(mCasted.Addr, sentMsg.Addr) && mCasted.Msg.Id == sentMsg.Msg.Id {
				foundMsg = m
				break
			}
		}
		msgQueueMutex.RUnlock()
		if foundMsg != nil {
			break
		}
        time.Sleep(MSG_QUEUE_CHECK_PERIOD)
	}

    if foundMsg != nil {
        msgQueueMutex.Lock()
        msgQueue.Remove(foundMsg)
        msgQueueMutex.Unlock()

        return foundMsg.Value.(addrUdpMsg), nil
    }
    return addrUdpMsg{}, fmt.Errorf("Msg not found in msg queue")
}

// TODO Check that we don't send replies or requests without a reply e.g. NoOp
// TODO The error returned should allow the caller to tell if NoDatum or ErrorReply
// This is not supposed to modify peers
func sendToAddrAndReceiveMsgWithReemissions(peerAddr *net.UDPAddr, toSend udpMsg) (udpMsg, error) {
    var retrieveErr error
    var replyMsg addrUdpMsg
    for i := 0; i < NUMBER_OF_REEMISSIONS + 1; i++ {
        if i != 0 {
            fmt.Printf("Reemission %d of ID %d\n", i, toSend.Id)
        }

        err := simpleSendMsgToAddr(peerAddr, toSend)
        if err != nil {
            return udpMsg{}, err
        }

		// The ID match check is here
        replyMsg, retrieveErr = retrieveInMsgQueue(addrUdpMsg{peerAddr, toSend})
        if retrieveErr == nil {
           break
        }
    }

    if retrieveErr != nil {
       return udpMsg{}, retrieveErr
    }

    err := checkMsgIntegrity(replyMsg.Msg)
    if err != nil {
        return udpMsg{}, err
    }

    // TODO Verify NatTraversal
    // TODO Reemit if ErrorReply?
    if !checkMsgTypePair(toSend.Type, replyMsg.Msg.Type) {
        return udpMsg{}, fmt.Errorf("invalid reply: " + udpMsgToString(replyMsg.Msg))
    }

    return replyMsg.Msg, nil
}

// Must not stop e.g. internet connection stops and comes back 10 minutes after...
// Keeps alive existing server addresses and new addresses obtained from REST
// TODO Redo this
// This maintains SERVER_PEER_NAME in peers, no other function should modify key SERVER_PEER_NAME in peers
func keepAliveMainPeer() {
	for {
		restMainPeerAddresses, err := restGetAddressesOfPeer(SERVER_PEER_NAME, false)
		currentMainPeerAddresses, found := peersGet(SERVER_PEER_NAME)

		allMainPeerAddresses := []*net.UDPAddr{}
		if err == nil {
			allMainPeerAddresses = appendAddressesIfNotPresent(allMainPeerAddresses, restMainPeerAddresses)
		}
		if found {
			allMainPeerAddresses = appendAddressesIfNotPresent(allMainPeerAddresses, currentMainPeerAddresses)
		}

		for _, a := range allMainPeerAddresses {
			if a.IP.To4() != nil { // If it is a v4 IP
				_, err := sendToAddrAndReceiveMsgWithReemissions(a, createHello())
				if err != nil {
					peersRemoveAddr(SERVER_PEER_NAME, a)
					LOGGING_FUNC("Main peer doesn't reply: ", err)
				} else {
					peersAddAddr(SERVER_PEER_NAME, a)
				}
			}
		}

		time.Sleep(KEEP_ALIVE_PERIOD)
	}
}

func natTraversal(addr *net.UDPAddr) error {
	natTraversalRequest := createNatTraversalRequestMsg(addr)

	mainPeerAddresses, found := peersGet(SERVER_PEER_NAME)
	if !found {
		return fmt.Errorf("no connection with main peer found during NAT traversal")
	}


	for i := 0; i < 2; i++ {
		simpleSendMsgToAddr(mainPeerAddresses[0], natTraversalRequest)
		_, err := sendToAddrAndReceiveMsgWithReemissions(addr, createHello())
		if err == nil {
			return nil
		}
	}

	return fmt.Errorf("NAT traversal failed")
}

////////////////////////////////////////////////// Below is API used by other files

// Can safely be used for SERVER_PEER_NAME (it should already be in peers, and anyways sending more Hellos is OK)
// TODO Check that we send a request that requires a reply
func ConnectAndSendAndReceive(peerName string, toSend udpMsg) (udpMsg, error) {
	addressesInPeers, found := peersGet(peerName)

	// If it is in peers we have already sent Hello before, first try to send toSend to the addresses already in peers

	if found {
		addressesInPeersCopy := []*net.UDPAddr{}
		addressesInPeersCopy = append(addressesInPeersCopy, addressesInPeers...)

		for _, a := range addressesInPeersCopy {
			replyMsg, err := sendToAddrAndReceiveMsgWithReemissions(a, toSend)
			if err != nil {
				peersRemoveAddr(peerName, a)
			} else {
				return replyMsg, nil
			}
		}
	}

	restPeerAddresses, err := restGetAddressesOfPeer(peerName, false)
	if err != nil {
		return udpMsg{}, err
	}

	for _, a := range restPeerAddresses {
		if a.IP.To4() != nil {
			_, helloWithoutNatErr := sendToAddrAndReceiveMsgWithReemissions(a, createHello())

			var natTraversalErr error
			if helloWithoutNatErr != nil {
				natTraversalErr = natTraversal(a)
			}

			if helloWithoutNatErr == nil || natTraversalErr == nil {
				replyMsg, err := sendToAddrAndReceiveMsgWithReemissions(a, toSend)
				if err == nil {
					peersAddAddr(peerName, a)
					return replyMsg, nil
				}
			}
		}
	}

	return udpMsg{}, fmt.Errorf("can't resolve peer named %s", peerName)
}

func DownloadDatum(peerName string, hash []byte) (byte, interface{}, error) {
	getDatumMsg := createMsg(GET_DATUM, hash)
	datumReply, err := ConnectAndSendAndReceive(peerName, getDatumMsg)
	if err != nil {
		return 0, nil, err
	}

	return parseDatum(datumReply.Body)
}
