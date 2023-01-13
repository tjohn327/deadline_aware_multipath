package main

import (
	"context"
)

type Receiver struct {
	receiver    *ScionReceiver
	egressChan  chan *DataBlock
	receiveQ    map[int]*DataQueue
	receiveMemQ map[int]*DataQueue
}

func NewReceiver(ctx context.Context, cfg *Config, numStreams int) (*Receiver, error) {
	ackSelector := &SendSelector{}
	egressChan := make(chan *DataBlock, 500)
	ackChan := make(chan []byte, 500)
	reInChan := make(chan []byte, 500)
	parityChan := make(chan []byte, 500)
	scionReceiver, err := NewScionReceiver(ctx, &cfg.Remote.ScionAddr, &cfg.Listen_port,
		ackSelector, nil, reInChan, ackChan, parityChan)
	if err != nil {
		return nil, err
	}
	scionReceiver.ackSelector.(*SendSelector).GetPathCount()
	scionReceiver.ackSelector.(*SendSelector).SetPath_r()

	receiveQ := make(map[int]*DataQueue)
	receiveMemQ := make(map[int]*DataQueue)
	for i := 0; i < numStreams; i++ {
		egressChanQ := make(chan *DataBlock, 500)
		receiveQ[i] = NewDataQueue(Receive, nil, egressChanQ, &cfg.Deadline.Duration, nil, i)
		receiveMemQ[i] = NewDataQueue(ReceiveMem, nil, nil, nil, nil, i)
	}

	receiver := &Receiver{
		receiver:    scionReceiver,
		egressChan:  egressChan,
		receiveQ:    receiveQ,
		receiveMemQ: receiveMemQ,
	}
	return receiver, nil
}

func (r *Receiver) Run() {
	go func() {
		for {
			select {
			case buf := <-r.receiver.reInChan:
				frag, err := NewFragmentFromBytes(buf)
				if err == nil {
					frag.isRetr = true
					// if !r.receiveMemQ.IsDataBlockIn(frag.blockID) {
					r.receiveQ[frag.streamID].ingressChan <- frag
					// }
				}

			case buf := <-r.receiver.egressChan:
				frag, err := NewFragmentFromBytes(buf)
				ack := frag.GetAckBytes()
				if err == nil {
					// if !r.receiveMemQ.IsDataBlockIn(frag.blockID) {
					r.receiveQ[frag.streamID].ingressChan <- frag
					// }
					// if frag.fragType == int(NORETRANSMIT) {
					// 	continue
					// }
					r.receiver.ackChan <- ack
				}
			case buf := <-r.receiver.parityChan:
				frag, err := NewFragmentFromBytes(buf)
				if err == nil {
					frag.isParity = true
					r.receiveQ[frag.streamID].ingressChan <- frag
				}
			}

		}
	}()

	for _, rq := range r.receiveQ {
		q := rq
		go func() {
			for {
				block := <-q.egressChan
				// r.receiveMemQ.InsertBlock(block)
				if block.canDecode {
					// fmt.Println(block.blockID, block.currentCount, block.fragmentCount, block.canDecode)
					r.egressChan <- block
				}

			}
		}()
	}
}
