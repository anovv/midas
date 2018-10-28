package brain

import (
	"midas/apis/binance"
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

var triangles = make(map[string]*arb.Triangle)
var pairs = make(map[string]*common.CoinPair)

var arbStates = sync.Map{}

var brainConfig = configuration.ReadBrainConfig()

func RunArbDetector() {
	initArbDetector()
	runReportArb()
	runDetectArbBLOCKING()
}

func initArbDetector() {
	log.Println("Initializing arb detector...")
	logging.CreateTableIfNotExistsMySQL()
	_pairs, err := binance.GetAllPairs()
	if err != nil {
		panic("Can't fetch list of pairs")
	}

	// memoize
	for _, pair := range _pairs {
		pairs[pair.PairSymbol] = pair
	}

	log.Println("Analyzing " + strconv.Itoa(len(_pairs)) + " pairs...")
	tStart := time.Now()

	for _, pairA := range _pairs {
		for _,  pairB := range _pairs {
			for _,  pairC := range _pairs {
				if isTriangle(pairA, pairB, pairC) {
					key, triangle := makeTriangle(pairA, pairB, pairC)
					if triangles[key] == nil {
						triangles[key] = triangle
					}
				}
			}
		}
	}

	delta := time.Since(tStart)
	log.Println("Initializing finished in " + delta.String())
	// TODO print number of triangles
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
		for _, triangle := range triangles {
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

	coinA := pairA.CoinA
	coinB := pairA.CoinB
	coinC := pairB.CoinA
	if coinA == pairB.CoinA || coinB == pairB.CoinA {
		coinC = pairB.CoinB
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
		coinSymbols[pair.CoinA.CoinSymbol] = true
		coinSymbols[pair.CoinB.CoinSymbol] = true
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
