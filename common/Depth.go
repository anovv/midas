package common

import (
	"encoding/json"
)

type DepthRecord struct {
	Price,
	Amount float64
}

type Depth struct {
	AskList,
	BidList DepthRecords
	LastUpdateId float64
}

type DepthRecords []DepthRecord

func (dr DepthRecords) Len() int {
	return len(dr)
}

func (dr DepthRecords) Swap(i, j int) {
	dr[i], dr[j] = dr[j], dr[i]
}

func (dr DepthRecords) Less(i, j int) bool {
	return dr[i].Price < dr[j].Price
}

func (depth *Depth) Serialize() string {
	out, err := json.Marshal(depth)
	if err != nil {
		panic (err)
	}

	return string(out)
}

func DeserializeDepth(depthSerialized string) *Depth {
	var depth *Depth
	err := json.Unmarshal([]byte(depthSerialized), &depth)
	if err != nil {
		panic(err)
	}

	return depth
}
