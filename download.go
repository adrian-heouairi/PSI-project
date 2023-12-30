package main

import (
	"fmt"
	"os"
    "strings"
)

// Shows all the names of the elements in the directory.
//   - peerName: the peer whose tree we are interested in
//   - hash: of the directory
//   - depth: number of tabs to print before the name
//     Returns : error if communication with peer was impossible or the sent datum is invalid
func lsRecursive(peerName string, hash []byte, depth int) error {
	datumType, datumToCast, err := downloadDatum(peerName, hash)
	if err != nil {
		return fmt.Errorf("Peer has invalid tree")
	}

	if datumType == DIRECTORY {
		datum := datumToCast.(datumDirectory)

		for key, value := range datum.Children { // TODO Sort keys by alphabetical order
			for i := 0; i < depth; i++ {
				fmt.Print("\t")
			}
			fmt.Println(key)

			err := lsRecursive(peerName, value, depth + 1)
			if err != nil {
				return err
			}
		}
	}
	return nil
}

// Wrapper for lsRecursive.
//   - peer: name of the peer we want to see root structure
//     Returns : error if we could not retrive root or lsRecursive returns error
func listAllFilesOfPeer(peer string) error {
	RESTPeerRootHash, err := restGetRootOfPeer(peer) // TODO This should try REST and UDP
	if err != nil {
		return err
	}

	return lsRecursive(peer, RESTPeerRootHash, 0)
}


func writeBigFile(peerName string, datum datumTree, path string) error {
    for _, hash := range datum.ChildrenHashes {
        datumType, datumToCast, err := downloadDatum(peerName, hash)
        if err != nil {
            return err
        }

        if datumType == CHUNK {
            newDatum := datumToCast.(datumChunk)
            writeChunk(path, newDatum.Contents)
        } else if datumType == TREE {
            newDatum := datumToCast.(datumTree)
            writeBigFile(peerName, newDatum, path)
        } else {
            return fmt.Errorf("Children of big file should be big file or chunk")
        }
    }

    return nil
}

// TODO handle case where an file becomes a directory
func downloadRecursive(peerName string, hash []byte, path string) error {
    fmt.Println("Downloading ", path)
    datumType, datumToCast, err := downloadDatum(peerName, hash)
    if err != nil {
        return err
    }
    fmt.Println(path, "->", replaceAllRegexBy(path, "/[^/]+$", ""))
    mkdir(replaceAllRegexBy(path, "/[^/]+$", ""))

    if (datumType == DIRECTORY) {
        datum := datumToCast.(datumDirectory)

        for key, value := range datum.Children {
            err := downloadRecursive(peerName, value, path + "/" + key)
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

        err = writeBigFile(peerName, datum, path)
        if err != nil {
            return err
        }
    }

    return nil
}

func downloadFullTreeOfPeer(peerName string) error {
    RESTPeerRootHash, err := restGetRootOfPeer(peerName) // TODO This should try REST and UDP
    if err != nil {
        return err
    }
    peerDirectory := strings.Replace(peerName, "/", "_", -1)
    return downloadRecursive(peerName, RESTPeerRootHash, peerDirectory)
}

func getPeerAllDataHashesRecursive(peerName string, hash []byte, path string, currentMap map[string][]byte) error{
	datumType, datumToCast, err := downloadDatum(peerName, hash)
	if err != nil {
		return fmt.Errorf("Peer has invalid tree")
	}
    currentMap[path] = hash

	if datumType == DIRECTORY {
		datum := datumToCast.(datumDirectory)

		for key, value := range datum.Children { // TODO Sort keys by alphabetical order
            getPeerAllDataHashesRecursive(peerName, value, path + "/" + key, currentMap)
		}
    }
    return nil
}

func getPeerAllDataHashes(peerName string) (map[string][]byte, error) {
    res := make(map[string][]byte)
    root, err := restGetRootOfPeer(peerName)
    if err != nil {
        return nil, err
    }
    err = getPeerAllDataHashesRecursive(peerName,root,strings.Replace(peerName, "/", "_", -1),res)
    if err != nil {
        return nil, err
    }
    return res, nil
}
