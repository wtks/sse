package sse

import "crypto/rand"

type ID [16]byte

func generateRandomID() (id ID) {
	if _, err := rand.Read(id[:]); err != nil {
		panic(err)
	}
	return
}
