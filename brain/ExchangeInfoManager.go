package brain

import (
	"midas/common"
	"midas/apis/binance"
	"time"
	"log"
	"math"
	"strings"
)

var exchangeInfo *common.ExchangeInfo
var allPairs []*common.CoinPair
var filtersMap *common.FiltersMap

const EXCHANGE_INFO_UPDATE_PERIOD_MIN = 1

func RunUpdateExchangeInfo() {
	updateExchangeInfo()
	initTickersMap()
	go func() {
		for {
			time.Sleep(time.Duration(EXCHANGE_INFO_UPDATE_PERIOD_MIN) * time.Minute)
			updateExchangeInfo()
		}
	}()
}

func updateExchangeInfo() {
	log.Println("Updating exchange info...")
	info, _ := binance.GetExchangeInfo()
	if info != nil {
		exchangeInfo = info
		allPairs = getAllPairs()
		filtersMap = getFiltersMap()
		log.Println("Updating exchange info... Done")
	}
}
func getAllPairs() []*common.CoinPair {
	var pairs []*common.CoinPair

	for _, symbol := range exchangeInfo.Symbols {
		// TODO dynamically add newly detected pairs
		if strings.Compare(symbol.Status, "TRADING") != 0 {
			continue
		}
		pair := &common.CoinPair{
			symbol.Symbol,
			common.Coin{
				symbol.BaseAsset,
			},
			common.Coin{
				symbol.QuoteAsset,
			},
		}
		pairs = append(pairs, pair)
	}

	return pairs
}

func getFiltersMap() *common.FiltersMap {
	filtersMap := make(common.FiltersMap)
	for _, symbol := range exchangeInfo.Symbols {
		filtersMap[symbol.Symbol] = symbol.Filters
	}

	return &filtersMap
}

func initTickersMap() {
	tickers, err := binance.GetAllTickers()
	if err != nil {
		panic("Unable to init tickers map: " + err.Error())
	}

	tickersMap = tickers
}

func GetMinPrice(symbol string) float64 {
	filter := GetFilter(symbol, common.FILTER_TYPE_PRICE_FILTER)
	return common.ToFloat64((*filter)["minPrice"])
}

func GetMaxPrice(symbol string) float64 {
	filter := GetFilter(symbol, common.FILTER_TYPE_PRICE_FILTER)
	return common.ToFloat64((*filter)["maxPrice"])
}

func GetTickSize(symbol string) float64 {
	filter := GetFilter(symbol, common.FILTER_TYPE_PRICE_FILTER)
	return common.ToFloat64((*filter)["tickSize"])
}

func GetMinQty(symbol string) float64 {
	filter := GetFilter(symbol, common.FILTER_TYPE_LOT_SIZE)
	return common.ToFloat64((*filter)["minQty"])
}

func GetMaxQty(symbol string) float64 {
	filter := GetFilter(symbol, common.FILTER_TYPE_LOT_SIZE)
	return common.ToFloat64((*filter)["maxQty"])
}

func GetStepSize(symbol string) float64 {
	filter := GetFilter(symbol, common.FILTER_TYPE_LOT_SIZE)
	return common.ToFloat64((*filter)["stepSize"])
}

func GetMinNotional(symbol string) float64 {
	filter := GetFilter(symbol, common.FILTER_TYPE_MIN_NOTIONAL)
	return common.ToFloat64((*filter)["minNotional"])
}

func GetMarketNotional(symbol string, qty float64) {

}

func FilterCheck(symbol string, qty float64, price float64) common.FilterCheck {
	if price != 0 && price < GetMinPrice(symbol) {
		return common.FilterCheckMinPrice
	}

	if price != 0 && price > GetMaxPrice(symbol) {
		return common.FilterCheckMaxPrice
	}

	if price != 0 && math.Mod(price - GetMinPrice(symbol), GetTickSize(symbol)) != 0 {
		return common.FilterCheckTickSize
	}

	if price != 0 && price * qty < GetMinNotional(symbol) {
		return common.FilterCheckMinNotional
	}

	if qty < GetMinQty(symbol) {
		return common.FilterCheckMinQty
	}

	if qty > GetMaxQty(symbol) {
		return common.FilterCheckMaxQty
	}

	if math.Mod(qty - GetMinQty(symbol), GetStepSize(symbol)) != 0 {
		return common.FilterCheckStepSize
	}

	return common.FilterCheckOk
}

// Rounds down qty based on stepSize
func FormatQty(symbol string, qty float64) float64 {
	stepSize := GetStepSize(symbol)
	factor := 1.0
	for stepSize != 1.0 {
		factor = factor * 10.0
		stepSize = stepSize * 10.0
	}

	qty = qty * factor
	qty = math.Floor(qty)

	return qty/factor
}

func GetFilter(symbol string, filterName string) *common.Filter {
	filtersList := (*filtersMap)[symbol]
	for _, filter := range filtersList {
		if strings.Compare(filter["filterType"].(string), filterName) == 0 {
			return &filter
		}
	}

	panic("Filter " + filterName + " for " + symbol + " does not exist")
}

func HasPair(symbol string) bool {
	if allPairs == nil {
		panic("Pairs are not fetched")
	}

	for _, pair := range allPairs {
		if strings.Compare(pair.PairSymbol, symbol) == 0 {
			return true
		}
	}

	return false
}
