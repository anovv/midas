package common

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

type FilterCheck string

var(
	FilterCheckOk = FilterCheck("OK")
	FilterCheckMinPrice = FilterCheck("MIN_PRICE")
	FilterCheckMaxPrice = FilterCheck("MAX_PRICE")
	FilterCheckTickSize = FilterCheck("TICK_SIZE")
	FilterCheckMinQty = FilterCheck("MIN_QTY")
	FilterCheckMaxQty = FilterCheck("MAX_QTY")
	FilterCheckStepSize = FilterCheck("STEP_SIZE")
	FilterCheckMinNotional = FilterCheck("MIN_NOTIONAL")
)