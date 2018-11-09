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
)

const (
	BINANCE_DEFAULT_FEE = 0.001
	BINANCE_BNB_FEE = 0.00075
)

var arbTriangles = make(map[string]*arb.Triangle)
var arbCoins = make(map[common.Coin]bool)
var arbPairs = make(map[common.CoinPair]bool)

var arbStates = sync.Map{}

var brainConfig = configuration.ReadBrainConfig()

func RunArbDetector() {
	InitArbDetector()
	runReportArb()
	runDetectArbBLOCKING()
}

func InitArbDetector() {
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
					arbCoins[triangle.CoinA] = true
					arbCoins[triangle.CoinB] = true
					arbCoins[triangle.CoinC] = true

					// Record arb pairs
					arbPairs[*triangle.PairAB] = true
					arbPairs[*triangle.PairBC] = true
					arbPairs[*triangle.PairAC] = true
				}
			}
		}
	}

	delta := time.Since(tStart)
	log.Println("Initializing finished in " + delta.String())
	log.Println("Arb triangles: " + strconv.Itoa(len(arbTriangles)))
	log.Println("Arb pairs: " + strconv.Itoa(len(arbPairs)))
	log.Println("Arb coins: " + strconv.Itoa(len(arbCoins)))
	logging.LogLineToFile("Launched at " + time.Now().String())
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
			qtyA := 1.0
			// A->B
			sucB, qtyB := simTrade(qtyA, triangle.PairAB.PairSymbol, triangle.CoinA.CoinSymbol)
			// B->C
			sucC, qtyC := simTrade(qtyB, triangle.PairBC.PairSymbol, triangle.CoinB.CoinSymbol)
			// C->A
			sucA, newQtyA := simTrade(qtyC, triangle.PairAC.PairSymbol, triangle.CoinC.CoinSymbol)

			if !sucA || !sucB || !sucC {
				continue
			}

			profit := (newQtyA - qtyA)/qtyA

			if newQtyA > qtyA {
				arbStateKey := triangle.Key + "_" + common.FloatToString(profit)
				now := time.Now()

				res, loaded := arbStates.LoadOrStore(arbStateKey, &arb.State{
					QtyBefore: qtyA,
					QtyAfter: newQtyA,
					ProfitRelative: profit,
					Triangle: triangle,
					StartTs: now,
					LastUpdateTs: now,
					FrameUpdateTsQueue: make([]time.Time, 0),
				})
				if loaded {
					arbState := res.(*arb.State)
					arbState.LastUpdateTs = time.Now()
				}
			}
		}
	}
}

// given rate B/A with bid price (or A/B with ask price),
// trades qtyA of A for B and returns qtyB
func simTrade(qtyA float64, pairSymbol string, coinASymbol string) (bool, float64) {
	if tickersMap == nil {
		return false, 0
	}

	buyA := false
	if strings.HasSuffix(pairSymbol, coinASymbol) {
		buyA = true
	}

	fee := BINANCE_DEFAULT_FEE
	if strings.Contains(pairSymbol, "BNB") {
		fee = BINANCE_BNB_FEE
	}

	ticker := (*tickersMap)[pairSymbol]

	if ticker == nil {
		return false, 0
	}

	if buyA {
		return true, (qtyA * ticker.BidPrice) * (1.0 - fee)
	} else {
		return true, (qtyA / ticker.AskPrice) * (1.0 - fee)
	}
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
