package main

import (
	"sync"

	"github.com/netsec-ethz/scion-apps/pkg/pan"
)

type SendSelector struct {
	mutex   sync.Mutex
	paths   []*pan.Path
	current int
}

func (s *SendSelector) Path() *pan.Path {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	if len(s.paths) == 0 {
		return nil
	}
	return s.paths[s.current]
}

func (s *SendSelector) SetPaths(remote pan.UDPAddr, paths []*pan.Path) {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	newcurrent := 0
	if len(s.paths) > 0 {
		currentFingerprint := s.paths[s.current].Fingerprint
		for i, p := range paths {
			if p.Fingerprint == currentFingerprint {
				newcurrent = i
				break
			}
		}
	}
	s.paths = paths
	s.current = newcurrent
}

func (s *SendSelector) SetPath(pathIndex int) {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	if len(s.paths) > pathIndex {
		s.current = pathIndex
	}
}

func (s *SendSelector) GetPathCount() int {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	return len(s.paths)
}

func (s *SendSelector) OnPathDown(pf pan.PathFingerprint, pi pan.PathInterface) {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	// current := s.paths[s.current]
	// if isInterfaceOnPath(current, pi) || pf == current.Fingerprint {
	// 	fmt.Println("down:", s.current, len(s.paths))
	// 	better := stats.FirstMoreAlive(current, s.paths)
	// 	if better >= 0 {
	// 		// Try next path. Note that this will keep cycling if we get down notifications
	// 		s.current = better
	// 		fmt.Println("failover:", s.current, len(s.paths))
	// 	}
	// }
}

func (s *SendSelector) Close() error {
	return nil
}
