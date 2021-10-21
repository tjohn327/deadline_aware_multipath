package main

import (
	"github.com/klauspost/reedsolomon"
)

type ReedSolomon struct {
	dataCount   int
	parityCount int
	encoder     reedsolomon.Encoder
}

type EncodedData struct {
	dataCount   int
	parityCount int
	fragSize    int
	padlen      int
	data        [][]byte
}

func NewReedSolomon(dataCount int, parityCount int) (*ReedSolomon, error) {
	enc, err := reedsolomon.New(dataCount, parityCount)
	if err != nil {
		return nil, err
	}
	r := &ReedSolomon{
		dataCount:   dataCount,
		parityCount: parityCount,
		encoder:     enc,
	}
	return r, nil
}

func (r *ReedSolomon) Encode(sd *SplitData, parityCount int) (*EncodedData, error) {
	if sd.nFragment != r.dataCount || r.parityCount != parityCount {
		enc, err := reedsolomon.New(sd.nFragment, parityCount)
		if err != nil {
			return nil, err
		}
		r.encoder = enc
		r.dataCount = sd.nFragment
		r.parityCount = parityCount
	}
	data := sd.data
	parityFrags := make([][]byte, parityCount)
	for i := range parityFrags {
		parityFrags[i] = make([]byte, sd.fragSize)
	}
	data = append(data, parityFrags...)
	err := r.encoder.Encode(data)
	if err != nil {
		return nil, err
	}
	return &EncodedData{
		dataCount:   sd.nFragment,
		parityCount: parityCount,
		fragSize:    sd.fragSize,
		padlen:      sd.padlen,
		data:        data,
	}, nil
}

func (r *ReedSolomon) Decode(ed *EncodedData) ([]byte, error) {
	if r.dataCount != ed.dataCount || r.parityCount != ed.parityCount {
		enc, err := reedsolomon.New(ed.dataCount, ed.parityCount)
		if err != nil {
			return nil, err
		}
		r.encoder = enc
		r.dataCount = ed.dataCount
		r.parityCount = ed.parityCount
	}
	err := r.encoder.Reconstruct(ed.data)
	if err != nil {
		return nil, err
	}
	splitData := &SplitData{
		fragSize:  ed.fragSize,
		padlen:    ed.padlen,
		nFragment: ed.dataCount,
		data:      ed.data[:ed.dataCount],
	}
	out, err := splitData.Join()
	if err != nil {
		return nil, err
	}
	return out, nil
}
