package main

import (
	"log"
	"time"
)

type Manager struct {
	scheduler           *Scheduler
	fragSize            int
	stats               *Stats
	parityCount         int
	deadline            time.Duration
	retransmitThreshold float64
}

func NewManager(cfg *Config, scheduler *Scheduler, fragSize int,
) (*Manager, error) {
	stats := NewStats(20)
	manager := &Manager{
		scheduler:           scheduler,
		fragSize:            fragSize,
		stats:               stats,
		parityCount:         2,
		deadline:            cfg.Deadline.Duration,
		retransmitThreshold: 0.75,
	}
	return manager, nil
}

func (m *Manager) Run() {
	rtDeadline := time.Duration(int64(float64(m.deadline) * (m.retransmitThreshold)))
	m.scheduler.unAckQ.deadline = &rtDeadline
	go func() {
		for {
			loss := <-m.scheduler.packetLossChan
			m.stats.InsertPacketLoss(loss)
			avg := m.stats.GetAveragePacketLoss()

			if float64(loss) > (avg * 1.2) {
				m.parityCount++
			} else if float64(loss) < (avg*0.8) && m.parityCount > 2 {
				m.parityCount--
			}
			log.Printf("Loss: %d, Avg: %f, ParityCount: %d\n", loss, avg, m.parityCount)
			rtDeadline := time.Duration(int64(float64(m.deadline) * (m.retransmitThreshold)))
			m.scheduler.unAckQ.deadline = &rtDeadline
		}
	}()

}
