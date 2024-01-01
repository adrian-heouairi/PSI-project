package main
import (
    "crypto/sha256"
    "os"
)


type chunk struct {
    Path string
    Index int
}

type MerkleTree struct {
    // Depends on Type:
    //  - CHUNK : [path, idx]
    //  - TREE : 2 <= len(ChildrenNodes) <= 32
    //  - CUNK : 0 <= len(ChildrenNodes) <= 16
    ChildrenNodes []*MerkleTree
    Hash []byte
    Type byte // CHUNK, TREE, DIRECTORY
    Content *chunk
    DirectoryChildrenNames []string // TODO fct which pads with trailing zeros utill 32 bytes
}

var OUR_TREE MerkleTree

func pathToMerkleTree(path string) MerkleTree {
    // TODO STAT FOR PATH
    fileIno, _ := os.Stat(path)
    if fileIno.IsDir() {
        entries, err := os.ReadDir(path)
        if err != nil {
            panic("TODO ADIRAN PLESASE IMPLEMENT ERROR CHECKCING")
        }
        chidrenNames := make([]string, len(entries))
        for _, entry:= range entries {
            chidrenNames = append(chidrenNames, entry.Name()) 
        }
    } else {
        if fileIno.Size() <= CHUNK_MAX_SIZE {
            hash, _ := chunkFile(path,0) 
            return MerkleTree{
                Hash: hash,
                Type: CHUNK,
                Content: &chunk{path, 0},
            }
        }
    }
    return MerkleTree{}

}

// Returns the conent of the asked chunk
// If file size is less than CHUNK_SIZE first chunk is returned
// If chunk is grater than the number of chunks present in the file the last chunk is returned
func chunkFile(path string, chunk int64)([]byte, [] byte){
    fi, err := os.Stat(path)
    checkErr(err)
    size := fi.Size()    
    f, err := os.Open(path)
    checkErr(err)
    defer f.Close()
    b1 := make([]byte, CHUNK_MAX_SIZE)
    if size < CHUNK_MAX_SIZE {
        b1 = make([]byte,size)
    }
    _,err = f.Seek(chunk * CHUNK_MAX_SIZE,0)
    checkErr(err)
    if chunk > size / CHUNK_MAX_SIZE {
        newChunk := size / CHUNK_MAX_SIZE - 1
        if chunk % CHUNK_MAX_SIZE != 0 {
           newChunk++
        }
        _,err = f.Seek(newChunk * CHUNK_MAX_SIZE, 0)
        checkErr(err)
    }
    _, err = f.Read(b1)
    checkErr(err)
    return getHashOfChunk(b1),b1
}

func getHashOfChunk(chunk []byte) []byte {
	hasher := sha256.New()
	hasher.Write(chunk)
	computedHash := hasher.Sum(nil) 
    return computedHash
}

func createChunkMsg(id uint32) udpMsg {
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
}
