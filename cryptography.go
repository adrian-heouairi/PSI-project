package main

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"fmt"
	"os"
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
    publicKey, ok := privateKey.Public().(*ecdsa.PublicKey)
    if !ok {
        fmt.Fprintf(os.Stderr, "Could not generate private key, stopping...")
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
