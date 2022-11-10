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
	packetLossChan     chan float64
}

func NewScheduler(ctx context.Context, cfg *Config) (*Scheduler, error) {
	sendSelector := SendSelector{} //TODO: implement custom selector
	retransmitSelector := SendSelector{}
	scionIngressChan := make(chan []byte, 500)
	scionAckChan := make(chan []byte, 500)
	scionRestransmitChan := make(chan []byte, 500)
	retransmitChan := make(chan *DataBlock, 500)
	ingressChan := make(chan *DataBlock, 500)
	paritySelector := SendSelector{}
	ingressChanParity := make(chan []byte, 500)
	packetLossChan := make(chan float64, 100)
	sender, err := NewScionSender(ctx, &cfg.Remote.ScionAddr,
		&cfg.Listen_port, &sendSelector, &retransmitSelector, scionIngressChan,
		scionAckChan, scionRestransmitChan, &paritySelector, ingressChanParity)
	if err != nil {
		return nil, err
	}
	// sender.sendSelector.(*SendSelector).GetPathCount()
	// sender.sendSelector.(*SendSelector).SetPath_s()
	// sender.retransmitSelector.(*SendSelector).GetPathCount()
	// sender.retransmitSelector.(*SendSelector).SetPath_r()
	// sender.paritySelector.(*SendSelector).GetPathCount()
	// sender.paritySelector.(*SendSelector).SetPath_r()
	// fmt.Println("paths", sender.sendSelector.(*SendSelector).GetPathCount())
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
			s.unAckQ.Enqueue(block)
			for _, v := range block.fragments {
				if v.isParity {
					s.sender.ingressChanParity <- v.data
				} else {
					s.sender.ingressChan <- v.data
				}
				// time.Sleep(10 * time.Microsecond)
			}
		}
	}()
	//retransmit
	go func() {
		for {
			block := <-s.unAckQ.egressChan
			for _, v := range block.fragments {
				if !v.acked && v.retransmit {
					s.sender.retransmitChan <- v.data
					// log.Println("retransmit")
				}
				// if !v.acked {
				// 	s.sender.retransmitChan <- v.data
				// }
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
