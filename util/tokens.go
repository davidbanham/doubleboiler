package util

import (
	"crypto/sha256"
	"doubleboiler/config"
	"encoding/base64"
	"time"
)

func CalcExpiry(days int) string {
	return time.Now().AddDate(0, 0, days).UTC().Format(time.RFC3339)
}

func CalcToken(input, expiry string) (token string) {
	plaintext := config.SECRET + input + expiry
	hash := sha256.New()
	encHash := hash.Sum([]byte(plaintext))
	token = base64.StdEncoding.EncodeToString(encHash)
	return
}
