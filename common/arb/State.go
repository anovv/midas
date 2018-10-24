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
	Reported bool
	NumFrames int
}
