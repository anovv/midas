package brain

import (
	"midas/apis/binance"
	. "midas/common"
	"net/http"
	"sort"
	"strconv"
	"time"
	"strings"
	"log"
	"sync"
	"midas/logging"
	"midas/configuration"
)

type Triangle struct {
	PairAB *CoinPair
	PairBC *CoinPair
	PairAC *CoinPair
	CoinA Coin
	CoinB Coin
	CoinC Coin
	Key string
}

type ArbState struct {
	QtyBefore float64
	QtyAfter float64
	ProfitRelative float64
	Triangle *Triangle
	StartTs time.Time
	LastUpdateTs time.Time
	Reported bool
}

var api = binance.New(http.DefaultClient, "", "")
var triangles = make(map[string]*Triangle)
var pairs = make(map[string]*CoinPair)

// No need for mutex as we simply update this variable with a new map instance on each write
var tickers = make(map[string]*Ticker)

// TODO use syncMap?
var arbStates = make(map[string]*ArbState)
var arbStatesMutex = &sync.RWMutex{}

var brainConfig = configuration.ReadBrainConfig()

func RunArbDetector() {
	initArbDetector()
	runTickerUpdates()
	runReportArb()
	runDetectArbBLOCKING()
	//time.Sleep(time.Duration(25000000*1000*1000) * time.Microsecond)
}

func initArbDetector() {
	log.Println("Initializing arb detector...")
	_pairs, err := api.GetAllPairs()
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
	printTriangleForSymbols()
	log.Println("Initializing finished in " + delta.String())
	logging.LogLineToFile("Launched at " + time.Now().String())
}

func runTickerUpdates() {
	log.Println("Running tickers updates...")
	go func() {
		for {
			tickers, _ = api.GetAllTickers() // weight is 40
			if len(tickers) == 0 {
				log.Println("Failed to fetch tickers")
			}
			//log.Println("Updated " + strconv.Itoa(len(tickers)) + " tickers")
			time.Sleep(time.Duration(brainConfig.TICKERS_UPDATE_PERIOD_MICROS) * time.Microsecond)
		}
	}()
}

func runReportArb() {
	// Goes through all arb states and prints unreported
	go func() {
		log.Println("Checking existing arbs...")
		for {
			arbStatesMutex.Lock()
			for _, arbState := range arbStates {
				if arbState.Reported {
					continue
				}

				// If arb state was not updated by detector routine for more than ARB_REPORT_UPDATE_THRESHOLD_MICROS
				// we consider arb opportunity is gone
				if time.Since(arbState.LastUpdateTs) > time.Duration(brainConfig.ARB_REPORT_UPDATE_THRESHOLD_MICROS) * time.Microsecond {
					arbState.Reported = true
					logging.LogLineToFile("Found arb opportunity: " +
						arbState.Triangle.CoinA.CoinSymbol + "->" +
							arbState.Triangle.CoinB.CoinSymbol + "->" +
								arbState.Triangle.CoinC.CoinSymbol + "->" +
									arbState.Triangle.CoinA.CoinSymbol +
										" Before: " + FloatToString(arbState.QtyBefore) + arbState.Triangle.CoinA.CoinSymbol +
											" After: " + FloatToString(arbState.QtyAfter) + arbState.Triangle.CoinA.CoinSymbol +
												" Relative Profit: " + FloatToString(arbState.ProfitRelative * 100.0) + "%" +
													" Lasted for " + arbState.LastUpdateTs.Sub(arbState.StartTs).String() +
														" Started at " + arbState.StartTs.String())
				}
			}
			arbStatesMutex.Unlock()
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
				// found arbitrage
				//	triangle.CoinA.CoinSymbol + "->" +
				//		triangle.CoinB.CoinSymbol + "->" +
				//			triangle.CoinC.CoinSymbol + "->" +
				//				triangle.CoinA.CoinSymbol +
				//					" Before: " + FloatToString(qtyA) + triangle.CoinA.CoinSymbol +
				//						" After: " + FloatToString(newQtyA) + triangle.CoinA.CoinSymbol +
				//							" Profit: " + FloatToString(profit))
				arbStateKey := triangle.Key + "_" + FloatToString(profit)

				if arbStates[arbStateKey] == nil {
					now := time.Now()
					arbStatesMutex.RLock()
					arbStates[arbStateKey] = &ArbState{
						QtyBefore: qtyA,
						QtyAfter: newQtyA,
						ProfitRelative: profit,
						Triangle: triangle,
						StartTs: now,
						LastUpdateTs: now,
						Reported: false,
					}
				} else {
					arbStatesMutex.RLock()
					arbStates[arbStateKey].LastUpdateTs = time.Now()
				}
				arbStatesMutex.RUnlock()
			}
		}
	}
}

// given rate B/A with bid price (or A/B with ask price),
// trades qtyA of A for B and returns qtyB
func simTrade(qtyA float64, pairSymbol string, coinASymbol string) (bool, float64) {
	buyA := false
	if strings.HasSuffix(pairSymbol, coinASymbol) {
		buyA = true
	}

	fee := 0.001
	if strings.Contains(pairSymbol, "BNB") {
		fee = 0.0005
	}

	ticker := tickers[pairSymbol]

	if ticker == nil {
		return false, 0
	}

	if buyA {
		return true, (qtyA * ticker.BidPrice) * (1.0 - fee)
	} else {
		return true, (qtyA / ticker.AskPrice) * (1.0 - fee)
	}
}

func isTriangle(pairA, pairB, pairC *CoinPair) bool {
	// make sure number of coin symbols is 3 and all symbols are different
	return strings.Compare(pairA.PairSymbol, pairB.PairSymbol) != 0 &&
		strings.Compare(pairA.PairSymbol, pairC.PairSymbol) != 0 &&
		strings.Compare(pairB.PairSymbol, pairC.PairSymbol) != 0 &&
		len(getCoinSymbols(pairA, pairB, pairC)) == 3
}

func makeTriangle(pairA, pairB, pairC *CoinPair) (string, *Triangle) {
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

	triangle := &Triangle{
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

func getCoinSymbols(pairs ...*CoinPair) map[string]bool {
	coinSymbols := make(map[string]bool)
	for _, pair := range pairs {
		coinSymbols[pair.CoinA.CoinSymbol] = true
		coinSymbols[pair.CoinB.CoinSymbol] = true
	}

	return coinSymbols
}

func findPairForCoins(coinA Coin, coinB Coin, pairs ...*CoinPair) *CoinPair {
	for _, pair := range pairs {
		if strings.Compare(pair.PairSymbol, coinA.CoinSymbol + coinB.CoinSymbol) == 0 ||
			strings.Compare(pair.PairSymbol, coinB.CoinSymbol + coinA.CoinSymbol) == 0 {
				return pair
		}
	}

	panic("Couldn't find coin pair")
}

func printTriangleForSymbols (symbols ...string) {
	for key, triangle := range triangles {
		keyContainsSymbols := true
		for _, symbol := range symbols {
			keyContainsSymbols = keyContainsSymbols && strings.Contains(key, symbol)
		}
		if keyContainsSymbols {
			log.Println(
				"Key: " + key +
					" C1: " + triangle.CoinA.CoinSymbol +
						" C2: " + triangle.CoinB.CoinSymbol +
							" C3: " + triangle.CoinC.CoinSymbol +
								" Triangle: " + triangle.PairAB.PairSymbol +
									"->" + triangle.PairBC.PairSymbol +
										"->" + triangle.PairAC.PairSymbol)
		}
	}
}
