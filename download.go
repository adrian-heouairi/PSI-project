package main

import (
	//"crypto/sha256"
)

func lsRecursive(hash []byte) {
    //datumReply := 
    sendAndReceiveMsg(createMsg(GET_DATUM, hash))
}

func listAllFilesOfPeer(peer string) {
    RESTPeerRootHash := getRootOfPeer(peer)
    
    lsRecursive(RESTPeerRootHash)
}
