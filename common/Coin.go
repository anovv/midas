package common

type Coin struct {
	CoinSymbol string
}

func (c Coin) String() string {
	return c.CoinSymbol
}

type CoinPair struct {
	PairSymbol string
	BaseCoin   Coin
	QuoteCoin  Coin
}

func (c CoinPair) String() string {
	return c.PairSymbol
}