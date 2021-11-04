package main

import (
	"fmt"
	"log"
	"sync"
	"time"
)

const DefaultDeadline = time.Duration(50 * time.Millisecond)

type DataQueueType int

const (
	Send DataQueueType = iota
	UnAck
	Receive
	ReceiveMem
)
const (
	MAX_QUEUE_LENGTH    = 50
	defaultTrimInterval = 5000 * time.Millisecond
)

type DataQueue struct {
	queueType           DataQueueType
	headBlockID         int
	tailBlockID         int
	len                 int
	prevCompleteBlockId int
	deadline            *time.Duration
	blocks              []*DataBlock
	ingressChan         chan *DataFragment
	egressChan          chan *DataBlock
	packetLossChan      chan int
	mutex               sync.Mutex
}

func NewDataQueue(t DataQueueType, ingressChan chan *DataFragment, egressChan chan *DataBlock,
	deadline *time.Duration, packetLossChan chan int) *DataQueue {

	if ingressChan == nil {
		ingressChan = make(chan *DataFragment, 10)
	}

	dq := &DataQueue{
		queueType:           t,
		headBlockID:         0,
		tailBlockID:         0,
		len:                 0,
		prevCompleteBlockId: -1,
		deadline:            deadline,
		blocks:              make([]*DataBlock, 0),
		ingressChan:         ingressChan,
		egressChan:          egressChan,
		packetLossChan:      packetLossChan,
	}

	if t == Receive {
		dq.runReceive()
	}
	if t == UnAck {
		dq.runAck()
	}
	if t == ReceiveMem {
		dq.runTrim()
	}
	return dq
}

func (dq *DataQueue) runReceive() {
	go func() {
		for {
			f := <-dq.ingressChan
			dq.InsertFragment(f)
		}
	}()
}

func (dq *DataQueue) runAck() {
	go func() {
		for {
			f := <-dq.ingressChan
			dq.processACK(f)
		}
	}()
}

func (dq *DataQueue) runTrim() {
	go func() {
		for {
			if dq.len > MAX_QUEUE_LENGTH {
				dq.mutex.Lock()
				dq.blocks = dq.blocks[dq.len-MAX_QUEUE_LENGTH:]
				dq.len = len(dq.blocks)
				if dq.len > 0 {
					dq.headBlockID = dq.blocks[0].blockID
				}
				dq.mutex.Unlock()
			}
			time.Sleep(defaultTrimInterval)
		}
	}()
}

func (dq *DataQueue) retransmit(blockID int) {
	deadline := (*dq.deadline)
	timer := time.NewTimer(deadline)
	<-timer.C
	out := dq.getDataBlockByID(blockID)
	if out != nil {
		loss := out.unackedCount
		if out.unackedCount > out.parityCount {
			log.Printf("retransmit- block: %d fragaments: %d", out.blockID, out.unackedCount)
			dq.egressChan <- out
		}
		if dq.packetLossChan != nil {
			dq.packetLossChan <- loss
		}
	}

}

func (dq *DataQueue) InsertBlock(db *DataBlock) error {
	dq.mutex.Lock()
	defer dq.mutex.Unlock()
	_, in := dq.isDataBlockIn(db.blockID)
	if !in {
		dq.blocks = append(dq.blocks, db)
		dq.len++
		if dq.len == 1 {
			dq.headBlockID = db.blockID
			dq.tailBlockID = db.blockID
		} else {
			dq.tailBlockID = db.blockID
		}
		if dq.queueType == UnAck {
			go dq.retransmit(db.blockID)
		}
		return nil
	}
	return fmt.Errorf("blockid: %d already exists", db.blockID)
}

func (dq *DataQueue) InsertFragment(f *DataFragment) {
	if f.blockID < dq.prevCompleteBlockId && dq.prevCompleteBlockId <= 65534 {
		return
	}
	i, in := dq.isDataBlockIn(f.blockID)
	if in {
		_, err := dq.blocks[i].InsertFragment(f)
		if err != nil {
			log.Println("insert fragment:", err)
			return
		}
		if dq.blocks[i].canDecode && dq.queueType == Receive && dq.egressChan != nil {
			out := dq.getDataBlock(i)
			if out != nil {
				dq.prevCompleteBlockId = out.blockID
				dq.egressChan <- out
			}
		}
	} else {
		db := NewDataBlockFromFragment(f)
		dq.InsertBlock(db)
	}
}

func (dq *DataQueue) GetDataBlock() (*DataBlock, error) {
	dq.mutex.Lock()
	defer dq.mutex.Unlock()
	if dq.len == 0 {
		err := fmt.Errorf("queue empty")
		return nil, err
	}
	out := dq.getDataBlock(0)
	return out, nil
}

func (dq *DataQueue) getDataBlockByID(blockID int) *DataBlock {
	i, in := dq.isDataBlockIn(blockID)
	if !in {
		return nil
	}
	out := dq.getDataBlock(i)
	return out
}

func (dq *DataQueue) getDataBlock(i int) *DataBlock {
	dq.mutex.Lock()
	defer dq.mutex.Unlock()
	if i >= dq.len || dq.len == 0 {
		return nil
	}
	out := dq.blocks[i]
	if i == 0 && dq.len == 1 {
		dq.blocks = dq.blocks[:0]
	} else if i == 0 && dq.len > 1 {
		dq.blocks = dq.blocks[1:]
	} else {
		dq.blocks = append(dq.blocks[:i], dq.blocks[i+1:]...)
	}
	dq.len--
	if dq.len > 0 {
		dq.headBlockID = dq.blocks[0].blockID
		dq.tailBlockID = dq.blocks[len(dq.blocks)-1].blockID
	} else {
		dq.headBlockID = 0
		dq.tailBlockID = 0
	}
	return out
}

func (dq *DataQueue) isDataBlockIn(blockID int) (int, bool) {
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

func (dq *DataQueue) processACK(f *DataFragment) {
	i, in := dq.isDataBlockIn(f.blockID)
	if in {
		dq.blocks[i].AcknowledgeFragment(f)
	}
}

func (dq *DataQueue) IsDataBlockIn(blockID int) bool {
	dq.mutex.Lock()
	defer dq.mutex.Unlock()
	if len(dq.blocks) == 0 {
		return false
	}
	for i := range dq.blocks {
		if dq.blocks[i].blockID == blockID {
			return true
		}
	}
	return false
}
