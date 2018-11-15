package common

import "time"

type Account struct {
	MakerCommission   int64
	TakerCommission  int64
	BuyerCommission  int64
	SellerCommission int64
	LastUpdateTs time.Time
	Balances		map[string]*Balance
}

type Balance struct {
	CoinSymbol string
	Free float64
	Locked float64
}