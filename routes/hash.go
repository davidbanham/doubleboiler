package routes

import (
	"crypto/md5"
	"encoding/hex"
)

func calcHash(str string) string {
	hash := md5.Sum([]byte(str))
	return hex.EncodeToString(hash[:])
}
