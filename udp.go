package main

import (
	"container/list"
	"fmt"
	"net"
	"os"
	"sync"
	"time"
    "slices"
)

var connIPv4 *net.UDPConn

//var connIPv6 *net.UDPConn // TODO Implement IPv6 e.g. use the right conn according to the IP address (v4 or v6 of the query)

// TODO Send ErrorReply

// Protected by a RWMutex
// If we received a Hello we send HelloReply and we assume the address is valid and add it to this map
// If we have sent a Hello and received a HelloReply we consider the address valid and add it to this map
// If we don't receive a reply to a request after NUMBER_OF_REEMISSIONS we consider the address invalid and remove it from this map (and the key if the slice becomes empty)
// We don't remove an address from this map after 180 s with nothing sent or received
var peers map[string][]*net.UDPAddr
var peersMutex *sync.RWMutex

type addrUdpMsg struct {
    Addr *net.UDPAddr
    Msg  udpMsg
}

// TODO Replace msgQueue with a map whose keys are struct {*net.UDPAddr, uint32 (Msg.Id)} and value is *udpMsg. When we sent a request we add the corresponding key with a nil value and wait for the value to become not nil, we then retrieve the value and remove the key
// If replies are sent when we haven't made a request, the replies accumulate forever in the message queue (fixed by the above task)
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

func peersGetKeyFromVal(addr *net.UDPAddr) string {
    peersMutex.RLock()
    defer peersMutex.RUnlock()

    for k, v := range peers {
        if addrIsInSlice(v, addr) {
            return k
        }
    }
    return ""
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
    LOGGING_FUNC("Binding", v4ListenAddr.String())

    peersAddAddr(OUR_PEER_NAME, &net.UDPAddr{IP: []byte{127, 0, 0, 1}, Port: UDP_LISTEN_PORT})

    /*v6ListenAddr, err := net.ResolveUDPAddr("udp6", ":"+fmt.Sprint(UDP_LISTEN_PORT))
    if err != nil {
        return err
    }
    LOGGING_FUNC("Binding", v6ListenAddr.String())*/

    connIPv4, err = net.ListenUDP("udp4", v4ListenAddr)
    if err != nil {
        return err
    }

    /*connIPv6, err = net.ListenUDP("udp6", v6ListenAddr)
    if err != nil {
        return err
    }*/

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
        LOGGING_FUNC("Appending to msgQueue", udpMsgToStringShort(receivedMsg.Msg))
        threadSafeAppendToList(msgQueue, msgQueueMutex, receivedMsg)
        return
    }

    // The received message is a request
    err := checkMsgIntegrity(receivedMsg.Msg)
    if err != nil {
        LOGGING_FUNC("not replying to invalid request received: " + udpMsgToString(receivedMsg.Msg))
        return
    }

    var peerName string
    if receivedMsg.Msg.Type == HELLO {
        hello, _ := parseHello(receivedMsg.Msg.Body)
        peerName = hello.PeerName
    } else {
        peerName = peersGetKeyFromVal(receivedMsg.Addr)
    }

    peerPublicKey := []byte{}
    if peerName != "" {
        peerPublicKey = restGetKey(peerName)
    }

    if receivedMsg.Msg.Signature != nil {
        if len(peerPublicKey) == SIGNATURE_SIZE {
            if !checkMsgSignature(receivedMsg.Msg, peerPublicKey) {
                LOGGING_FUNC("Bad signature in received request")
                return
            } else {
                LOGGING_FUNC("Successfully verified signature of request")
            }
        } else {
            LOGGING_FUNC("Received signed request but couldn't get peer key")
            return
        }
    }

    if len(peerPublicKey) == SIGNATURE_SIZE && receivedMsg.Msg.Signature == nil && slices.Contains(MANDATORILY_SIGNED_MSGS, receivedMsg.Msg.Type) {
        LOGGING_FUNC("Peer that implements cryptography sent an unsigned request of a type that must be signed")
        return
    }

    var replyMsg udpMsg
    switch receivedMsg.Msg.Type {
    case NOOP:
        return
    case HELLO:
        replyMsg, _ = createComplexHello(receivedMsg.Msg.Id, HELLO_REPLY)
        parsedHello, _ := parseHello(receivedMsg.Msg.Body)
        peersAddAddr(parsedHello.PeerName, receivedMsg.Addr)
    case PUBLIC_KEY:
        replyMsg = createMsgWithId(receivedMsg.Msg.Id, PUBLIC_KEY_REPLY, publicKeyToHexaString())
    case ROOT:
        replyMsg = createMsgWithId(receivedMsg.Msg.Id, ROOT_REPLY, ourTree.Hash)
    case GET_DATUM:
        value, found := ourTreeMap[string(receivedMsg.Msg.Body)]
        if found {
            replyMsg, err = value.toDatum(receivedMsg.Msg.Id)
            if err != nil {
                LOGGING_FUNC(err)
                return
            }
        } else {
            replyMsg = createMsgWithId(receivedMsg.Msg.Id, NO_DATUM, receivedMsg.Msg.Body)
        }
    case NAT_TRAVERSAL:
        if receivedMsg.Msg.Length != IPV4_SIZE + PORT_SIZE { // TODO IPv6
            LOGGING_FUNC("Received a NAT traversal request with an IPv6, ignoring")
            return
        }
        peerAddr, _ := byteSliceToUDPAddr(receivedMsg.Msg.Body)

        LOGGING_FUNC("NAT traversal started by peer", peerAddr.String())

        var theirNatTraversalErr error
        for i := 0; i < NAT_TRAVERSAL_RETRIES; i++ {
            _, theirNatTraversalErr = sendToAddrAndReceiveMsgWithReemissions(peerAddr, createHello())
            // A SOFT error is not tolerated
            if theirNatTraversalErr == nil {
                break
            }
        }
        if theirNatTraversalErr == nil {
            LOGGING_FUNC("Received HelloReply from peer", peerAddr.String(), "after they started a NAT traversal")
        } else {
            LOGGING_FUNC("NAT traversal started by", peerAddr.String(), "failed")
        }
        return
    default:
        LOGGING_FUNC("received request that we don't handle: " + udpMsgToString(receivedMsg.Msg))
        return
    }

    if DEBUG {
        t, _ := byteToMsgTypeAsStr(receivedMsg.Msg.Type)
        t2, _ := byteToMsgTypeAsStr(replyMsg.Type)
        fmt.Printf("From %s: received ID %d, sent ID %d, received type %s, sent type %s\n", receivedMsg.Addr.String(), receivedMsg.Msg.Id, replyMsg.Id, t, t2)
    }

    // Note that we reply to peers even if they have never sent Hello
    simpleSendMsgToAddr(receivedMsg.Addr, replyMsg)
}

func listenAndRespond() {
    for {
        addrMsg, err := receiveAnyMsg()
        if err == nil {
            go handleMsg(addrMsg)
        } else {
            LOGGING_FUNC(err)
        }
    }
}

func retrieveInMsgQueue(sentMsg addrUdpMsg) (addrUdpMsg, error) {
    var foundMsg *list.Element

    startTime := time.Now()

    for time.Since(startTime) < MSG_QUEUE_MAX_WAIT {
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
    }

    if foundMsg != nil {
        msgQueueMutex.Lock()
        msgQueue.Remove(foundMsg)
        msgQueueMutex.Unlock()

        return foundMsg.Value.(addrUdpMsg), nil
    }
    return addrUdpMsg{}, fmt.Errorf("msg not found in msg queue")
}

// TODO Check that we don't send replies or requests without a reply e.g. NoOp (verify toSend.Type)
// This is not supposed to modify peers
// This function has errors that start by "SOFT ", they mean that a reply was received but it was invalid. If an error is not "SOFT ", assume that a reply was not received.
func sendToAddrAndReceiveMsgWithReemissions(peerAddr *net.UDPAddr, toSend udpMsg) (udpMsg, error) {
    var retrieveErr error
    var replyMsg addrUdpMsg
    for i := 0; i < NUMBER_OF_REEMISSIONS+1; i++ {
        if i != 0 {
            LOGGING_FUNC_F("Reemission %d of ID %d\n", i, toSend.Id)
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

    if DEBUG {
        t, _ := byteToMsgTypeAsStr(toSend.Type)
        t2, _ := byteToMsgTypeAsStr(replyMsg.Msg.Type)
        fmt.Printf("To %s: sent ID %d, received ID %d, sent type %s, received type %s\n", peerAddr.String(), toSend.Id, replyMsg.Msg.Id, t, t2)
    }

    if replyMsg.Msg.Type == ERROR_REPLY {
        fmt.Fprintln(os.Stderr, udpMsgToString(replyMsg.Msg))
    }
    
    /* else if replyMsg.Msg.Type == HELLO_REPLY {
        if replyMsg.Msg.Signature != nil {
            hello, err := parseHello(replyMsg.Msg.Body)
            if err != nil {
                return udpMsg{}, err
            }
            pk := restGetKey(hello.PeerName)
            if len(pk) == 0 {
                return udpMsg{}, fmt.Errorf("Could not retrevie public key from REST API")
            }
            if !checkMsgSignature(replyMsg.Msg, pk) {
                return udpMsg{}, err
            }
        }
    } */

    err := checkMsgIntegrity(replyMsg.Msg)
    if err != nil {
        return udpMsg{}, fmt.Errorf("SOFT " + err.Error())
    }

    // TODO Reemit if ErrorReply?
    if !checkMsgTypePair(toSend.Type, replyMsg.Msg.Type) {
        return udpMsg{}, fmt.Errorf("SOFT reply doesn't match type of pair: " + udpMsgToString(replyMsg.Msg))
    }

    var peerName string
    if replyMsg.Msg.Type == HELLO_REPLY {
        hello, _ := parseHello(replyMsg.Msg.Body)
        peerName = hello.PeerName
    } else {
        peerName = peersGetKeyFromVal(replyMsg.Addr)
    }

    peerPublicKey := []byte{}
    if peerName != "" {
        peerPublicKey = restGetKey(peerName)
    }

    if replyMsg.Msg.Signature != nil {
        if len(peerPublicKey) == SIGNATURE_SIZE {
            if !checkMsgSignature(replyMsg.Msg, peerPublicKey) {
                return udpMsg{}, fmt.Errorf("bad signature in received reply")
            } else {
                LOGGING_FUNC("Successfully verified signature of reply message")
            }
        } else {
            return udpMsg{}, fmt.Errorf("received signed reply but couldn't get peer key")
        }
    }

    if len(peerPublicKey) == SIGNATURE_SIZE && replyMsg.Msg.Signature == nil && slices.Contains(MANDATORILY_SIGNED_MSGS, replyMsg.Msg.Type) {
        return udpMsg{}, fmt.Errorf("peer that implements cryptography sent an unsigned reply of a type that must be signed")
    }

    return replyMsg.Msg, nil
}

// Must not stop e.g. internet connection stops and comes back 10 minutes after...
// Keeps alive existing server addresses and new addresses obtained from REST
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
                    LOGGING_FUNC("Main peer doesn't reply or replies incorrectly: ", err)
                } else {
                    peersAddAddr(SERVER_PEER_NAME, a)
                }
            }
        }

        time.Sleep(KEEP_ALIVE_PERIOD)
    }
}

func natTraversal(addr *net.UDPAddr) error {
    LOGGING_FUNC("Starting NAT traversal with peer", addr.String())

    natTraversalRequest := createNatTraversalRequestMsg(addr)

    mainPeerAddresses, found := peersGet(SERVER_PEER_NAME)
    if !found {
        return fmt.Errorf("no connection with main peer found during our NAT traversal")
    }

    var err2 error
    for i := 0; i < NAT_TRAVERSAL_RETRIES; i++ {
        simpleSendMsgToAddr(mainPeerAddresses[0], natTraversalRequest)
        _, err2 = sendToAddrAndReceiveMsgWithReemissions(addr, createHello())
        if err2 == nil {
            return nil
        }
    }

    return fmt.Errorf("our NAT traversal failed (%s)", err2.Error())
}

////////////////////////////////////////////////// Below is API used by other files

// Can safely be used for SERVER_PEER_NAME or OUR_PEER_NAME (they should already be in peers, and anyways sending more Hellos is OK)
// TODO Check that we send a request that requires a reply
// Returns an error starting by "SOFT " if a reply was received but it was invalid e.g. NoDatum
func ConnectAndSendAndReceive(peerName string, toSend udpMsg) (udpMsg, error) {
    addressesInPeers, found := peersGet(peerName)

    // If it is in peers we have already sent Hello before, first try to send toSend to the addresses already in peers

    if found {
        addressesInPeersCopy := []*net.UDPAddr{}
        addressesInPeersCopy = append(addressesInPeersCopy, addressesInPeers...)

        for _, a := range addressesInPeersCopy {
            replyMsg, err := sendToAddrAndReceiveMsgWithReemissions(a, toSend)
            if err != nil && !grep("^SOFT ", err.Error()) {
                LOGGING_FUNC("Removing address", a, "from peers because of HARD error", err)
                peersRemoveAddr(peerName, a)
            } else if err != nil && grep("^SOFT ", err.Error()) {
                LOGGING_FUNC("SOFT error, not removing address from peers:", err)
                return replyMsg, err
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
                if natTraversalErr == nil {
                    LOGGING_FUNC("NAT traversal started by us succeeded for", a.String())
                } else {
                    LOGGING_FUNC("NAT traversal started by us failed for", a.String())
                }
            }

            if helloWithoutNatErr == nil || natTraversalErr == nil {
                replyMsg, err := sendToAddrAndReceiveMsgWithReemissions(a, toSend)
                // If HARD error do nothing
                if err != nil && grep("^SOFT ", err.Error()) {
                    peersAddAddr(peerName, a)
                    return replyMsg, err
                } else if err == nil {
                    peersAddAddr(peerName, a)
                    return replyMsg, nil
                }
            }
        }
    }

    return udpMsg{}, fmt.Errorf("can't resolve or communicate with peer %s", peerName)
}

func DownloadDatum(peerName string, hash []byte) (byte, interface{}, error) {
    getDatumMsg := createMsg(GET_DATUM, hash)
    datumReply, err := ConnectAndSendAndReceive(peerName, getDatumMsg)
    if err != nil {
        return 0, nil, err
    }

    return parseDatum(datumReply.Body)
}

// TODO Return error if hash of empty string
func GetRootOfPeerUDPThenREST(peerName string) ([]byte, error) {
    rootMsg := createMsg(ROOT, ourTree.Hash)
    rootReplyMsg, err := ConnectAndSendAndReceive(peerName, rootMsg)
    if err != nil {
        LOGGING_FUNC(err)
        return restGetRootOfPeer(peerName)
    }
    return rootReplyMsg.Body, nil
}
