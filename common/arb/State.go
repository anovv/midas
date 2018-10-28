package arb

import (
	"time"
)

type State struct {
	QtyBefore float64
	QtyAfter float64
	ProfitRelative float64
	Triangle *Triangle
	StartTs time.Time
	LastUpdateTs time.Time
	FrameUpdateTsQueue []time.Time // holds timestamps of frame updates
}

func (s *State) GetFrameUpdateCount() int {
	count := 0
	for _, ts := range s.FrameUpdateTsQueue {
		if ts.Before(s.LastUpdateTs) {
			count++
		}
	}
	return count
}
