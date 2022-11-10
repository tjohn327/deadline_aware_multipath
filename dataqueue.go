package main

import (
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
	MAX_QUEUE_LENGTH    = 1000
	defaultTrimInterval = 30 * time.Second
)

//Ring buffer implementation of DataQueue
type DataQueue struct {
	started             bool
	queueType           DataQueueType
	head                int
	tail                int
	len                 int
	prevCompleteBlockId int
	currentBlockId      int
	deadline            *time.Duration
	packetLossChan      chan float64
	ingressChan         chan *DataFragment
	egressChan          chan *DataBlock
	mutex               sync.Mutex
	blocks              []*DataBlock
}

func NewDataQueue(t DataQueueType, ingressChan chan *DataFragment, egressChan chan *DataBlock,
	deadline *time.Duration, packetLossChan chan float64) *DataQueue {

	if ingressChan == nil {
		ingressChan = make(chan *DataFragment, 200)
	}

	dq := &DataQueue{
		queueType:           t,
		len:                 MAX_QUEUE_LENGTH,
		prevCompleteBlockId: -1,
		currentBlockId:      -1,
		deadline:            deadline,
		blocks:              make([]*DataBlock, MAX_QUEUE_LENGTH),
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
	// if t == ReceiveMem {
	// 	dq.runTrim()
	// }
	return dq
}

func (dq *DataQueue) runReceive() {
	go func() {
		for {
			f := <-dq.ingressChan
			dq.InsertFragment(f)
		}
	}()

	// go func() {
	// 	for {
	// 		out, _ := dq.GetDataBlock()

	// 		if out != nil {
	// 			dq.egressChan <- out
	// 		}
	// 		time.Sleep(10 * time.Millisecond)
	// 	}

	// }()
}

func (dq *DataQueue) runAck() {
	go func() {
		for {
			f := <-dq.ingressChan
			dq.processACK(f)
		}
	}()
}

// func (dq *DataQueue) runTrim() {
// 	go func() {
// 		for {
// 			if dq.len > MAX_QUEUE_LENGTH {
// 				dq.mutex.Lock()
// 				dq.blocks = dq.blocks[dq.len-MAX_QUEUE_LENGTH:]
// 				dq.len = len(dq.blocks)
// 				if dq.len > 0 {
// 					dq.headBlockID = dq.blocks[0].blockID
// 				}
// 				dq.mutex.Unlock()
// 			}
// 			time.Sleep(defaultTrimInterval)
// 		}
// 	}()
// }

func (dq *DataQueue) retransmit(blockID int) {
	deadline := (*dq.deadline)
	timer := time.NewTimer(deadline)
	<-timer.C
	dq.mutex.Lock()
	defer dq.mutex.Unlock()
	if i, ok := dq.isBlockIDInQueue(blockID); ok {
		out := dq.blocks[i]
		// loss := float64(out.unackedCount) / float64(out.fragmentCount)
		loss := float64(out.unackedCount)
		retr := 0

		if out.unackedCount > out.parityCount {
			// log.Printf("retransmit- block: %d fragments: %d", out.blockID, out.unackedCount)
			if out.unackedCount < (out.fragmentCount-out.parityCount)/2 {
				dq.egressChan <- out
				retr = out.unackedCount
			}
			// loss = out.unackedCount - out.parityCount
		}
		receiveRetrChan <- TimeEntry{id: out.blockID, retr: retr, loss: out.unackedCount}
		if dq.packetLossChan != nil {
			dq.packetLossChan <- loss
		}
	}

}

func (dq *DataQueue) Enqueue(db *DataBlock) bool {
	if dq.IsFull() {
		dq.Dequeue()
	}
	dq.blocks[dq.tail] = db
	dq.tail = (dq.tail + 1) % dq.len
	dq.currentBlockId = db.blockID
	if dq.queueType == UnAck {
		go dq.retransmit(db.blockID)
	}
	return true
}

func (dq *DataQueue) Dequeue() *DataBlock {
	if dq.IsEmpty() {
		return nil
	}
	out := dq.blocks[dq.head]
	dq.head = (dq.head + 1) % dq.len
	return out
}

func (dq *DataQueue) IsEmpty() bool {
	return dq.head == dq.tail
}

func (dq *DataQueue) IsFull() bool {
	return (dq.tail+1)%dq.len == dq.head
}

func (dq *DataQueue) isBlockIDInQueue(blockID int) (int, bool) {
	// if dq.currentBlockId == blockID {
	// 	return dq.tail, true
	// }
	for i := dq.head; i != dq.tail; i = (i + 1) % dq.len {
		if dq.blocks[i].blockID == blockID {
			return i, true
		}
	}
	return -1, false
}

func (dq *DataQueue) eject() {
	time.Sleep(40 * time.Millisecond)
	// deadline := time.Duration((20) * time.Millisecond)
	// timer := time.NewTimer(deadline)
	// <-timer.C
	for {
		if !dq.IsEmpty() {
			dq.mutex.Lock()
			if dq.blocks[dq.head].canDecode {
				out := dq.Dequeue()
				dq.prevCompleteBlockId = out.blockID
				dq.egressChan <- out
			}
			// else {
			// 	if dq.blocks[(dq.head+1)%dq.len].canDecode {
			// 	dq.Dequeue()
			// 	out := dq.Dequeue()
			// 	dq.prevCompleteBlockId = out.blockID
			// 	dq.egressChan <- out
			// }
			dq.mutex.Unlock()

		}
		time.Sleep(1 * time.Millisecond)
	}
	// if i, ok := dq.isBlockIDInQueue(blockID); ok {
	// 	out := dq.blocks[i]
	// 	dq.prevCompleteBlockId = out.blockID
	// 	dq.head = (i + 1) % dq.len
	// 	if dq.blocks[i].canDecode {
	// 		dq.egressChan <- out
	// 	}
	// }
}

func (dq *DataQueue) InsertFragment(f *DataFragment) {
	dq.mutex.Lock()
	defer dq.mutex.Unlock()
	if f.blockID <= dq.prevCompleteBlockId {
		return
	}
	if i, ok := dq.isBlockIDInQueue(f.blockID); ok {
		dq.blocks[i].InsertFragment(f)
		// if dq.blocks[i].canDecode {
		// 	dq.prevCompleteBlockId = dq.blocks[i].blockID
		// 	dq.head = (i + 1) % dq.len
		// 	dq.egressChan <- dq.blocks[i]
		// }
	} else {
		db := NewDataBlockFromFragment(f)
		dq.Enqueue(db)
		if !dq.started {
			dq.started = true
			go dq.eject()
		}
	}
	// if dq.blocks[dq.head].canDecode {
	// 	out := dq.Dequeue()
	// 	dq.prevCompleteBlockId = out.blockID
	// 	dq.egressChan <- out
	// }

}

func (dq *DataQueue) walkQueue() {
	for i := dq.head; i != dq.tail; i = (i + 1) % dq.len {
		if dq.blocks[i].canDecode {
			if i != dq.head {
				dq.head = i
			}
			out := dq.Dequeue()
			dq.prevCompleteBlockId = out.blockID
			dq.egressChan <- out
		}
	}
}

func (dq *DataQueue) processACK(f *DataFragment) {
	dq.mutex.Lock()
	defer dq.mutex.Unlock()
	if f.blockID <= dq.prevCompleteBlockId {
		return
	}
	if i, ok := dq.isBlockIDInQueue(f.blockID); ok {
		dq.blocks[i].AcknowledgeFragment(f)
	}
}

// func (dq *DataQueue) insertBlock(db *DataBlock) error {
// 	_, in := dq.isDataBlockIn(db.blockID)
// 	if !in {
// 		dq.blocks = append(dq.blocks, db)
// 		dq.len++
// 		if dq.len == 1 {
// 			dq.headBlockID = db.blockID
// 			dq.tailBlockID = db.blockID
// 		} else {
// 			dq.tailBlockID = db.blockID
// 		}
// 		if dq.queueType == UnAck {
// 			go dq.retransmit(db.blockID)
// 		}
// 		return nil
// 	}
// 	return fmt.Errorf("blockid: %d already exists", db.blockID)
// }

// func (dq *DataQueue) InsertBlock(db *DataBlock) error {
// 	dq.mutex.Lock()
// 	defer dq.mutex.Unlock()
// 	_, in := dq.isDataBlockIn(db.blockID)
// 	if !in {
// 		dq.blocks = append(dq.blocks, db)
// 		dq.len++
// 		if dq.len == 1 {
// 			dq.headBlockID = db.blockID
// 			dq.tailBlockID = db.blockID
// 		} else {
// 			dq.tailBlockID = db.blockID
// 		}
// 		if dq.queueType == UnAck {
// 			go dq.retransmit(db.blockID)
// 		}
// 		return nil
// 	}
// 	return fmt.Errorf("blockid: %d already exists", db.blockID)
// }

// func (dq *DataQueue) InsertFragment(f *DataFragment) {
// 	dq.mutex.Lock()
// 	defer dq.mutex.Unlock()
// 	if f.blockID < dq.prevCompleteBlockId && dq.prevCompleteBlockId <= 65534 {
// 		return
// 	}
// 	// if f.blockID-1 > dq.prevCompleteBlockId {
// 	// 	x := dq.prevCompleteBlockId + 1
// 	// 	for j := x; j < f.blockID; j++ {
// 	// 		if i, ok := dq.isDataBlockIn(j); ok {
// 	// 			if dq.blocks[i].canDecode && dq.blocks[i].blockID > dq.prevCompleteBlockId {
// 	// 				dq.prevCompleteBlockId = dq.blocks[i].blockID
// 	// 				dq.egressChan <- dq.blocks[i]
// 	// 			}
// 	// 		}
// 	// 	}
// 	// }

// 	i, in := dq.isDataBlockIn(f.blockID)
// 	if in {
// 		_, err := dq.blocks[i].InsertFragment(f)
// 		if err != nil {
// 			log.Println("insert fragment:", err)
// 			return
// 		}
// 		if dq.blocks[i].canDecode && dq.blocks[i].blockID > dq.prevCompleteBlockId {
// 			dq.prevCompleteBlockId = dq.blocks[i].blockID
// 			dq.egressChan <- dq.blocks[i]
// 		}

// 	} else {
// 		db := NewDataBlockFromFragment(f)
// 		dq.insertBlock(db)
// 	}
// }

// func (dq *DataQueue) GetDataBlock() (*DataBlock, error) {
// 	// dq.mutex.Lock()
// 	// defer dq.mutex.Unlock()
// 	if dq.len == 0 {
// 		err := fmt.Errorf("queue empty")
// 		return nil, err
// 	}
// 	if dq.blocks[0].canDecode {
// 		out := dq.getDataBlock(0)
// 		if out != nil {
// 			dq.prevCompleteBlockId = out.blockID
// 			return out, nil
// 		}
// 	} else if dq.blocks[0].inTime.Add(time.Duration(500 * time.Millisecond)).After(time.Now()) {
// 		out := dq.getDataBlock(0)
// 		fmt.Println("time out", out.blockID)
// 		if out != nil {
// 			return out, nil
// 		}
// 	}
// 	return nil, nil
// }

// func (dq *DataQueue) getDataBlockByID(blockID int) *DataBlock {
// 	i, in := dq.isDataBlockIn(blockID)
// 	if !in {
// 		return nil
// 	}
// 	out := dq.getDataBlock(i)
// 	return out
// }

// func (dq *DataQueue) getDataBlock(i int) *DataBlock {
// 	dq.mutex.Lock()
// 	defer dq.mutex.Unlock()
// 	if i >= dq.len || dq.len == 0 {
// 		return nil
// 	}
// 	out := dq.blocks[i]
// 	if i == 0 && dq.len == 1 {
// 		dq.blocks = dq.blocks[:0]
// 	} else if i == 0 && dq.len > 1 {
// 		dq.blocks = dq.blocks[1:]
// 	} else {
// 		dq.blocks = append(dq.blocks[:i], dq.blocks[i+1:]...)
// 	}
// 	dq.len--
// 	if dq.len > 0 {
// 		dq.headBlockID = dq.blocks[0].blockID
// 		dq.tailBlockID = dq.blocks[len(dq.blocks)-1].blockID
// 	} else {
// 		dq.headBlockID = 0
// 		dq.tailBlockID = 0
// 	}

// 	return out
// }

func copyBlock(dbin *DataBlock) *DataBlock {
	dbout := &DataBlock{
		blockID:       dbin.blockID,
		fragments:     make([]*DataFragment, dbin.fragmentCount),
		parityCount:   dbin.parityCount,
		padlen:        dbin.padlen,
		complete:      dbin.complete,
		fragmentCount: dbin.fragmentCount,
		canDecode:     dbin.canDecode,
		inTime:        dbin.inTime,
		unackedCount:  dbin.unackedCount,
		currentCount:  dbin.currentCount,
	}
	for i, f := range dbin.fragments {
		if f != nil {
			if f.data != nil {
				dbout.fragments[i] = &DataFragment{
					blockID:    f.blockID,
					fragmentID: f.fragmentID,
					data:       make([]byte, len(f.data)),
				}
				if f.data != nil {
					copy(dbout.fragments[i].data, f.data)
				}
			} else {
				dbout.fragments[i] = &DataFragment{
					blockID:    f.blockID,
					fragmentID: f.fragmentID,
					data:       nil,
				}
			}
		}

	}

	return dbout
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

// func (dq *DataQueue) processACK(f *DataFragment) {
// 	dq.mutex.Lock()
// 	defer dq.mutex.Unlock()
// 	i, in := dq.isDataBlockIn(f.blockID)
// 	if in {
// 		dq.blocks[i].AcknowledgeFragment(f)
// 	}
// }

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
