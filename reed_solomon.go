package main

import "github.com/klauspost/reedsolomon"

type ReedSolomon struct {
	fragmentCount int
	parityCount   int
	encoder       reedsolomon.Encoder
}
