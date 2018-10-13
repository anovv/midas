package arb

import (
	"midas/common"
)

type Triangle struct {
	PairAB *common.CoinPair
	PairBC *common.CoinPair
	PairAC *common.CoinPair
	CoinA common.Coin
	CoinB common.Coin
	CoinC common.Coin
	Key string
}
