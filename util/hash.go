package util

import (
	"crypto/sha256"
	"encoding/hex"
)

// Hash hash SHA256
func Hash(s string) string {
	h := sha256.New()
	h.Write([]byte(s))
	return hex.EncodeToString(h.Sum(nil))
}
