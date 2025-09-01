// Package utils provides helper functions for hashing and related utilities.
package utils

import (
	"crypto/sha256"
	"encoding/hex"
)

// CalculateHash returns a SHA256 hash of the request body combined with the secret key.
func CalculateHash(body []byte, key string) string {
	h := sha256.New()
	h.Write(body)
	h.Write([]byte(key))
	return hex.EncodeToString(h.Sum(nil))
}
