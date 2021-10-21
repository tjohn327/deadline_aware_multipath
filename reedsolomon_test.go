package main

import (
	"crypto/md5"
	"io/ioutil"
	"math/rand"
	"testing"
)

func TestEncodeDecode(t *testing.T) {
	in, err := ioutil.ReadFile("testdata/test_image.jpg")
	if err != nil {
		t.Error("error reading test image")
	}
	inHash := md5.Sum(in)
	splitData, err := Split(in, 999)
	if err != nil {
		t.Error("error splitting data")
	}
	parityCount := 3
	reedSolomon, err := NewReedSolomon(splitData.fragmentCount, parityCount)
	if err != nil {
		t.Error("error creating encoder:", err)
		return
	}

	encodedData, err := reedSolomon.Encode(splitData, parityCount)
	if err != nil {
		t.Error("error encoding:", err)
		return
	}
	l := len(encodedData.data)
	for i := 0; i < parityCount; i++ {
		encodedData.data[rand.Intn(l)] = nil
	}

	out, err := reedSolomon.Decode(encodedData)
	if err != nil {
		t.Error("error decoding:", err)
		return
	}
	outHash := md5.Sum(out)
	if inHash != outHash {
		t.Error("decoding failed, hash mismatch")
	}

}
