package common

import "strings"

// TODO generalize to arbitrary exchange
var BASE_SYMBOLS = [4]string{"BTC", "ETH", "BNB", "USDT"} // list of base coins on Binance

type Coin struct {
	CoinSymbol string
}

func (c Coin) String() string {
	return c.CoinSymbol
}

type CoinPair struct {
	PairSymbol string
	CoinA Coin
	CoinB Coin
}

func (c CoinPair) String() string {
	return c.PairSymbol
}

func SymbolToPair(pairSymbol string) *CoinPair {
	for _, baseSymbol := range BASE_SYMBOLS {
		// Check suffix of the pair symbol
		suffix := pairSymbol[len(pairSymbol) - len(baseSymbol):]
		if strings.Compare(suffix, baseSymbol) == 0 {
			CoinSymbolA := pairSymbol[:len(pairSymbol) - len(baseSymbol)]
			CoinSymbolB := suffix
			CoinA := Coin{CoinSymbolA}
			CoinB := Coin{CoinSymbolB}
			return &CoinPair{pairSymbol, CoinA, CoinB}
		}
	}
	return nil
}