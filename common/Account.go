package common

import "time"

type Account struct {
	Balances        []Balance
	LastUpdateTime time.Time
}

type Balance struct {
	Coin Coin
	Balance float64
}

// used to parse ws message
type RawAccount struct {
	Type            string  	`json:"e"`
	Time            float64 	`json:"E"`
	OpenTime        float64 	`json:"t"`
	MakerCommission  int64   	`json:"m"`
	TakerCommission  int64   	`json:"t"`
	BuyerCommission  int64   	`json:"b"`
	SellerCommission int64   	`json:"s"`
	CanTrade        bool    	`json:"T"`
	CanWithdraw     bool    	`json:"W"`
	CanDeposit      bool    	`json:"D"`
	Balances        []RawBalance  	`json:"B"`
}

type RawBalance struct {
	Asset            string `json:"a"`
	AvailableBalance string `json:"f"`
	Locked           string `json:"l"`
}