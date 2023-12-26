package main

import (
	"fmt"
)

func lsRecursive(hash []byte, depth int) error {
    datumReply, err := sendAndReceiveMsg(createMsg(GET_DATUM, hash))
    if err != nil {
        return err
    }

    datumType, datumToCast, err := parseDatum(datumReply.Body)
    if err != nil {
        LOGGING_FUNC("Peer has invalid tree")
        return err
    }
    
    if (datumType == DIRECTORY) {
        datum := datumToCast.(datumDirectory)

        for key, value := range datum.Children {
            for i := 0; i < depth; i++ {
                fmt.Print("\t")
            }
            fmt.Println(key)

            return lsRecursive(value, depth + 1)
        }
    }

    return nil
}

func listAllFilesOfPeer(peer string) error {
    RESTPeerRootHash, err := getRootOfPeer(peer) // TODO This should try REST and UDP
    if err != nil {
        return err
    }
    
    return lsRecursive(RESTPeerRootHash, 0)
}
