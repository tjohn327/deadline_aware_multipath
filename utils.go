package main

import (
	"container/list"
	"encoding/binary"
)

type FragUnit struct {
	index uint16
	frag  uint16
	buf   []byte
}

func IsInInt(val int, ls []int) bool {
	for _, v := range ls {
		if val == v {
			return true
		}
	}
	return false
}

func RemoveFrag(ls list.List, ack []byte) {
	index := binary.BigEndian.Uint16(ack[:2])
	frag := binary.BigEndian.Uint16(ack[2:4])
	for temp := ls.Front(); temp != nil; temp = temp.Next() {
		if v, ok := temp.Value.(FragUnit); ok {
			if v.index == index && v.frag == frag {
				ls.Remove(temp)
				return
			}
		}
	}
}
