package main

import (
	"sync"
	"time"
)

const (
	MAX_QUEUE_LENGTH = 20
	trimInterval     = 500 * time.Millisecond
)

type DataQueue struct {
	currentBlockID uint16
	latestBlockID  uint16
	len            int
	mutex          sync.Mutex
	blocks         []*DataBlock
	ingressChan    chan []byte
	egressChan     chan *DataBlock
	ackChan        chan []byte
	isRemote       bool
}

func NewDataQueue(ingressChan chan []byte, egressChan chan *DataBlock, ackChan chan []byte, isRemote bool) *DataQueue {
	if ingressChan == nil {
		ingressChan = make(chan []byte, 10)
	}
	if egressChan == nil {
		egressChan = make(chan *DataBlock, 10)
	}
	if ackChan == nil {
		ackChan = make(chan []byte, 10)
	}
	return &DataQueue{
		currentBlockID: 0,
		latestBlockID:  0,
		len:            0,
		blocks:         make([]*DataBlock, 0),
		ingressChan:    ingressChan,
		egressChan:     egressChan,
		ackChan:        ackChan,
		isRemote:       isRemote,
	}
}

func (q *DataQueue) RunIngress() {
	go func() {
		for {
			buf := <-q.ingressChan
			q.Insert(buf)
		}
	}()
}

func (q *DataQueue) RunACK() {
	if q.isRemote {
		return
	}
	go func() {
		for {
			ack := <-q.ackChan
			q.ProcessACK(ack)
		}
	}()
}

func (q *DataQueue) RunTrim() {
	go func() {
		for {
			if q.len > MAX_QUEUE_LENGTH {
				q.mutex.Lock()
				q.blocks = q.blocks[q.len-MAX_QUEUE_LENGTH:]
				q.len = len(q.blocks)
				q.mutex.Unlock()
			}
			time.Sleep(trimInterval)
		}
	}()
}

func (q *DataQueue) Insert(data []byte) error {
	frag, err := NewFragment(data)
	if err != nil {
		return err
	}
	err = q.InsertFragment(frag)
	if err != nil {
		return err
	}
	if q.isRemote {
		ack := data[:4]
		q.ackChan <- ack
	}
	return nil
}

func (q *DataQueue) InsertFragment(frag *DataFragment) error {
	i, in := q.isInQueue(frag.blockID)
	if in {
		complete, err := q.blocks[i].InsertFragment(frag)
		if err != nil {
			return err
		}
		if complete {
			q.deQueue(i)
		}
		return nil
	}
	frame := NewFrame(frag.blockID)
	_, err := frame.InsertFragment(frag)
	if err != nil {
		return err
	}
	q.InsertFrame(frame)
	return nil
}

func (q *DataQueue) InsertFrame(frame *DataBlock) {
	q.mutex.Lock()
	defer q.mutex.Unlock()
	q.blocks = append(q.blocks, frame)
	q.latestBlockID = frame.blockID
	q.len++
	// fmt.Println("Frame IN", frame.frameID)
}

func (q *DataQueue) deQueue(i int) {
	q.mutex.Lock()
	defer q.mutex.Unlock()
	frame := q.blocks[i]
	q.blocks = q.blocks[i:]
	q.len = len(q.blocks)
	if frame.complete {
		q.egressChan <- frame
	}
}

func (q *DataQueue) GetFrame() *DataBlock {
	q.mutex.Lock()
	defer q.mutex.Unlock()
	if q.len == 0 {
		return nil
	}
	frame := q.blocks[0]
	if !frame.complete {
		return nil
	}
	q.currentBlockID = frame.blockID
	q.blocks = q.blocks[1:]
	q.len--
	return frame
}

func (q *DataQueue) DiscardFrame() *DataBlock {
	q.mutex.Lock()
	defer q.mutex.Unlock()
	if q.len == 0 {
		return nil
	}
	frame := q.blocks[0]
	q.currentBlockID = frame.blockID
	q.blocks = q.blocks[1:]
	q.len--
	return frame
}

func (q *DataQueue) isInQueue(frameID uint16) (int, bool) {
	q.mutex.Lock()
	defer q.mutex.Unlock()
	for i, v := range q.blocks {
		if v.blockID == frameID {
			return i, true
		}
	}
	return 0, false
}

func (q *DataQueue) IsIn(frag *DataFragment) bool {
	q.mutex.Lock()
	defer q.mutex.Unlock()
	for _, v := range q.blocks {
		if v.blockID == frag.blockID {
			return true
		}
	}
	return false
}

// func (q *VideoQueue) removeFrameByID(frameID uint16) {
// 	q.mutex.Lock()
// 	defer q.mutex.Unlock()
// 	i, in := q.isInQueue(frameID)
// 	if !in {
// 		return
// 	}
// 	q.frames = append(q.frames[:i], q.frames[i+1:]...)
// }

func (q *DataQueue) ProcessACK(ackBuf []byte) {
	ack, err := NewFragment(ackBuf)
	// fmt.Println("ACK IN", ack.frameID, ack.fragmentID, q.len)
	if err != nil {
		return
	}
	i, in := q.isInQueue(ack.blockID)
	if in {
		removed := q.blocks[i].removeFragmentByID(ack.fragmentID)
		if removed {
			q.mutex.Lock()
			defer q.mutex.Unlock()
			if q.blocks[i].len == 0 {
				q.blocks = append(q.blocks[:i], q.blocks[i+1:]...)
				q.len--
				// fmt.Println("Frame removed", ack.frameID, ack.fragmentID, q.len)
			}
		}
	}
}
