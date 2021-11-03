package main

import "sync"

type Stats struct {
	packetLoss []int
	length     int
	mutex      sync.Mutex
}

func NewStats(length int) *Stats {
	stats := &Stats{
		packetLoss: make([]int, 0),
		length:     length,
	}
	return stats
}

func (s *Stats) InsertPacketLoss(loss int) {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	if len(s.packetLoss) < s.length {
		s.packetLoss = append(s.packetLoss, loss)
	} else {
		s.packetLoss = append(s.packetLoss[1:], loss)
	}
}

func (s *Stats) GetLatestPacketLoss() int {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	loss := 0
	if len(s.packetLoss) > 0 {
		loss = s.packetLoss[len(s.packetLoss)-1]
	}
	return loss
}

func (s *Stats) GetAveragePacketLoss() float64 {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	avg := 0.0
	if len(s.packetLoss) > 0 {
		sum := 0
		for _, v := range s.packetLoss {
			sum = sum + v
		}
		avg = float64(sum) / float64(s.length)
	}
	return avg
}
