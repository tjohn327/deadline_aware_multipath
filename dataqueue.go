package main

import (
	"fmt"
	"sync"
)

type DataQueue struct {
	headBlockID int
	tailBlockID int
	len         int
	mutex       sync.Mutex
	blocks      []*DataBlock
	ingressChan chan *DataBlock
	egressChan  chan *DataBlock
	ackChan     chan *DataFragment
	isRemote    bool
}

func NewDataQueue(ingressChan chan *DataBlock, egressChan chan *DataBlock,
	ackChan chan *DataFragment, isRemote bool) *DataQueue {
	if ingressChan == nil {
		ingressChan = make(chan *DataBlock, 10)
	}
	if egressChan == nil {
		egressChan = make(chan *DataBlock, 10)
	}
	if ackChan == nil {
		ackChan = make(chan *DataFragment, 10)
	}
	return &DataQueue{
		headBlockID: 0,
		tailBlockID: 0,
		len:         0,
		blocks:      make([]*DataBlock, 0),
		ingressChan: ingressChan,
		egressChan:  egressChan,
		ackChan:     ackChan,
		isRemote:    isRemote,
	}
}

func (dq *DataQueue) InsertBlock(db *DataBlock) error {
	dq.mutex.Lock()
	defer dq.mutex.Unlock()
	_, in := dq.IsDataBlockIn(db.blockID)
	if !in {
		dq.blocks = append(dq.blocks, db)
		dq.len++
		if dq.len == 1 {
			dq.headBlockID = db.blockID
			dq.tailBlockID = db.blockID
		} else {
			dq.tailBlockID = db.blockID
		}
		return nil
	}
	return fmt.Errorf("blockid: %d already exists", db.blockID)
}

func (dq *DataQueue) InsertFragment(f *DataFragment) {
	if f.blockID == dq.headBlockID {
		dq.blocks[0].InsertFragment(f)
	} else {
		i, in := dq.IsDataBlockIn(f.blockID)
		if in {
			dq.blocks[i].InsertFragment(f)
		} else {
			db := NewDataBlockFromFragment(f)
			dq.InsertBlock(db)
		}
	}
}

func (dq *DataQueue) IsDataBlockIn(blockID int) (int, bool) {
	if len(dq.blocks) == 0 {
		return 0, false
	}
	for i := range dq.blocks {
		if dq.blocks[i].blockID == blockID {
			return i, true
		}
	}
	return 0, false
}
