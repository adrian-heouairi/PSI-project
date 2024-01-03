package main

import (
	"fmt"
	"os"
)

// TODO Organize order of functions/methods

type merkleTreeNode struct {
	// Depends on Type:
	//  - CHUNK: ChunkIndex starting at 0
	//  - BIG_FILE: 2 <= len(ChildrenNodes) <= 32
	//  - DIRECTORY: 0 <= len(ChildrenNodes) <= 16

	// The parent node, useful for hash computation
	// Never nil except for the root node
	Parent *merkleTreeNode

	// Path of the file or directory this node represents
	Path string

	// Never nil (even for CHUNK)
	Children []*merkleTreeNode

	// Nil if not computed yet
	Hash []byte

	// CHUNK, TREE, DIRECTORY
	Type byte

	// nil if not CHUNK
	ChunkIndex int
}

var ourTree *merkleTreeNode
// Maps a hash as string(h), with h being the hash in []byte to a pointer of the merkleTreeNode that represents it
var ourTreeMap map[string]*merkleTreeNode

func (node *merkleTreeNode) basename() string {
	return replaceAllRegexBy(node.Path, ".*/", "")
}

// First call is supposed to be done on the directory representing the root, its Parent will be nil
// Computes hashes for all leaf nodes (DIRECTORY or CHUNK)
func recursivePathToMerkleTreeWithoutInternalHashes(path string, parent *merkleTreeNode) (*merkleTreeNode, error) {
	fileInfo, err := os.Stat(path)
	if err != nil {
		return nil, err
	}

	ret := &merkleTreeNode{Children: []*merkleTreeNode{}, Parent: parent, Path: path, ChunkIndex: -1}

	if fileInfo.IsDir() {
		ret.Type = DIRECTORY

		entries, err := os.ReadDir(path)
		if err != nil {
			return nil, err
		}

		if len(entries) == 0 {
			ret.Hash = getHashOfByteSlice([]byte{DIRECTORY})
		}

		for _, entry := range entries {
			recursiveCall, err := recursivePathToMerkleTreeWithoutInternalHashes(path+"/"+entry.Name(), ret)
			if err != nil {
				return nil, err
			}
			ret.Children = append(ret.Children, recursiveCall)
		}
	} else {
		if fileInfo.Size() <= CHUNK_MAX_SIZE {
			ret.Type = CHUNK
			ret.ChunkIndex = 0

			chunkWithoutType, _ := getChunkContents(path, 0)
			ret.Hash = getChunkHash(chunkWithoutType)
		} else {
			// TODO BIG_FILE
			ret.Type = TREE
			return nil, fmt.Errorf("BIG_FILE not implemented")
		}
	}

	return ret, nil
}

// Returns the content of the chunk at index chunkIndex in the file at path (without CHUNK type byte as first byte)
func getChunkContents(path string, chunkIndex int64) ([]byte, error) {
	fi, err := os.Stat(path)
	if err != nil {
		return nil, err
	}

	if fi.IsDir() {
		return nil, fmt.Errorf("can't obtain a chunk of a directory")
	}

	size := fi.Size()

	lastChunkIndex := size/CHUNK_MAX_SIZE - 1
	if size%CHUNK_MAX_SIZE != 0 {
		lastChunkIndex++
	}
	if lastChunkIndex == -1 {
		lastChunkIndex = 0
	}

	if chunkIndex > lastChunkIndex {
		return nil, fmt.Errorf("chunkIndex %d out of bounds", chunkIndex)
	}

	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	var buf []byte
	if chunkIndex == lastChunkIndex {
		buf = make([]byte, size%CHUNK_MAX_SIZE)
	} else {
		buf = make([]byte, CHUNK_MAX_SIZE)
	}

	_, err = f.Seek(chunkIndex*CHUNK_MAX_SIZE, 0)
	if err != nil {
		return nil, err
	}

	bytesRead, err := f.Read(buf)
	if err != nil {
		return nil, err
	} else if bytesRead != len(buf) {
		return nil, fmt.Errorf("wrong size read")
	}

	return buf, nil
}

func getChunkHash(chunkWithoutType []byte) []byte {
	chunkWithType := []byte{CHUNK}
	chunkWithType = append(chunkWithType, chunkWithoutType...)
	return getHashOfByteSlice(chunkWithType)
}

func (node *merkleTreeNode) toString() string {
	res := ""
	typeStr, _ := byteToDatumTypeAsStr(node.Type)
	res += fmt.Sprintf("Parent == nil: %v\nHash: %s\nType: %s\nPath: %s", node.Parent == nil, fmt.Sprint(node.Hash), typeStr, node.Path)
	if node.Type != CHUNK {
		res += fmt.Sprintf("\nNb of children: %d", len(node.Children))
	} else {
		res += fmt.Sprintf("\nChunk %d", node.ChunkIndex)
	}

	return res
}

func (node *merkleTreeNode) printMerkleTreeRecursively() {
	fmt.Println(node.toString())
	fmt.Println()

	for _, child := range node.Children {
		child.printMerkleTreeRecursively()
	}
}

func (node *merkleTreeNode) computeHashesRecursively() {
	if node.Hash != nil {
		return
	}

	// We don't have a hash, so we are not a leaf, thus we have children
	value := []byte{node.Type}
	if node.Type == DIRECTORY {
		for _, child := range node.Children {
			child.computeHashesRecursively()
			value = append(value, stringToZeroPaddedByteSlice(child.basename())...)
			value = append(value, child.Hash...)
		}
		node.Hash = getHashOfByteSlice(value)
	} else {
		panic("Big file not supported")
	}
}

func exportMerkleTree() error {
	var err error
	ourTree, err = recursivePathToMerkleTreeWithoutInternalHashes(SHARED_FILES_DIR, nil)
	if err != nil {
		return err
	}
	ourTree.computeHashesRecursively()

    ourTreeMap = ourTree.toMap()
	return nil
}

func (node *merkleTreeNode) toDatum(id uint32) (udpMsg, error) {
	body := node.Hash
	body = append(body, node.Type)
	switch node.Type {
	case CHUNK:
		chunk, err := getChunkContents(node.Path, int64(node.ChunkIndex))
		if err != nil {
			return udpMsg{}, err
		}
		body = append(body, chunk...)
	case DIRECTORY:
		for _, child := range node.Children {
			body = append(body, stringToZeroPaddedByteSlice(child.basename())...)
			body = append(body, child.Hash...)
		}
	case TREE:
		panic("big file not implemented yet")
	}
	return createMsgWithId(id, DATUM, body), nil
}

func (node *merkleTreeNode) toMapRecursively(currentMap map[string]*merkleTreeNode) {
	currentMap[string(node.Hash)] = node
	for _, child := range node.Children {
		child.toMapRecursively(currentMap)
	}
}

func (node *merkleTreeNode) toMap() map[string]*merkleTreeNode {
	currentMap := make(map[string]*merkleTreeNode)
	node.toMapRecursively(currentMap)
	return currentMap
}
