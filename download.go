package main

/*import (
	"fmt"
	"os"
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

func listAllFilesOfPeer(peer string) error {
    RESTPeerRootHash, err := getRootOfPeer(peer) // TODO This should try REST and UDP
    if err != nil {
        return err
    }
    
    return lsRecursive(RESTPeerRootHash, 0)
}

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