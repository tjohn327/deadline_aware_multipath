package main

import "sync"

type Stats struct {
	packetLoss []float64
	length     int
	mutex      sync.Mutex
}

func NewStats(length int) *Stats {
	stats := &Stats{
		packetLoss: make([]float64, 0),
		length:     length,
	}
	return stats
}

func (s *Stats) InsertPacketLoss(loss float64) {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	if len(s.packetLoss) < s.length {
		s.packetLoss = append(s.packetLoss, loss)
	} else {
		s.packetLoss = append(s.packetLoss[1:], loss)
	}
}

func (s *Stats) GetLatestPacketLoss() float64 {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	loss := 0.0
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
		sum := 0.0
		for _, v := range s.packetLoss {
			sum = sum + v
		}
		avg = float64(sum) / float64(s.length)
	}
	return avg
}

func (s *Stats) GetMaxPacketLoss() float64 {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	max := 0.0
	if len(s.packetLoss) > 0 {
		max = s.packetLoss[0]
		for _, v := range s.packetLoss {
			if v > max {
				max = v
			}
		}
	}
	return max
}
