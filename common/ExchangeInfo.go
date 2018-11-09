package common

import "strings"

type ExchangeInfo struct {
	Timezone string `json:"timezone"`
	ServerTime int64 `json:"serverTime"`
	RateLimits []struct {
		RateLimitType string `json:"rateLimitType"`
		Interval string `json:"interval"`
		Limit int64 `json:"limit"`
	} `json:"rateLimits"`
	Symbols []struct {
		Symbol string `json:"symbol"`
		Status string `json:"status"`
		BaseAsset string `json:"baseAsset"`
		BaseAssetPrecision int64 `json:"baseAssetPrecision"`
		QuoteAsset string `json:"quoteAsset"`
		QuoteAssetPrecision int64 `json:"quotePrecision"`
		OrderTypes []string `json:"orderTypes"`
		IcebergAllowed bool `json:"icebergAllowed"`
		Filters []Filter `json:"filters"`
	} `json:"symbols"`
}

// symbol->[]{filter_key-->filter_value}
type FiltersMap map[string][]Filter
type Filter map[string]interface{}

const (
	FILTER_TYPE_PRICE_FILTER = "PRICE_FILTER"
	FILTER_TYPE_LOT_SIZE = "LOT_SIZE"
	FILTER_TYPE_MIN_NOTIONAL = "MIN_NOTIONAL"
	FILTER_TYPE_ICEBERG_PARTS = "ICEBERG_PARTS"
	FILTER_TYPE_MAX_NUM_ALGO_ORDERS = "MAX_NUM_ALGO_ORDERS"
	FILTER_TYPE_MAX_NUM_ORDERS = "MAX_NUM_ORDERS"
	FILTER_TYPE_EXCHANGE_MAX_NUM_ORDERS = "EXCHANGE_MAX_NUM_ORDERS"
	FILTER_TYPE_EXCHANGE_MAX_NUM_ALGO_ORDERS = "EXCHANGE_MAX_NUM_ALGO_ORDERS"
)

func (exchangeInfo *ExchangeInfo) GetAllPairs() []*CoinPair {
	var pairs []*CoinPair

	for _, symbol := range exchangeInfo.Symbols {
		pair := &CoinPair{
			symbol.Symbol,
			Coin{
				symbol.BaseAsset,
			},
			Coin{
				symbol.QuoteAsset,
			},
		}
		pairs = append(pairs, pair)
	}

	return pairs
}

func (exchangeInfo *ExchangeInfo) GetFiltersMap() *FiltersMap {
	filtersMap := make(FiltersMap)
	for _, symbol := range exchangeInfo.Symbols {
		filtersMap[symbol.Symbol] = symbol.Filters
	}

	return &filtersMap
}

func (filtersMap *FiltersMap) GetMinPrice(symbol string) float64 {
	filter := filtersMap.GetFilter(symbol, FILTER_TYPE_PRICE_FILTER)
	return ToFloat64((*filter)["minPrice"])
}

func (filtersMap *FiltersMap) GetMaxPrice(symbol string) float64 {
	filter := filtersMap.GetFilter(symbol, FILTER_TYPE_PRICE_FILTER)
	return ToFloat64((*filter)["maxPrice"])
}

func (filtersMap *FiltersMap) GetTickSize(symbol string) float64 {
	filter := filtersMap.GetFilter(symbol, FILTER_TYPE_PRICE_FILTER)
	return ToFloat64((*filter)["tickSize"])
}

func (filtersMap *FiltersMap) GetMinQty(symbol string) float64 {
	filter := filtersMap.GetFilter(symbol, FILTER_TYPE_LOT_SIZE)
	return ToFloat64((*filter)["minQty"])
}

func (filtersMap *FiltersMap) GetMaxQty(symbol string) float64 {
	filter := filtersMap.GetFilter(symbol, FILTER_TYPE_LOT_SIZE)
	return ToFloat64((*filter)["maxQty"])
}

func (filtersMap *FiltersMap) GetStepSize(symbol string) float64 {
	filter := filtersMap.GetFilter(symbol, FILTER_TYPE_LOT_SIZE)
	return ToFloat64((*filter)["stepSize"])
}

func (filtersMap *FiltersMap) GetMinNotional(symbol string) float64 {
	filter := filtersMap.GetFilter(symbol, FILTER_TYPE_MIN_NOTIONAL)
	return ToFloat64((*filter)["minNotional"])
}

func (filtersMap *FiltersMap) GetFilter(symbol string, filterName string) *Filter {
	filtersList := (*filtersMap)[symbol]
	for _, filter := range filtersList {
		if strings.Compare(filter["filterType"].(string), filterName) == 0 {
			return &filter
		}
	}

	panic("Filter " + filterName + " for " + symbol + " does not exist")
}