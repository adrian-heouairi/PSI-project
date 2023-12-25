package main

import (
	"fmt"
)

func lsRecursive(hash []byte) {
    datumReply := sendAndReceiveMsg(createMsg(GET_DATUM, hash))
    datumType, datumToCast := parseDatum(datumReply.Body)
    
    if (datumType == DIRECTORY) {
        datum := datumToCast.(datumDirectory)

        fmt.Print("Map : ")
        fmt.Println(datum.Children)

        for _, value := range datum.Children {
            lsRecursive(value)
        }
    }
}

func listAllFilesOfPeer(peer string) {
    RESTPeerRootHash := getRootOfPeer(peer)
    
    lsRecursive(RESTPeerRootHash)
}
