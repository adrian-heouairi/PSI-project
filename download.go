package main

import (
	"fmt"
	"os"
	"strings"
)

func writeBigFile(peerName string, datum datumTree, path string) error {
	for i, hash := range datum.ChildrenHashes {
		datumType, datumToCast, err := DownloadDatum(peerName, hash)
        fmt.Printf("\rDownloaded %d/%d chlidren of %s", i + 1, len(datum.ChildrenHashes), path)
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
	datumType, datumToCast, err := DownloadDatum(peerName, hash)
	if err != nil {
		return err
	}
	mkdir(replaceAllRegexBy(path, "/[^/]+$", ""))

	if datumType == DIRECTORY {
		datum := datumToCast.(datumDirectory)
        i := 0
		for key, value := range datum.Children {
			err := downloadRecursive(peerName, value, path+"/"+key)
			if err != nil {
				return err
			}
            i++
            fmt.Printf("\rDownloaded %d/%d chlidren of %s", i + 1, len(datum.Children), path)
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

func getPeerAllDataHashesRecursive(peerName string, hash []byte, path string, currentMap map[string][]byte) error {
	datumType, datumToCast, err := DownloadDatum(peerName, hash)
	if err != nil {
		return fmt.Errorf("Peer has invalid tree")
	}
	currentMap[path] = hash

	if datumType == DIRECTORY {
		datum := datumToCast.(datumDirectory)

		for key, value := range datum.Children { // TODO Sort keys by alphabetical order
			getPeerAllDataHashesRecursive(peerName, value, path+"/"+key, currentMap)
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
	err = getPeerAllDataHashesRecursive(peerName, root, strings.Replace(peerName, "/", "_", -1), res)
	if err != nil {
		return nil, err
	}
	return res, nil
}
