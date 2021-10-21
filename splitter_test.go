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
		t.Error("error splitting data")
	}
	if s.nFragment != 776 {
		t.Errorf("wrong number of fragments, expected %d got %d", 776, s.nFragment)
	}

	if s.padlen != 592 {
		t.Errorf("wrong pad length, expected %d got %d", 592, s.padlen)
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
