package main

import (
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
		parityCount:         0,
		deadline:            cfg.Deadline.Duration,
		retransmitThreshold: 0.84,
	}
	return manager, nil
}

func (m *Manager) Run() {
	rtTime := int64(float64(m.deadline.Milliseconds()) * m.retransmitThreshold)
	rtDeadline := time.Duration(rtTime) * time.Millisecond
	// rtDeadline := time.Duration(55 * time.Millisecond)
	m.scheduler.unAckQ.deadline = &rtDeadline
	go func() {
		for {
			loss := <-m.scheduler.packetLossChan
			m.stats.InsertPacketLoss(loss)
			avg := m.stats.GetAveragePacketLoss()
			if float64(loss) > (avg * 1.02) {
				m.parityCount = m.parityCount + 2
				if m.parityCount > MAX_PARITY {
					m.parityCount = MAX_PARITY
				}
			} else if float64(loss) < (avg*0.85) && m.parityCount > 2 {
				m.parityCount--
			}
			// m.parityCount = 2
			// if loss > 0 {
			// 	// log.Printf("loss: %d, avg: %0.2f,  parity count: %d\n", loss, avg, m.parityCount)
			// }
			// rtDeadline := time.Duration(int64(float64(m.deadline) * (m.retransmitThreshold)))
			// m.scheduler.unAckQ.deadline = &rtDeadline
		}
	}()

}
