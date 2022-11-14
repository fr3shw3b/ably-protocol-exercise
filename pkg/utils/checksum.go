package utils

import (
	"crypto/sha1"
	"encoding/hex"
	"encoding/json"
)

func CreateChecksum(sequence []uint32) string {
	hasher := sha1.New()
	// todo: improve with explicit error handling.
	bytes, _ := json.Marshal(sequence)
	hasher.Write(bytes)
	return hex.EncodeToString(hasher.Sum(nil))
}
