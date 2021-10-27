package main

import (
	"context"

	"github.com/netsec-ethz/scion-apps/pkg/pan"
)

type Receiver struct {
	receiver    *ScionReceiver
	egressChan  chan *DataBlock
	receiveQ    *DataQueue
	receiveMemQ *DataQueue
}

func NewReceiver(ctx context.Context, cfg *Config) (*Receiver, error) {
	ackSelector := &pan.DefaultSelector{}
	egressChan := make(chan *DataBlock, 10)
	ackChan := make(chan []byte, 10)
	scionReceiver, err := NewScionReceiver(ctx, &cfg.remote.scionAddr, &cfg.listen_port,
		ackSelector, nil, ackChan)
	if err != nil {
		return nil, err
	}

	receiveQ := NewDataQueue(Receive, nil, egressChan, nil)
	receiveMemQ := NewDataQueue(ReceiveMem, nil, nil, nil)

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
			buf := r.receiver.egressChan
			frag, err := NewFragmentFromBytes(buf)
			if err == nil {

			}
		}
	}()
}
