package platform

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
)

type IDGenerator interface {
	New(prefix string) string
}

type CryptoIDGenerator struct{}

func (CryptoIDGenerator) New(prefix string) string {
	var buffer [10]byte
	if _, err := rand.Read(buffer[:]); err != nil {
		panic(fmt.Sprintf("secure random source failed: %v", err))
	}
	return prefix + "_" + hex.EncodeToString(buffer[:])
}
