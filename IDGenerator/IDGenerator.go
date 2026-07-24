package idgenerator

import (
	"crypto/rand"
	"math/big"
)

func GenerateID(length int) string {
	const charset = "abcdefghijklmnopqrstuvwxyz"

	b := make([]byte, length)
	for i := range b {
		n, err := rand.Int(rand.Reader, big.NewInt(int64(len(charset))))
		if err != nil {
			panic(err)
		}
		b[i] = charset[n.Int64()]
	}
	return string(b)
}
