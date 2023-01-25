package main

import (
	"sync"
	"time"
)

type Manager struct {
	scheduler           *Scheduler
	fragSize            int
	stats               *Stats
	parityCount         int
	deadline            time.Duration
	retransmitThreshold float64
	mutex               sync.Mutex
}

func NewManager(cfg *Config, scheduler *Scheduler, fragSize int,
) (*Manager, error) {
	stats := NewStats(50)
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

func (m *Manager) GetParityCount() int {
	// m.mutex.Lock()
	// defer m.mutex.Unlock()
	return m.parityCount
}

func (m *Manager) Run() {
	// rtTime := int64(float64(m.deadline.Milliseconds()) * m.retransmitThreshold)
	// rtDeadline := time.Duration(rtTime) * time.Millisecond
	rtDeadline := time.Duration(60+60+10) * time.Millisecond

	// rtDeadline := time.Duration(55 * time.Millisecond)
	for i := 0; i < numStreams; i++ {
		m.scheduler.unAckQ[i].deadline = &rtDeadline
	}
	// m.scheduler.unAckQ.deadline = &rtDeadline
	// fmt.Println("                                                             $$$$$$$$$$$$$$$$$$$$$$$$$$$$$Packet loss:")
	go func() {
		for {
			loss := <-pktloss
			// fmt.Println("                                                             $$$$$$$$$$$$$$$$$$$$$$$$$$$$$Packet asdfasd:")
			if loss > 70 {
				continue
			}
			m.stats.InsertPacketLoss(loss)
			avg := m.stats.GetMaxPacketLoss()

			newParityCount := int(avg * 1.1)
			// m.mutex.Lock()
			if m.parityCount == 0 {
				m.parityCount = 0
			}
			if newParityCount > m.parityCount {
				m.parityCount = newParityCount
			} else if newParityCount < m.parityCount/2 && m.parityCount > 1 {
				m.parityCount = m.parityCount - 1
			} else {
				m.parityCount = newParityCount
			}
			if m.parityCount > MAX_PARITY {
				m.parityCount = MAX_PARITY
			}
			// m.mutex.Unlock()

			// fmt.Println("$$$$$$$$$$$$$$$$$$$$$$$$$$$$$Packet loss:", loss, newParityCount, m.parityCount)
			// newParityCount := int(loss)
			// m.parityCount = newParityCount
			// if newParityCount > m.parityCount {
			// 	m.parityCount = newParityCount
			// } else if m.parityCount > 0 {
			// 	m.parityCount = m.parityCount - 1
			// }

			// if loss > (avg * 1.02) {
			// 	m.parityCount = m.parityCount + 1

			// } else if loss < (avg*0.98) && m.parityCount > 0 {
			// 	m.parityCount--
			// }
			// m.parityCount = 2
			// if loss > 0 {
			// 	// log.Printf("loss: %d, avg: %0.2f,  parity count: %d\n", loss, avg, m.parityCount)
			// }
			// rtDeadline := time.Duration(int64(float64(m.deadline) * (m.retransmitThreshold)))
			// m.scheduler.unAckQ.deadline = &rtDeadline
		}
	}()

}
