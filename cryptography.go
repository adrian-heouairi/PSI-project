package main

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/sha256"
	"crypto/x509"
	"encoding/binary"
	"encoding/pem"
	"fmt"
	"math/big"
	"os"
)

var privateKey *ecdsa.PrivateKey = nil
var publicKey *ecdsa.PublicKey = nil

func setKeys() {
	privateFile, errPrivate := os.Open(PRIVATE_KEY_PATH)
	if errPrivate != nil { // Private key file doesn't exist
		f, _ := os.Create(PRIVATE_KEY_PATH)
		privateKey, _ = ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
		privateEncoded := encode(privateKey)
		f.WriteString(privateEncoded)
	} else { // Private key file exists
		privateStat, _ := os.Stat(PRIVATE_KEY_PATH)
		encodedPrivateKeyAsBytesSlice := make([]byte, privateStat.Size())
		privateFile.Read(encodedPrivateKeyAsBytesSlice)
		privateKey = decode(string(encodedPrivateKeyAsBytesSlice))
	}
	publicKey = &privateKey.PublicKey
}

func publicKeyToHexaString() []byte {
	formatted := make([]byte, 64)
	publicKey.X.FillBytes(formatted[:32])
	publicKey.Y.FillBytes(formatted[32:])
	return formatted
}

func parsePublicKey(key []byte) *ecdsa.PublicKey {
	fmt.Println("len of parsing key:", len(key))
	var x, y big.Int
	x.SetBytes(key[:32])
	y.SetBytes(key[32:])
	parsedPublicKey := ecdsa.PublicKey{
		Curve: elliptic.P256(),
		X:     &x,
		Y:     &y,
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

func checkMsgSignature(msg udpMsg, peerPublicKey []byte) bool {
	if msg.Signature == nil { // Peer doesn't implement cryptography
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
	var r, s big.Int
	r.SetBytes(msg.Signature[:32])
	s.SetBytes(msg.Signature[32:])
	LOGGING_FUNC("signature that we want to check :", msg.Signature)
	ok := ecdsa.Verify(parsePublicKey(peerPublicKey), hashed[:], &r, &s)
	return ok
}

// https://stackoverflow.com/a/41315404
func encode(privateKey *ecdsa.PrivateKey) string { // publicKey *ecdsa.PublicKey
	x509Encoded, _ := x509.MarshalECPrivateKey(privateKey)
	pemEncoded := pem.EncodeToMemory(&pem.Block{Type: "PRIVATE KEY", Bytes: x509Encoded})

	//x509EncodedPub, _ := x509.MarshalPKIXPublicKey(publicKey)
	//pemEncodedPub := pem.EncodeToMemory(&pem.Block{Type: "PUBLIC KEY", Bytes: x509EncodedPub})

	return string(pemEncoded) // , string(pemEncodedPub)
}

// https://stackoverflow.com/a/41315404
func decode(pemEncoded string) *ecdsa.PrivateKey { // pemEncodedPub string  *ecdsa.PublicKey
	block, _ := pem.Decode([]byte(pemEncoded))
	x509Encoded := block.Bytes
	privateKey, _ := x509.ParseECPrivateKey(x509Encoded)

	/*blockPub, _ := pem.Decode([]byte(pemEncodedPub))
	x509EncodedPub := blockPub.Bytes
	genericPublicKey, _ := x509.ParsePKIXPublicKey(x509EncodedPub)
	publicKey := genericPublicKey.(*ecdsa.PublicKey)*/

	return privateKey //, publicKey
}
