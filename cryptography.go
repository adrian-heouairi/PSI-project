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
// https://stackoverflow.com/a/41315404
func encode(privateKey *ecdsa.PrivateKey, publicKey *ecdsa.PublicKey) (string, string) {
    x509Encoded, _ := x509.MarshalECPrivateKey(privateKey)
    pemEncoded := pem.EncodeToMemory(&pem.Block{Type: "PRIVATE KEY", Bytes: x509Encoded})

    x509EncodedPub, _ := x509.MarshalPKIXPublicKey(publicKey)
    pemEncodedPub := pem.EncodeToMemory(&pem.Block{Type: "PUBLIC KEY", Bytes: x509EncodedPub})

    return string(pemEncoded), string(pemEncodedPub)
}

// https://stackoverflow.com/a/41315404
func decode(pemEncoded string, pemEncodedPub string) (*ecdsa.PrivateKey, *ecdsa.PublicKey) {
    block, _ := pem.Decode([]byte(pemEncoded))
    x509Encoded := block.Bytes
    privateKey, _ := x509.ParseECPrivateKey(x509Encoded)

    blockPub, _ := pem.Decode([]byte(pemEncodedPub))
    x509EncodedPub := blockPub.Bytes
    genericPublicKey, _ := x509.ParsePKIXPublicKey(x509EncodedPub)
    publicKey := genericPublicKey.(*ecdsa.PublicKey)

    return privateKey, publicKey
}

func storeKeysInFile(pemEncoded string, pemEncodedPub string) {
    finfo, err := os.Open(PUB_KEY_FILE_PATH)
    if err != nil {
       fmt.Fprint(os.Stderr, "Could not store public key") 
    }
    finfo1, err := os.Open(KEY_FILE_PATH)
    if err != nil {
       fmt.Fprint(os.Stderr, "Could not store private key") 
    }
    finfo.WriteString(pemEncodedPub)
    finfo1.WriteString(pemEncodedPub)
}
