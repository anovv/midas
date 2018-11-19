package brain

import (
	"midas/apis/binance"
	"strings"
	"midas/common"
	"log"
	"math"
	"time"
	"fmt"
)

var weights = map[string]float64{
	"BTC" : 0.1,
	"ETH" : 0.05,
	"BNB" : 0.05,
	"USDT" : 0.025,
}

const (
	// estimation error relative to BTC balance (e.g. how much we tolerate to lose due to price fluctuation, fees, etc)
	VALUE_ERR = 0.1

	// how much we tolerate for real balance to diverge from estimated btc value to NOT execute a rebalance trade
	EXECUTION_THRESHOLD = 0.1
)

func RebalancePortfolio() {
	log.Println("Rebalancing portfolio...")
	checkWeights()
	checkAccountInfoIsFetched()

	qtys := projectBTCQtys()
	log.Println("Qtys: ", qtys)
	trades := scheduleTrades(qtys)
	log.Println("Trades: ", trades)
	log.Println("Num Trades: ", len(trades))
	executeTrades(trades)
	log.Println("Rebalancing portfolio... Done")
}

func projectBTCQtys() map[string]float64 {
	totalEstimatedBTCQty := 0.0
	numCoins := 0
	projectedBTCQtys := make(map[string]float64)

	// TODO make sure we use only arb coins and arb pairs
	// Find eligible coins and estimate total BTC value
	for _, balance := range account.Balances {
		estimatedBTCQty, _, _, _ := estimateBTCQty(balance, false)
		if estimatedBTCQty < 0 {
			continue
		}

		projectedBTCQtys[balance.CoinSymbol] = 0.0
		totalEstimatedBTCQty += estimatedBTCQty
		numCoins++
	}

	// Calc projected btc values for weighted coins first
	totalWeight := 1.0
	for coinSymbol, weight := range weights {
		projectedBTCQtys[coinSymbol] = totalEstimatedBTCQty * weight
		totalWeight -= weight
		numCoins--
	}

	log.Println("Total estimated BTC: ", totalEstimatedBTCQty)
	log.Println("Num coins: ", numCoins + len(weights))

	// Calc projected btc values for other coins
	weightPerCoin := totalWeight/float64(numCoins)
	for coinSymbol, _ := range projectedBTCQtys {
		if _, ok := weights[coinSymbol]; !ok {
			projectedBTCQtys[coinSymbol] = weightPerCoin * totalEstimatedBTCQty
		}
	}

	return projectedBTCQtys
}

func estimateBTCQty(coinBalance *common.Balance, shouldPanic bool) (float64, float64, string, common.OrderSide) {
	coinSymbol := coinBalance.CoinSymbol
	coinQty := coinBalance.Free
	if strings.Compare(coinSymbol, "BTC") == 0 {
		return coinQty, 1.0, "", ""
	}

	pairSymbol := coinSymbol + "BTC"
	side := common.SideSell
	if !HasPair(pairSymbol) {
		pairSymbol = "BTC" + coinSymbol
		side = common.SideBuy
		if !HasPair(pairSymbol) {
			msg := "Pair for BTC and " + coinSymbol + " does not exist"
			if shouldPanic {
				panic(msg)
			} else {
				log.Println(msg)
				return -1, 0, "", ""
			}
		}
	}

	ticker := (*tickersMap)[pairSymbol]
	if ticker == nil {
		msg := "Ticker for BTC and " + coinSymbol + " does not exist"
		if shouldPanic {
			panic(msg)
		} else {
			log.Println(msg)
			return -1, 0, "", ""
		}
	}

	midPrice := (ticker.BidPrice + ticker.AskPrice)/2.0

	estimatedBTCQty := 0.0
	if side == common.SideSell {
		estimatedBTCQty = coinQty * midPrice
	} else {
		estimatedBTCQty = coinQty / midPrice
	}

	return estimatedBTCQty, midPrice, pairSymbol, side
}

func scheduleTrades(projectedBTCQtys map[string]float64) []*common.OrderRequest {
	toBTC := make([]*common.OrderRequest, 0)
	fromBTC := make([]*common.OrderRequest, 0)
	for coinSymbol, projectedBTCQty := range projectedBTCQtys {
		if strings.Compare(coinSymbol, "BTC") == 0 {
			continue
		}
		estimatedBTCQty, midPrice, pairSymbol, side := estimateBTCQty(account.Balances[coinSymbol], true)

		deltaBTCQty := math.Abs(projectedBTCQty - estimatedBTCQty)
		deltaCoinQty := 0.0
		if side == common.SideSell {
			deltaCoinQty = deltaBTCQty/midPrice
		} else {
			deltaCoinQty = deltaBTCQty * midPrice
		}

		// TODO implement dynamic allocation based on amount BTC left
		if projectedBTCQty < estimatedBTCQty {
			// to BTC
			if deltaBTCQty > EXECUTION_THRESHOLD * estimatedBTCQty {
				toBTC = append(toBTC, &common.OrderRequest{
					pairSymbol,
					side,
					common.TypeMarket,
					FormatQty(pairSymbol, deltaCoinQty),
					0.0,
				})
			}
		} else {
			// from BTC
			if deltaBTCQty > EXECUTION_THRESHOLD * estimatedBTCQty {
				if side == common.SideSell {
					side = common.SideBuy
					deltaCoinQty = deltaCoinQty * (1 - VALUE_ERR)
				} else {
					side = common.SideSell
					deltaCoinQty = deltaBTCQty * (1 - VALUE_ERR)
				}
				fromBTC = append(fromBTC, &common.OrderRequest{
					pairSymbol,
					side,
					common.TypeMarket,
					FormatQty(pairSymbol, deltaCoinQty),
					0.0,
				})
			}
		}


	}
	return append(toBTC, fromBTC...)
}

func executeTrades(scheduledTrades []*common.OrderRequest) {
	// TODO implement proper executor
	for _, orderRequest := range scheduledTrades {
		ts := common.UnixMillis(time.Now())
		clientOrderId := fmt.Sprintf("%s%d", orderRequest.Symbol, ts)
		res, err := binance.NewOrder(
			orderRequest.Symbol,
			orderRequest.Side,
			orderRequest.Type,
			orderRequest.Qty,
			orderRequest.Price,
			clientOrderId,
			ts,
			EXECUTION_MODE_TEST,
		)

		if err != nil {
			log.Println("Qty: ", orderRequest.Qty)
			log.Println("Symbol: ", orderRequest.Symbol)
			log.Println("Min notional", GetMinNotional(orderRequest.Symbol))
		} else {
			log.Println("Executed trade: ", orderRequest)
			log.Println("Executed trade res: ", res)
		}
	}
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