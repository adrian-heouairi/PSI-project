package main

import (
	"fmt"
	"net"
	"strings"
)

// Displays connected peers.
// Returns: -error if server is not available
func restGetPeers(show bool) ([]string, error) {
	resp, bodyAsByteSlice, err := httpGet(SERVER_ADDRESS + PEERS_PATH)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode != HTTP_OK {
		return nil, fmt.Errorf("code other than HTTP OK received")
	}

	if show {
		fmt.Println(strings.TrimSpace(string(bodyAsByteSlice)))
	}

	peers := strings.Split(string(bodyAsByteSlice), "\n")
	return peers[:len(peers)-1], nil
}

// Gives the addresses of the given peer.
// - peerName: the peer whose addresses we want
// Returns: - a slice with the peer addresses
//   - error if peer was not found
func restGetAddressesOfPeer(peerName string, display bool) ([]*net.UDPAddr, error) {
	resp, bodyAsByteSlice, err := httpGet(SERVER_ADDRESS + PEERS_PATH + "/" + peerName + "/addresses")
	if err != nil {
		return nil, err
	}

	if resp.StatusCode == HTTP_NOT_FOUND {
		return nil, fmt.Errorf(peerName + " is not known by server")
	}

	if resp.StatusCode != HTTP_OK {
		return nil, fmt.Errorf("code other than HTTP OK received")
	}

	if display {
		fmt.Println(string(bodyAsByteSlice))
	}
	addrAsStrings := strings.Split(string(bodyAsByteSlice[:len(bodyAsByteSlice)-1]), "\n")

	if len(addrAsStrings) == 0 {
		return nil, fmt.Errorf("REST API: peer exists but has no addresses")
	}

	res := []*net.UDPAddr{}
	for _, s := range addrAsStrings {
		addr, err := net.ResolveUDPAddr("udp", s)
		if err != nil {
			return nil, err
		}
		res = append(res, addr)
	}
	return res, nil
}

// Gives the hash of the peer's root
// - peerName: the peer whose root we want
// Returns: - the root hash
//   - error if peer does not exist or the main server is not available
func restGetRootOfPeer(peerName string) ([]byte, error) {
	resp, bodyAsByteSlice, err := httpGet(SERVER_ADDRESS + PEERS_PATH + "/" + peerName + "/root")
	if err != nil {
		return nil, err
	}

	if resp.StatusCode == HTTP_NO_CONTENT {
		return nil, fmt.Errorf(peerName + " has not declared a root yet")
	} else if resp.StatusCode == HTTP_NOT_FOUND {
		return nil, fmt.Errorf(peerName + " is not known by server")
	}

	if resp.StatusCode != HTTP_OK {
		return nil, fmt.Errorf("code other than HTTP OK received")
	}

	return bodyAsByteSlice, nil
}

func restDisplayAllpeersWithTheirAddresses () {
    var res string
    var addrOfPeer string
    peers, err := restGetPeers(false)
    if err != nil {
       return 
    }
    for _, peerName := range peers {
        addrOfPeer = ""
        addrs, err := restGetAddressesOfPeer(peerName, false)
        if err != nil {
            return 
        }
        for _, a := range addrs {
           addrOfPeer += a.String() + " "
        }
        res += peerName + ": " + addrOfPeer + "\n"
    }
    fmt.Println(res)
}
