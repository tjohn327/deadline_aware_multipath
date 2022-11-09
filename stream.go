package main

import "time"

type Stream struct {
	streamID    int
	priority    int
	deadline    time.Duration
	pktSize     int
	parityCount int
	InChan      chan []byte
	OutChan     chan []byte
}
