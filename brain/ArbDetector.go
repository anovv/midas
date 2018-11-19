package brain

import (
	"midas/common"
	"midas/common/arb"
	"sort"
	"strconv"
	"time"
	"strings"
	"log"
	"sync"
	"midas/logging"
	"midas/configuration"
	"math"
)

const (
	BINANCE_DEFAULT_FEE = 0.001
	BINANCE_BNB_FEE = 0.00075
)

var arbTriangles = make(map[string]*arb.Triangle)
var arbCoins = make(map[string]bool)
var arbPairs = make(map[string]bool)

var arbStates = sync.Map{}

var brainConfig = configuration.ReadBrainConfig()

func RunArbDetector() {
	initArbDetector()
	runReportArb()
	runDetectArbBLOCKING()
}

func initArbDetector() {
	log.Println("Initializing arb detector...")
	if allPairs == nil {
		panic("Arb detector error: pairs are not fetched")
	}
	log.Println("Analyzing " + strconv.Itoa(len(allPairs)) + " pairs...")
	tStart := time.Now()

	for _, pairA := range allPairs {
		for _,  pairB := range allPairs {
			for _,  pairC := range allPairs {
				if isTriangle(pairA, pairB, pairC) {
					key, triangle := makeTriangle(pairA, pairB, pairC)

					// Record arb triangle
					if arbTriangles[key] == nil {
						arbTriangles[key] = triangle
					}

					// Record arb coins
					arbCoins[triangle.CoinA.CoinSymbol] = true
					arbCoins[triangle.CoinB.CoinSymbol] = true
					arbCoins[triangle.CoinC.CoinSymbol] = true

					// Record arb pairs
					arbPairs[(*triangle.PairAB).PairSymbol] = true
					arbPairs[(*triangle.PairBC).PairSymbol] = true
					arbPairs[(*triangle.PairAC).PairSymbol] = true
				}
			}
		}
	}

	delta := time.Since(tStart)
	log.Println("Initializing finished in " + delta.String())
	log.Println("Arb triangles: " + strconv.Itoa(len(arbTriangles)))
	log.Println("Arb pairs: " + strconv.Itoa(len(arbPairs)))
	log.Println("Arb coins: " + strconv.Itoa(len(arbCoins)))
	logging.LogLineToFile("Launched at " + time.Now().String(), logging.ARB_STATES_FILE_PATH)
}

func runReportArb() {
	// Goes through all arb states and prints unreported
	go func() {
		log.Println("Checking existing arbs...")
		for {
			arbStates.Range(func(k, v interface{}) bool {
				arbState := v.(*arb.State)
				// If arb state was not updated by detector routine for more than ARB_REPORT_UPDATE_THRESHOLD_MICROS
				// we consider arb opportunity is gone
				if time.Since(arbState.LastUpdateTs) > time.Duration(brainConfig.ARB_REPORT_UPDATE_THRESHOLD_MICROS) * time.Microsecond {
					arbStates.Delete(k)
					logging.QueueEvent(&logging.Event{
						EventType: logging.EventTypeArbState,
						Value: arbState,
					})
				}
				return true
			})
		}
	} ()
}


func runDetectArbBLOCKING() {
	log.Println("Looking for arb opportunities...")
	for {
		for _, triangle := range arbTriangles {
			arbState := findArb(triangle)
			// TODO build queueing system
			if arbState != nil {
				arbStateKey := triangle.Key + "_" + common.FloatToString(arbState.ProfitRelative)
				res, loaded := arbStates.LoadOrStore(arbStateKey, arbState)
				if loaded {
					arbState := res.(*arb.State)
					arbState.LastUpdateTs = time.Now()
				}

				// TODO is it correct?
				// TODO make sure delayed frames do not trigger trade exec
				if filterArbStateForExecution(arbState){
					SubmitOrders(arbState)
				}
			}
		}
	}
}

func filterArbStateForExecution(state *arb.State) bool {
	// TODO min notional here?
	//return state.ProfitRelative > 0.0001 && state.GetFrameUpdateCount() > 0
	return true
}

func findArb(triangle *arb.Triangle) *arb.State {
	if tickersMap == nil {
		return nil
	}

	tickerAB := (*tickersMap)[triangle.PairAB.PairSymbol]
	tickerBC := (*tickersMap)[triangle.PairBC.PairSymbol]
	tickerAC := (*tickersMap)[triangle.PairAC.PairSymbol]

	if tickerAB == nil || tickerBC == nil || tickerAC == nil {
		return nil
	}

	balanceA := account.Balances[triangle.CoinA.CoinSymbol].Free
	balanceB := account.Balances[triangle.CoinB.CoinSymbol].Free
	balanceC := account.Balances[triangle.CoinC.CoinSymbol].Free

	qtyA := 1.0 // we use arbitrary qty first, if prices form arbitrage we calculate tradable qty later

	// Check if prices form arbitrage
	// A->B eth->btc
	qtyB, sideAB, orderQtyAB, priceAB := simTradeWithTicker(qtyA, triangle.CoinA, tickerAB, true)
	// B->C btc->dnt
	qtyC, sideBC, orderQtyBC, priceBC := simTradeWithTicker(qtyB, triangle.CoinB, tickerBC, true)
	// C->A dnt->eth
	newQtyA, sideAC, orderQtyAC, priceAC := simTradeWithTicker(qtyC, triangle.CoinC, tickerAC, true)

	if newQtyA <= qtyA {
		// No arb
		return nil
	}

	// Find max tradable qty equivalent in A
	balanceBinA := simTradeWithPrice(balanceB, priceAB, triangle.CoinB, triangle.PairAB)
	balanceCinA := simTradeWithPrice(balanceC, priceAC, triangle.CoinC, triangle.PairAC)

	var orderQtyABinA float64
	if isBaseCoin(triangle.CoinA, triangle.PairAB) {
		orderQtyABinA = orderQtyAB
	} else {
		orderQtyABinA = simTradeWithPrice(orderQtyAB, priceAB, triangle.CoinB, triangle.PairAB)
	}

	var orderQtyAСinA float64
	if isBaseCoin(triangle.CoinA, triangle.PairAC) {
		orderQtyAСinA = orderQtyAC
	} else {
		orderQtyAСinA = simTradeWithPrice(orderQtyAC, priceAC, triangle.CoinC, triangle.PairAC)
	}

	var orderQtyBCinA float64
	if isBaseCoin(triangle.CoinB, triangle.PairBC) {
		orderQtyBCinA = simTradeWithPrice(orderQtyBC, priceAB, triangle.CoinB, triangle.PairAB)
	} else {
		orderQtyBCinA = simTradeWithPrice(orderQtyBC, priceAC, triangle.CoinC, triangle.PairAC)
	}

	minBalanceInA := math.Min(math.Min(balanceBinA, balanceCinA), balanceA)
	minOrderQtyInA := math.Min(math.Min(orderQtyABinA, orderQtyAСinA), orderQtyBCinA)
	minOrderQtyInA = math.Min(minBalanceInA, minOrderQtyInA)

	var tradeQtyAB float64
	var tradeQtyBC float64
	var tradeQtyAC float64
	// Convert this qty back for each coin
	if isBaseCoin(triangle.CoinA, triangle.PairAB) {
		tradeQtyAB = minOrderQtyInA
	} else {
		tradeQtyAB = simTradeWithPrice(minOrderQtyInA, priceAB, triangle.CoinA, triangle.PairAB)
	}

	if isBaseCoin(triangle.CoinA, triangle.PairAC) {
		tradeQtyAC = minOrderQtyInA
	} else {
		tradeQtyAC = simTradeWithPrice(minOrderQtyInA, priceAC, triangle.CoinA, triangle.PairAC)
	}

	if isBaseCoin(triangle.CoinB, triangle.PairBC) {
		tradeQtyBC = simTradeWithPrice(minOrderQtyInA, priceAB, triangle.CoinA, triangle.PairAB)
	} else {
		tradeQtyBC = simTradeWithPrice(minOrderQtyInA, priceAC, triangle.CoinA, triangle.PairAC)
	}

	// TODO check filters here?
	orders := make(map[string]*common.OrderRequest)
	orders["AB"] = &common.OrderRequest{
		triangle.PairAB.PairSymbol,
		sideAB,
		common.TypeLimit,
		FormatQty(triangle.PairAB.PairSymbol, tradeQtyAB),
		priceAB,
	}
	orders["BC"] = &common.OrderRequest{
		triangle.PairBC.PairSymbol,
		sideBC,
		common.TypeLimit,
		FormatQty(triangle.PairBC.PairSymbol, tradeQtyBC),
		priceBC,
	}
	orders["AC"] = &common.OrderRequest{
		triangle.PairAC.PairSymbol,
		sideAC,
		common.TypeLimit,
		FormatQty(triangle.PairAC.PairSymbol, tradeQtyAC),
		priceAC,
	}

	now := time.Now()
	profit := (newQtyA - qtyA)/qtyA

	id := triangle.Key + "_" + common.FloatToString(profit) + "_" + strconv.FormatInt(common.UnixMillis(now), 10)

	arbState := &arb.State{
		Id: id,
		QtyBefore: minOrderQtyInA,
		QtyAfter: minOrderQtyInA * (1 + profit),
		ProfitRelative: profit,
		Triangle: triangle,
		StartTs: now,
		LastUpdateTs: now,
		FrameUpdateTsQueue: make([]time.Time, 0),
		Orders: orders,
		BalanceA: balanceA,
		BalanceB: balanceB,
		BalanceC: balanceC,
		OrderQtyAB: orderQtyAB,
		OrderQtyBC: orderQtyBC,
		OrderQtyAC: orderQtyAC,
	}

	return arbState
}

func isBaseCoin(
	coinA common.Coin,
	pair *common.CoinPair) bool {
	if pair.BaseCoin.CoinSymbol == coinA.CoinSymbol {
		return true
	} else if pair.QuoteCoin.CoinSymbol == coinA.CoinSymbol{
		return false
	} else {
		panic("Incorrect coin structure")
	}
}

func simTradeWithPrice(
	qty float64,
	price float64,
	coinA common.Coin,
	pair *common.CoinPair) float64 {
	if isBaseCoin(coinA, pair) {
		return qty * price
	} else {
		return qty / price
	}
}

// given ticker with bid price and ask price,
// trades qtyA of A for B and returns qtyB
func simTradeWithTicker(
	qtyA float64,
	coinA common.Coin,
	ticker *common.Ticker,
	withFee bool) (float64, common.OrderSide, float64, float64) {
	side := common.SideSell
	if strings.HasSuffix(ticker.Symbol, coinA.CoinSymbol) {
		side = common.SideBuy
	}

	qty := 0.0
	if side == common.SideBuy {
		qty = qtyA / ticker.AskPrice
	} else {
		qty = qtyA * ticker.BidPrice
	}
	if withFee {
		qty = applyFee(qty)
	}
	if side == common.SideBuy {
		return qty, side, ticker.AskQty, ticker.AskPrice
	} else {
		return qty, side, ticker.BidQty, ticker.BidPrice
	}
}

func applyFee(qty float64) float64 {
	// TODO properly calc fee
	return qty * (1.0 - BINANCE_BNB_FEE)
}

func isTriangle(pairA, pairB, pairC *common.CoinPair) bool {
	// make sure number of coin symbols is 3 and all symbols are different
	return strings.Compare(pairA.PairSymbol, pairB.PairSymbol) != 0 &&
		strings.Compare(pairA.PairSymbol, pairC.PairSymbol) != 0 &&
		strings.Compare(pairB.PairSymbol, pairC.PairSymbol) != 0 &&
		len(getCoinSymbols(pairA, pairB, pairC)) == 3
}

func makeTriangle(pairA, pairB, pairC *common.CoinPair) (string, *arb.Triangle) {
	// only works if isTriangle == true
	coinSymbols := getCoinSymbols(pairA, pairB, pairC)
	var keys []string
	for k := range coinSymbols {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	triangleKey := ""
	for _, key := range keys {
		triangleKey += key
	}

	coinA := pairA.BaseCoin
	coinB := pairA.QuoteCoin
	coinC := pairB.BaseCoin
	if coinA == pairB.BaseCoin || coinB == pairB.BaseCoin {
		coinC = pairB.QuoteCoin
	}

	triangle := &arb.Triangle{
		PairAB: findPairForCoins(coinA, coinB, pairA, pairB, pairC),
		PairBC: findPairForCoins(coinB, coinC, pairA, pairB, pairC),
		PairAC: findPairForCoins(coinA, coinC, pairA, pairB, pairC),
		CoinA: coinA,
		CoinB: coinB,
		CoinC: coinC,
		Key: triangleKey,
	}

	return triangleKey, triangle
}

func getCoinSymbols(pairs ...*common.CoinPair) map[string]bool {
	coinSymbols := make(map[string]bool)
	for _, pair := range pairs {
		coinSymbols[pair.BaseCoin.CoinSymbol] = true
		coinSymbols[pair.QuoteCoin.CoinSymbol] = true
	}

	return coinSymbols
}

func findPairForCoins(coinA common.Coin, coinB common.Coin, pairs ...*common.CoinPair) *common.CoinPair {
	for _, pair := range pairs {
		if strings.Compare(pair.PairSymbol, coinA.CoinSymbol + coinB.CoinSymbol) == 0 ||
			strings.Compare(pair.PairSymbol, coinB.CoinSymbol + coinA.CoinSymbol) == 0 {
				return pair
		}
	}

	panic("Couldn't find coin pair")
}

func updateFrameCounters() {
	ts := time.Now()
	arbStates.Range(func(k, v interface{}) bool {
		arbState := v.(*arb.State)
		arbState.FrameUpdateTsQueue = append(arbState.FrameUpdateTsQueue, ts)
		return true
	})
}
