package main

import (
	"fmt"
    "net"
//	"os"
)

// Shows all the names of the elements in the directory.
//  - hash: of the directory
//  - depth: number of tabs to print before the name
//  Returns : error if communication with peer was impossible or the sent datum is invalid
func lsRecursive(hash []byte, depth int) error {
	serverUdpAddresses, err := getAdressesOfPeer(SERVER_PEER_NAME)
	checkErr(err)
	
	jchAddr, err := net.ResolveUDPAddr("udp4", serverUdpAddresses[0])
	checkErr(err)
    datumReply, err := sendAndReceiveMsg(addrUdpMsg{jchAddr, createMsg(GET_DATUM, hash)})
    if err != nil {
        return err
    }

    datumType, datumToCast, err := parseDatum(datumReply.Msg.Body)
    if err != nil {
        LOGGING_FUNC("Peer has invalid tree")
        return err
    }
    
    if (datumType == DIRECTORY) {
        datum := datumToCast.(datumDirectory)

        for key, value := range datum.Children { // TODO Sort keys by alphabetical order
            for i := 0; i < depth; i++ {
                fmt.Print("\t")
            }
            fmt.Println(key)

            err := lsRecursive(value, depth + 1)
            if err != nil {
                return err
            }
        }
    }

    return nil
}

// Wrapper for lsRecursive.
//  - peer: name of the peer we want to see root structure
//  Returns : error if we could not retrive root or lsRecursive returns error
func listAllFilesOfPeer(peer string) error {
    RESTPeerRootHash, err := getRootOfPeer(peer) // TODO This should try REST and UDP
    if err != nil {
        return err
    }
    
    return lsRecursive(RESTPeerRootHash, 0)
}
/*
func writeBigFile(datum datumTree, path string) error {
    for _, hash := range datum.ChildrenHashes {
        datumType, datumToCast, err := downloadDatum(hash)
        if err != nil {
            return err
        }

        if datumType == CHUNK {
            datum := datumToCast.(datumChunk)
            writeChunk(path, datum.Contents)
        } else if datumType == TREE {
            datum := datumToCast.(datumTree)
            writeBigFile(datum, path)
        } else {
            return fmt.Errorf("Children of big file should be big file or chunk")
        }
    }

    return nil
}

func downloadRecursive(hash []byte, path string) error {
    datumType, datumToCast, err := downloadDatum(hash)
    if err != nil {
        return err
    }
    
    if (datumType == DIRECTORY) {
        datum := datumToCast.(datumDirectory)

        mkdir(path)

        for key, value := range datum.Children { // TODO Sort keys by alphabetical order
            err := downloadRecursive(value, path + "/" + key)
            if err != nil {
                return err
            }
        }
    } else if datumType == CHUNK {
        datum := datumToCast.(datumChunk)
        os.Remove(path)
        writeChunk(path, datum.Contents)
    } else { // Tree/big file
        datum := datumToCast.(datumTree)

        os.Remove(path)

        err = writeBigFile(datum, path)
        if err != nil {
            return err
        }
    }

    return nil
}

func downloadFullTreeOfPeer(peer string) error {
    RESTPeerRootHash, err := getRootOfPeer(peer) // TODO This should try REST and UDP
    if err != nil {
        return err
    }
    // TODO Replace slashes by underscore in peer name
    return downloadRecursive(RESTPeerRootHash, peer)
}
*/
