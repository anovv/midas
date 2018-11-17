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
					// TODO async logging
					logging.RecordArbStateMySQL(arbState)
					logging.LogLineToFile(arbState.String(), logging.ARB_STATES_FILE_PATH)
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
			if arbState != nil {
				arbStateKey := triangle.Key + "_" + common.FloatToString(arbState.ProfitRelative)
				res, loaded := arbStates.LoadOrStore(arbStateKey, arbState)
				if loaded {
					arbState := res.(*arb.State)
					arbState.LastUpdateTs = time.Now()
				}
			}
		}
	}
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
	// A->B
	qtyB, sideAB, tradeQtyAB, priceAB := simTrade(qtyA, triangle.PairAB.PairSymbol, triangle.CoinA.CoinSymbol, tickerAB)
	// B->C
	qtyC, sideBC, tradeQtyBC, priceBC := simTrade(qtyB, triangle.PairBC.PairSymbol, triangle.CoinB.CoinSymbol, tickerBC)
	// C->A
	newQtyA, sideAC, tradeQtyAC, priceAC := simTrade(qtyC, triangle.PairAC.PairSymbol, triangle.CoinC.CoinSymbol, tickerAC)

	if newQtyA <= qtyA {
		// No arb
		return nil
	}

	// Find max tradable qty equivalent in A
	balanceBinA := 0.0
	if sideAB == common.SideSell {
		balanceBinA = balanceB * priceAB
	} else {
		balanceBinA = balanceB / priceAB
	}

	balanceCinA := 0.0
	if sideAC == common.SideSell {
		balanceCinA = balanceC * priceAC
	} else {
		balanceCinA = balanceC / priceAC
	}

	tradeQtyABinA := 0.0
	if sideAB == common.SideSell {
		tradeQtyABinA = tradeQtyAB * priceAB
	} else {
		tradeQtyABinA = tradeQtyAB / priceAB
	}

	tradeQtyACinA := 0.0
	if sideAC == common.SideSell {
		tradeQtyACinA = tradeQtyAC * priceAC
	} else {
		tradeQtyACinA = tradeQtyAC / priceAC
	}

	tradeQtyBCinA := 0.0
	if sideBC == common.SideSell {
		if sideAB == common.SideSell {
			tradeQtyBCinA = tradeQtyBC * priceAB
		} else {
			tradeQtyBCinA = tradeQtyBC / priceAB
		}
	} else {
		if sideAC == common.SideSell {
			tradeQtyBCinA = tradeQtyBC * priceAC
		} else {
			tradeQtyBCinA = tradeQtyBC / priceAC
		}
	}

	minBalanceInA := math.Min(math.Min(balanceBinA, balanceCinA), balanceA)
	minTradeQtyInA := math.Min(math.Min(tradeQtyABinA, tradeQtyACinA), tradeQtyBCinA)

	minTradeQtyInA = math.Min(minBalanceInA, minTradeQtyInA)

	// Convert this qty back for each coin
	if sideAB == common.SideSell {
		tradeQtyAB = minTradeQtyInA
	} else {
		tradeQtyAB = minTradeQtyInA * priceAB
	}

	if sideAC == common.SideSell {
		tradeQtyAC = minTradeQtyInA
	} else {
		tradeQtyAC = minTradeQtyInA * priceAC
	}

	if sideBC == common.SideSell {
		if sideAB == common.SideSell {
			tradeQtyBC = minTradeQtyInA / priceAB
		} else {
			tradeQtyBC = minTradeQtyInA * priceAB
		}
	} else {
		if sideAC == common.SideSell {
			tradeQtyBC = minTradeQtyInA / priceAC
		} else {
			tradeQtyBC = minTradeQtyInA * priceAC
		}
	}

	orders := make(map[string]*common.OrderRequest)
	orders["AB"] = &common.OrderRequest{
		triangle.PairAB.PairSymbol,
		sideAB,
		common.TypeLimit,
		tradeQtyAB,
		priceAB,
	}
	orders["BC"] = &common.OrderRequest{
		triangle.PairBC.PairSymbol,
		sideBC,
		common.TypeLimit,
		tradeQtyBC,
		priceBC,
	}
	orders["AC"] = &common.OrderRequest{
		triangle.PairAC.PairSymbol,
		sideAC,
		common.TypeLimit,
		tradeQtyAC,
		priceAC,
	}

	now := time.Now()
	profit := (newQtyA - qtyA)/qtyA

	arbState := &arb.State{
		QtyBefore: minTradeQtyInA,
		QtyAfter: minTradeQtyInA * (1 + profit),
		ProfitRelative: profit,
		Triangle: triangle,
		StartTs: now,
		LastUpdateTs: now,
		FrameUpdateTsQueue: make([]time.Time, 0),
		Orders: orders,
	}

	return arbState
}

// given rate B/A with bid price (or A/B with ask price),
// trades qtyA of A for B and returns qtyB
func simTrade(
	qtyA float64,
	pairSymbol string,
	coinASymbol string,
	ticker *common.Ticker) (float64, common.OrderSide, float64, float64) {
	side := common.SideSell
	if strings.HasSuffix(pairSymbol, coinASymbol) {
		side = common.SideBuy
	}

	if side == common.SideBuy {
		return applyFee(qtyA * ticker.BidPrice), side, ticker.BidQty, ticker.BidPrice
	} else {
		return applyFee(qtyA / ticker.AskPrice), side, ticker.AskQty, ticker.AskPrice
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
