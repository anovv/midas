package arb

import (
	"time"
	"midas/common"
	"encoding/json"
)

type State struct {
	QtyBefore float64
	QtyAfter float64
	ProfitRelative float64
	Triangle *Triangle
	StartTs time.Time
	LastUpdateTs time.Time
	FrameUpdateTsQueue []time.Time // holds timestamps of frame updates
	Orders map[string]*common.OrderRequest
	BalanceA float64
	BalanceB float64
	BalanceC float64
	OrderQtyAB float64
	OrderQtyBC float64
	OrderQtyAC float64
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

func (s *State) String() string {
	b, err := json.Marshal(s)
	if err != nil {
		panic("Error marshaling arb state: " + err.Error())
	}

	return string(b)
}
