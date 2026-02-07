package security

import (
	"crypto/sha256"
	"encoding/hex"
)

func HashRefreshToken(raw, pepper string) string {
	h := sha256.Sum256([]byte(raw + ":" + pepper))
	return hex.EncodeToString(h[:])
}
