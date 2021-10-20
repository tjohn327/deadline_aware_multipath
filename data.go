package main

import (
	"encoding/binary"
	"fmt"
	"sort"
	"sync"
)

type DataBlock struct {
	blockID   uint16
	len       int
	complete  bool
	mutex     sync.Mutex
	fragments []*DataFragment
}

func NewFrame(blockID uint16) *DataBlock {
	return &DataBlock{
		blockID:   blockID,
		len:       0,
		complete:  false,
		fragments: make([]*DataFragment, 0),
	}
}

func (v *DataBlock) InsertFragment(fragment *DataFragment) (bool, error) {
	if fragment.blockID != v.blockID {
		return false, fmt.Errorf("frameID mismatch")
	}
	_, in := v.isInFrame(fragment.fragmentID)
	if !in {
		v.mutex.Lock()
		defer v.mutex.Unlock()
		v.fragments = append(v.fragments, fragment)
		v.len++
		// fmt.Println("Frame frag IN", fragment.frameID, fragment.fragmentID)
		v.sortFragments()
		f := v.fragments
		if f[len(f)-1].fragmentID == 1 && int(f[0].fragmentID) == v.len {
			v.complete = true
			// fmt.Println("Frame Complete", v.len, v.frameID)
			return true, nil
		}
	}
	return false, nil
}

func (vf *DataBlock) removeFragmentByID(fragID uint16) bool {
	for i, v := range vf.fragments {
		if v.fragmentID == fragID {
			vf.fragments = append(vf.fragments[:i], vf.fragments[i+1:]...)
			vf.len--
			return true
		}
	}
	return false
}

func (vf *DataBlock) sortFragments() {
	sort.Slice(vf.fragments, func(i, j int) bool {
		return vf.fragments[i].fragmentID > vf.fragments[j].fragmentID
	})
}

func (vf *DataBlock) isInFrame(fragmentID uint16) (int, bool) {
	vf.mutex.Lock()
	defer vf.mutex.Unlock()
	for i, v := range vf.fragments {
		if v.fragmentID == fragmentID {
			return i, true
		}
	}
	return 0, false
}

type DataFragment struct {
	blockID    uint16
	fragmentID uint16
	data       []byte
}

func getFrameFragIDs(data []byte) (uint16, uint16, error) {
	if len(data) < 4 {
		return 0, 0, fmt.Errorf("invalid error fragment")
	}
	frameID := binary.BigEndian.Uint16(data[:2])
	fragID := binary.BigEndian.Uint16(data[2:4])
	return frameID, fragID, nil
}

func NewFragment(data []byte) (*DataFragment, error) {
	blockID, fragID, err := getFrameFragIDs(data)
	if err != nil {
		return nil, err
	}
	return &DataFragment{
		blockID:    blockID,
		fragmentID: fragID,
		data:       data,
	}, nil
}
