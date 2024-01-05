package main

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
    "crypto/sha256"
    "math/big"
	"fmt"
	"os"
	"encoding/binary"
)

var privateKey *ecdsa.PrivateKey = nil
func getPrivateKey() *ecdsa.PrivateKey {
    var err error
    if privateKey == nil {

        privateKey, err = ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
        if err != nil {
            fmt.Fprintf(os.Stderr, "Could not generate private key, stopping...")
            os.Exit(1)
        }
    }
    return privateKey
}

func getPublicKey() *ecdsa.PublicKey {
    privateKey = getPrivateKey()
    publicKey, ok := privateKey.Public().(*ecdsa.PublicKey)
    if !ok {
        fmt.Fprintf(os.Stderr, "Could not get public key, stopping...")
        os.Exit(1)
    }
    return publicKey
}

func publicKeyToHexaString(publicKey *ecdsa.PublicKey) []byte {
    formatted := make([]byte, 64)
    publicKey.X.FillBytes(formatted[:32])
    publicKey.Y.FillBytes(formatted[32:])
    return formatted
}

func getPublicKeyAsHexaString() []byte {
    return publicKeyToHexaString(getPublicKey())
}

func parsePublickey(key []byte) *ecdsa.PublicKey {
    fmt.Println("len of parsing key:", len(key))
    var x, y big.Int
    x.SetBytes(key[:32])
    y.SetBytes(key[32:])
    parsedPublicKey := ecdsa.PublicKey{
        Curve: elliptic.P256(),
        X: &x,
        Y: &y,
    }
    return &parsedPublicKey
}

func signMsg(msg udpMsg) udpMsg {
    toHash := []byte{}
	var idToByteSlice []byte = make([]byte, 4)
	binary.BigEndian.PutUint32(idToByteSlice, msg.Id)
	var typeToByteSlice []byte = make([]byte, 1)
	typeToByteSlice[0] = msg.Type
	var lengthToByteSlice []byte = make([]byte, 2)
	binary.BigEndian.PutUint16(lengthToByteSlice, msg.Length)
    toHash = append(toHash, idToByteSlice...)
    toHash = append(toHash, typeToByteSlice...)
    toHash = append(toHash, lengthToByteSlice...)
    toHash = append(toHash, msg.Body...)
    hashed := sha256.Sum256(toHash)
    r, s, err := ecdsa.Sign(rand.Reader, privateKey, hashed[:])
    if err != nil {
       fmt.Fprint(os.Stderr, "Could not sign msg") 
       os.Exit(0)
       return udpMsg{}
    }
    signature := make([]byte, SIGNATURE_SIZE)
    r.FillBytes(signature[:32])
    s.FillBytes(signature[32:])
    msg.Signature = append(msg.Signature, signature...)
    return msg
}

func checkMsgSignature(msg udpMsg, publicKey []byte) bool {
    if msg.Signature == nil {
       return true
    }

    toHash := []byte{}
	var idToByteSlice []byte = make([]byte, 4)
	binary.BigEndian.PutUint32(idToByteSlice, msg.Id)
	var typeToByteSlice []byte = make([]byte, 1)
	typeToByteSlice[0] = msg.Type
	var lengthToByteSlice []byte = make([]byte, 2)
	binary.BigEndian.PutUint16(lengthToByteSlice, msg.Length)
    toHash = append(toHash, idToByteSlice...)
    toHash = append(toHash, typeToByteSlice...)
    toHash = append(toHash, lengthToByteSlice...)
    toHash = append(toHash, msg.Body...)
    hashed := sha256.Sum256(toHash)
    fmt.Println("len hashed",len(hashed))
    var r, s big.Int
    r.SetBytes(msg.Signature[:32])
    s.SetBytes(msg.Signature[32:])
    fmt.Println("sginature that we want to check :",msg.Signature)
    return ecdsa.Verify(parsePublickey(publicKey), hashed[:], &r, &s)
}
