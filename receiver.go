package main

import (
	"context"
)

type Receiver struct {
	receiver    *ScionReceiver
	egressChan  chan *DataBlock
	receiveQ    *DataQueue
	receiveMemQ *DataQueue
}

func NewReceiver(ctx context.Context, cfg *Config) (*Receiver, error) {
	ackSelector := &SendSelector{}
	egressChan := make(chan *DataBlock, 500)
	egressChanQ := make(chan *DataBlock, 500)
	ackChan := make(chan []byte, 500)
	reInChan := make(chan []byte, 500)
	scionReceiver, err := NewScionReceiver(ctx, &cfg.Remote.ScionAddr, &cfg.Listen_port,
		ackSelector, nil, reInChan, ackChan)
	if err != nil {
		return nil, err
	}
	scionReceiver.ackSelector.(*SendSelector).GetPathCount()
	scionReceiver.ackSelector.(*SendSelector).SetPath_r()

	receiveQ := NewDataQueue(Receive, nil, egressChanQ, &deadline, nil)
	receiveMemQ := NewDataQueue(ReceiveMem, nil, nil, nil, nil)

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
					if !r.receiveMemQ.IsDataBlockIn(frag.blockID) {
						r.receiveQ.ingressChan <- frag
					}
				}

			case buf := <-r.receiver.egressChan:
				frag, err := NewFragmentFromBytes(buf)
				ack := frag.GetAckBytes()
				if err == nil {
					if !r.receiveMemQ.IsDataBlockIn(frag.blockID) {
						r.receiveQ.ingressChan <- frag
					}
					// if frag.fragType == int(NORETRANSMIT) {
					// 	continue
					// }
					r.receiver.ackChan <- ack
				}
			}

		}
	}()

	go func() {
		for {
			block := <-r.receiveQ.egressChan
			r.receiveMemQ.InsertBlock(block)
			if block.canDecode {
				// fmt.Println(block.blockID, block.currentCount, block.fragmentCount, block.canDecode)
				r.egressChan <- block
			}

		}
	}()
}
