package main

import (
	"fmt"
	"os"
	"strings"
)

// TODO Handle peers whose root is not a DIRECTORY datum

func writeBigFile(peerName string, datum datumTree, path string, depth int) error {
	for i, hash := range datum.ChildrenHashes {
		if depth == 0 {
			LOGGING_FUNC_F("Downloading big file %s whose root has %d children\n", path, len(datum.ChildrenHashes))
			progressPercentage := int(float32(i) / float32(len(datum.ChildrenHashes)) * float32(100))
			fmt.Printf("\rDownloading big file %s: %d %%", path, progressPercentage)
		}

		datumType, datumToCast, err := DownloadDatum(peerName, hash)
		if err != nil {
			return err
		}

		if datumType == CHUNK {
			newDatum := datumToCast.(datumChunk)
			writeChunk(path, newDatum.Contents)
		} else if datumType == TREE {
			newDatum := datumToCast.(datumTree)
			writeBigFile(peerName, newDatum, path, depth+1)
		} else {
			return fmt.Errorf("children of big file should be big file or chunk")
		}
	}

	if depth == 0 {
		fmt.Printf("\rDownloading big file %s: 100 %%\n", path)
	}

	return nil
}

// TODO Handle case where a file becomes a directory (peer updated their tree)
func downloadRecursive(peerName string, hash []byte, path string) error {
	datumType, datumToCast, err := DownloadDatum(peerName, hash)
	if err != nil {
		return err
	}
	mkdirP(replaceAllRegexBy(path, "/[^/]+$", ""))

	if datumType == DIRECTORY {
		datum := datumToCast.(datumDirectory)

		fmt.Println("Creating directory", path)

		mkdirP(path)

		i := 0
		for key, value := range datum.Children {
			err := downloadRecursive(peerName, value, path+"/"+key)
			if err != nil {
				return err
			}
			i++
		}
	} else if datumType == CHUNK {
		datum := datumToCast.(datumChunk)

		fmt.Println("Downloading single-chunk file", path)

		os.Remove(path)
		writeChunk(path, datum.Contents)
	} else { // Tree/big file
		datum := datumToCast.(datumTree)

		os.Remove(path)

		err = writeBigFile(peerName, datum, path, 0)
		if err != nil {
			return err
		}
	}

	return nil
}

func getPeerPathHashMapRecursive(peerName string, hash []byte, path string, currentMap map[string][]byte) error {
	datumType, datumToCast, err := DownloadDatum(peerName, hash)
	if err != nil {
		return err
	}
	currentMap[path] = hash

	if datumType == DIRECTORY {
		datum := datumToCast.(datumDirectory)

		for key, value := range datum.Children {
			err = getPeerPathHashMapRecursive(peerName, value, path+"/"+key, currentMap)
			if err != nil {
				return err
			}
		}
	}
	return nil
}

func getPeerPathHashMap(peerName string) (map[string][]byte, error) {
	res := make(map[string][]byte)
	root, err := GetRootOfPeerUDPThenREST(peerName)
	if err != nil {
		return nil, err
	}
	err = getPeerPathHashMapRecursive(peerName, root, strings.Replace(peerName, "/", "_", -1), res)
	if err != nil {
		return nil, err
	}
	return res, nil
}
