package main

import (
	"crypto/md5"
	"io/ioutil"
	"testing"
)

func TestDataBlock(t *testing.T) {
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

	db := NewDataBlockFromEncodedData(encodedData, 1)

	encodedDataOut, err := db.GetEncodedData()
	if err != nil {
		t.Error("error retreiving encoded data:", err)
		return
	}
	out, err := reedSolomon.Decode(encodedDataOut)
	if err != nil {
		t.Error("error decoding:", err)
		return
	}
	outHash := md5.Sum(out)
	if inHash != outHash {
		t.Error("decoding failed, hash mismatch")
	}

}

func TestDataBlockFrags(t *testing.T) {
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

	ed, err := reedSolomon.Encode(splitData, parityCount)
	if err != nil {
		t.Error("error encoding:", err)
		return
	}

	db := NewDataBlockFromEncodedData(ed, 1)

	frag1, err := NewFragmentFromBytes(db.fragments[0].data)
	if err != nil {
		t.Error("error creating fragment from bytes:", err)
		return
	}
	db_out := NewDataBlockFromFragment(frag1)
	for i := 2; i < frag1.fragmentCount-parityCount+1; i++ {
		frag, err := NewFragmentFromBytes(db.fragments[i].data)
		if err != nil {
			t.Error("error creating fragment from bytes:", err)
			return
		}
		_, err = db_out.InsertFragment(frag)
		if err != nil {
			t.Error("error inserting fragment into datablock:", err)
			return
		}
	}

	encodedDataOut, err := db_out.GetEncodedData()
	if err != nil {
		t.Error("error retreiving encoded data:", err)
		return
	}
	out, err := reedSolomon.Decode(encodedDataOut)
	if err != nil {
		t.Error("error decoding:", err)
		return
	}
	outHash := md5.Sum(out)
	if inHash != outHash {
		t.Error("decoding failed, hash mismatch")
	}

}
