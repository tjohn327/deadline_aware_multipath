package main

import (
	"crypto/md5"
	"io/ioutil"
	"testing"
	"time"
)

func TestDataQueue(t *testing.T) {
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

	receiveChan := make(chan *DataFragment, 2)
	ackChan := make(chan *DataFragment, 2)
	retransmitChan := make(chan *DataBlock, 2)
	egressChan := make(chan *DataBlock, 2)

	deadline := time.Duration(time.Second)
	unackQueue := NewDataQueue(UnAck, ackChan, retransmitChan, &deadline, nil)
	NewDataQueue(Receive, receiveChan, egressChan, nil, nil) //receive queue
	unackQueue.InsertBlock(db)

	for i, v := range db.fragments {
		receiveChan <- v
		ackChan <- v
		if i == 185 {
			break
		}
	}

	db_ret := <-retransmitChan
	for _, v := range db_ret.fragments {
		if !v.acked {
			receiveChan <- v
		}
	}

	db_out := <-egressChan

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
