package main

import (
	"crypto/sha256"
)

func castNameFromBytesSliceToString(name []byte) string {
	i := 0
	for name[i] != 0 {
		i++
	}
	return string(name[:i])
}

// Returns a map containing the names and the hashes of the Datum message
// Assumes that msg is datum of type Directory
// Otherwise empty map is returned
func getDataHashes(msg udpMsg) map[string][]byte {
	res := make(map[string][]byte)
	if msg.Body[DATUM_TYPE_INDEX] == DIRECTORY {
		nbEntry := (msg.Length - 33) / 64
		for i := 0; i < int(nbEntry); i++ {
			res[castNameFromBytesSliceToString(msg.Body[33+i*64:33+i*64+32])] = msg.Body[65+i*64 : 65+i*64+32]

		}
	}
	return res
}

func check_data_integrity(hash []byte, content []byte) bool {
	computed_hash := sha256.New()
	computed_hash.Write(content)
	hash_sum := computed_hash.Sum(nil)
	if len(hash_sum) != len(hash) {

		return false
	}
	for i := 0; i < len(hash); i++ {
		if hash_sum[i] != hash[i] {
			return false
		}
	}
	return true
}
