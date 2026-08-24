package domain

import (
	"crypto/rand"
	"encoding/hex"
	"strings"
)

func NewID(prefix string) string {
	b := make([]byte, 10)
	if _, err := rand.Read(b); err != nil {
		panic(err)
	}
	return prefix + "_" + hex.EncodeToString(b)
}

func NormalizeText(value string) string {
	return strings.Join(strings.Fields(strings.TrimSpace(value)), " ")
}

func ValidIdempotencyKey(key string) bool {
	key = strings.TrimSpace(key)
	return len(key) >= 8 && len(key) <= 128
}
