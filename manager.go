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
	stats := NewStats(5)
	if fragSize >= (MAX_BUFFER_SIZE - 8) {
		fragSize = MAX_BUFFER_SIZE - 8
	}
	manager := &Manager{
		scheduler:           scheduler,
		fragSize:            fragSize,
		stats:               stats,
		parityCount:         2,
		deadline:            cfg.Deadline.Duration,
		retransmitThreshold: 0.7,
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
			if float64(loss) > (avg * 1.05) {
				m.parityCount = m.parityCount + 1
				if m.parityCount > 35 {
					m.parityCount = 35
				}
			} else if float64(loss) < (avg*0.80) && m.parityCount > 2 {
				m.parityCount--
			}
			if loss > 0 {
				log.Printf("loss: %d, avg: %0.2f,  parity count: %d\n", loss, avg, m.parityCount)
			}
			rtDeadline := time.Duration(int64(float64(m.deadline) * (m.retransmitThreshold)))
			m.scheduler.unAckQ.deadline = &rtDeadline
		}
	}()

}
