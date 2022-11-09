package main

import "fmt"

const (
	MAX_FRAG_SIZE = 1200
)

type SplitData struct {
	fragSize      int
	padlen        int
	fragmentCount int
	data          [][]byte
}

func SplitDataFromByteArray(data [][]byte) (*SplitData, error) {
	fragmentCount := len(data)
	fragSize := len(data[0])
	padlen := 0
	s := &SplitData{
		fragSize:      fragSize,
		padlen:        padlen,
		fragmentCount: fragmentCount,
		data:          data,
	}
	return s, nil
}

func Split(data []byte, fragSize int) (*SplitData, error) {
	if fragSize > MAX_FRAG_SIZE {
		err := fmt.Errorf("fragment size greater than maximum allowable fragment size")
		return nil, err
	}
	blocklen := len(data)
	nFrags := blocklen / fragSize
	padlen := 0
	if blocklen%nFrags != 0 {
		nFrags++
	}
	block := make([][]byte, nFrags)
	j := 0
	for i := 0; i < nFrags; i++ {
		k := j + fragSize
		if (k) <= blocklen {
			block[i] = data[j:k]
		} else {
			block[i] = data[j:]
			padlen = fragSize - len(block[i])
			block[i] = append(block[i], createPadding(padlen)...)
		}
		j = k
	}

	splitData := &SplitData{
		fragSize:      fragSize,
		padlen:        padlen,
		fragmentCount: nFrags,
		data:          block,
	}
	return splitData, nil
}

func createPadding(padlen int) []byte {
	pad := make([]byte, padlen)
	for i := range pad {
		pad[i] = 0x00
	}
	return pad
}

func (s *SplitData) Print() {
	fmt.Println(s.fragmentCount, s.fragSize, s.padlen, len(s.data))
}

func (s *SplitData) Join() ([]byte, error) {
	size := (s.fragSize * s.fragmentCount) - s.padlen
	out := make([]byte, 0)
	for i := range s.data {
		if i < s.fragmentCount-1 {
			out = append(out, s.data[i]...)
		} else {
			out = append(out, s.data[i][:s.fragSize-s.padlen]...)
		}
	}
	if size != len(out) {
		err := fmt.Errorf("error joining the split data")
		return nil, err
	}
	return out, nil
}
