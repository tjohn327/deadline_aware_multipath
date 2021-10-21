package main

import (
	"crypto/md5"
	"io/ioutil"
	"testing"
)

func TestSplit(t *testing.T) {
	in, err := ioutil.ReadFile("testdata/test_image.jpg")
	if err != nil {
		t.Error("error reading test image")
	}
	inHash := md5.Sum(in)
	s, err := Split(in, 999)
	if err != nil {
		t.Error("error splitting data: ", err)
	}
	if s.fragmentCount != 191 {
		t.Errorf("wrong number of fragments, expected %d got %d", 191, s.fragmentCount)
	}

	if s.padlen != 659 {
		t.Errorf("wrong pad length, expected %d got %d", 659, s.padlen)
	}
	out, err := s.Join()
	if err != nil {
		t.Error("error joining data")
	}
	outHash := md5.Sum(out)

	if inHash != outHash {
		t.Error("joining failed, hash mismatch")
	}

}
