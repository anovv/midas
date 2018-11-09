package brain

//import "strings"

var weights = map[string]float64{
	"BNB" : 0.25,
}

func rebalancePortfolio() {
	checkWeights()
	checkCoinsAreFetched()
	checkAccountInfoIsFetched()
	convertAllToBTC()
}

// sells all coins for BTC
func convertAllToBTC() {
	//totalBTC := 0.0
	//
	//for _, balance := range account.Balances {
	//	coinSymbol := balance.Coin.CoinSymbol
	//	if strings.Compare(coinSymbol, "BNB") == 0 {
	//		continue
	//	}
	//	pairSymbol := coinSymbol + "BNB"
	//	coinQty := balance.Free
	//	ticker := (*tickersMap)[pairSymbol]
	//	// TODO check nil ticker
	//	midPrice := (ticker.AskPrice + ticker.BidPrice)/2
	//	BTCQty := coinQty/midPrice
	//	totalBNB += BTCQty
	//}
	//
	//avgBTCQty := totalBTC/numCoins
}

func distributeBTC() {

}

func checkWeights() {
	sum := 0.0

	for _, weight := range weights {
		sum += weight
	}

	if sum > 1.0 {
		panic("Portfolio rebalancer error: weights are incorrect")
	}
}

func checkCoinsAreFetched() {
	if len(arbCoins) == 0 {
		panic("Portfolio rebalancer error: rebalancing before arb coins are fetched")
	}
}

func checkAccountInfoIsFetched() {
	if account == nil {
		panic("Portfolio rebalancer error: rebalancing before account info is fetched")
	}
}