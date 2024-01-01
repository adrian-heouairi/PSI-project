package main

import (
	"crypto/sha256"
	"fmt"
	"os"
)

// May have index 0 for empty files
type chunk struct {
    Path string
    Index int
}

type merkleTreeNode struct {
    // Depends on Type:
    //  - CHUNK: {path, id}
    //  - BIG_FILE: 2 <= len(ChildrenNodes) <= 32
    //  - DIRECTORY: 0 <= len(ChildrenNodes) <= 16

    // The parent node of the current one useful for hash computation
    // Root parent node is root
    Parent *merkleTreeNode
    // Never nil
    ChildrenNodes []*merkleTreeNode
    // Never nil, len == 32
    Hash []byte
    Type byte // CHUNK, TREE, DIRECTORY
    // nil if not CHUNK
    ChunkContent *chunk
    // There are no \0 at the end of children names, this field is nil if not DIRECTORY
    DirectoryChildrenNames [][]byte
}

var ourTree merkleTreeNode

// First call is supposed to be done on the directory represneting root and parent is the same as the current node we try to produce.
func pathToMerkleTreeWithoutHashComputation(path string, parent *merkleTreeNode) (*merkleTreeNode, error) {
    fileInfo, err := os.Stat(path)
    if err != nil {
        return nil, err
    }

    ret := &merkleTreeNode{ChildrenNodes: []*merkleTreeNode{}, Hash: make([]byte, HASH_SIZE), Parent: parent}

    if fileInfo.IsDir() {
        ret.Type = DIRECTORY
        ret.DirectoryChildrenNames = [][]byte{}

        entries, err := os.ReadDir(path)
        if err != nil {
            return nil, err
        }

        for _, entry := range entries {
            ret.DirectoryChildrenNames = append(ret.DirectoryChildrenNames, stringToZeroPaddedByteSlice(entry.Name()))
            recursiveCall, err := pathToMerkleTreeWithoutHashComputation(path + entry.Name(), ret)
            if err != nil {
                return nil, err
            }
            ret.ChildrenNodes = append(ret.ChildrenNodes, recursiveCall)
        }
    } else {
        if fileInfo.Size() <= CHUNK_MAX_SIZE {
            ret.Type = CHUNK
            ret.ChunkContent = &chunk{path, 0}
        } else {
            // TODO BIG_FILE
            return nil, fmt.Errorf("BIG_FILE not implemented")
        }
    }

    return ret, nil
}

// Returns the hash and the content of the chunk at index chunkIndex in the file at path
func chunkFile(path string, chunkIndex int64) ([]byte, []byte, error) {
    fi, err := os.Stat(path)
    if err != nil {
        return nil, nil, err
    }

    if fi.IsDir() {
        return nil, nil, fmt.Errorf("can't obtain a chunk of a directory")
    }

    size := fi.Size()

    lastChunkIndex := size / CHUNK_MAX_SIZE - 1
    if size % CHUNK_MAX_SIZE != 0 {
        lastChunkIndex++
    }
    if lastChunkIndex == -1 {
        lastChunkIndex = 0
    }

    if chunkIndex > lastChunkIndex {
        return nil, nil, fmt.Errorf("chunkIndex %d out of bounds", chunkIndex)
    }

    f, err := os.Open(path)
    if err != nil {
        return nil, nil, err
    }
    defer f.Close()

    var buf []byte
    if chunkIndex == lastChunkIndex {
        buf = make([]byte, size % CHUNK_MAX_SIZE)
    } else {
        buf = make([]byte, CHUNK_MAX_SIZE)
    }

    _, err = f.Seek(chunkIndex * CHUNK_MAX_SIZE, 0)
    if err != nil {
        return nil, nil, err
    }
    
    bytesRead, err := f.Read(buf)
    if err != nil {
        return nil, nil, err
    } else if bytesRead != len(buf) {
        return nil, nil, fmt.Errorf("wrong size read")
    }

    return getHashOfChunk(buf), buf, nil
}

func getHashOfChunk(chunk []byte) []byte {
	hasher := sha256.New()
	hasher.Write(chunk)
	return hasher.Sum(nil)
}

/*func createChunkMsg(id uint32) udpMsg {
    body := []byte{CHUNK, 65, 66, 67}
    body = append(getHashOfChunk(body),body...)
    return  createMsgWithId(id,DATUM,body)
}

func createRootDatum(id uint32) udpMsg {
    body := []byte{DIRECTORY}
    body = append(body,[]byte("READMEAS891111111111111111111111")...)
    body = append(getHashOfChunk(body),body...)
    res := createMsgWithId(id,DATUM,body)
    return res
}

func createRoot(id uint32) udpMsg {
    body := getHashOfChunk(createRootDatum(id).Body[32:])
    return createMsgWithId(id,ROOT_REPLY,body)
}*/
