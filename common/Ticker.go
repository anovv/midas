package common

import "encoding/json"

type TickersMap map[string]*Ticker

type Ticker struct {
	Symbol string
	BidPrice float64
	BidQty float64
	AskPrice float64
	AskQnty float64
}

func (tickersMap *TickersMap) Serialize() string {
	out, err := json.Marshal(tickersMap)
	if err != nil {
		panic (err)
	}

	return string(out)
}

func DeserializeTickersMap(tickersMapSerialized string) *TickersMap {
	var depth *TickersMap
	err := json.Unmarshal([]byte(tickersMapSerialized), &depth)
	if err != nil {
		panic(err)
	}

	return depth
}
