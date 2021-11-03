package main

import (
	"context"

	"github.com/netsec-ethz/scion-apps/pkg/pan"
)

type Scheduler struct {
	sender             ScionSender
	sendSelector       pan.Selector
	retransmitSelector pan.Selector
	ingressChan        chan *DataBlock
	unAckQ             *DataQueue
	packetLossChan     chan int
}

func NewScheduler(ctx context.Context, cfg *Config) (*Scheduler, error) {
	sendSelector := pan.DefaultSelector{} //TODO: implement custom selector
	retransmitSelector := pan.DefaultSelector{}
	scionIngressChan := make(chan []byte, 10)
	scionAckChan := make(chan []byte, 10)
	scionRestransmitChan := make(chan []byte, 10)
	retransmitChan := make(chan *DataBlock, 10)
	ingressChan := make(chan *DataBlock, 10)
	packetLossChan := make(chan int, 10)
	sender, err := NewScionSender(ctx, &cfg.Remote.ScionAddr,
		&cfg.Listen_port, &sendSelector, &retransmitSelector, scionIngressChan,
		scionAckChan, scionRestransmitChan)
	if err != nil {
		return nil, err
	}

	unAckQ := NewDataQueue(UnAck, nil, retransmitChan, &cfg.Deadline.Duration, packetLossChan)
	scheduler := &Scheduler{
		sender:             *sender,
		sendSelector:       &sendSelector,
		retransmitSelector: &retransmitSelector,
		ingressChan:        ingressChan,
		unAckQ:             unAckQ,
		packetLossChan:     packetLossChan,
	}
	return scheduler, nil
}

func (s *Scheduler) Run() {
	//send
	go func() {
		for {
			block := <-s.ingressChan
			s.unAckQ.InsertBlock(block)
			for _, v := range block.fragments {
				s.sender.ingressChan <- v.data
			}
		}
	}()
	//retransmit
	go func() {
		for {
			block := <-s.unAckQ.egressChan
			for _, v := range block.fragments {
				if !v.acked {
					s.sender.retransmitChan <- v.data
				}
			}
		}
	}()
	//receive and process ack
	go func() {
		for {
			ack := <-s.sender.ackChan
			frag, err := NewFragmentFromBytes(ack)
			if err == nil {
				s.unAckQ.ingressChan <- frag
			}
		}
	}()
}

func (s *Scheduler) Send(db *DataBlock) {
	s.ingressChan <- db
}
