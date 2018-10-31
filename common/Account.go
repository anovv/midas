package common

import "time"

type Account struct {
	MakerCommission   int64
	TakerCommission  int64
	BuyerCommission  int64
	SellerCommission int64
	LastUpdateTs time.Time
	Balances        []*Balance
}

type Balance struct {
	Coin Coin
	Free float64
	Locked float64
}