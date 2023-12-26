package main

import (
	"fmt"
)

func lsRecursive(hash []byte, depth int)(error) {
    datumReply,err := sendAndReceiveMsg(createMsg(GET_DATUM, hash))
    if err != nil {
        return err

    }
    datumType, datumToCast := parseDatum(datumReply.Body)
    
    if (datumType == DIRECTORY) {
        datum := datumToCast.(datumDirectory)

        //fmt.Print("Map : ")
        //fmt.Println(datum.Children)

        for key, value := range datum.Children {
            for i := 0; i < depth; i++ {
                fmt.Print("\t")
            }
            fmt.Println(key)
            lsRecursive(value,depth+1)
        }
    }
    return nil
}

func listAllFilesOfPeer(peer string) {
    RESTPeerRootHash := getRootOfPeer(peer)
    
    lsRecursive(RESTPeerRootHash,0)
}
