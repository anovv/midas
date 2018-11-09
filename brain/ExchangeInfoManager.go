package brain

import (
	"midas/common"
	"midas/apis/binance"
	"time"
	"log"
)

var exchangeInfo *common.ExchangeInfo
var allPairs []*common.CoinPair
var filtersMap *common.FiltersMap

const EXCHANGE_INFO_UPDATE_PERIOD_MIN = 1

func RunUpdateExchangeInfo() {
	updateExchangeInfo()
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
		allPairs = exchangeInfo.GetAllPairs()
		filtersMap = exchangeInfo.GetFiltersMap()
		log.Println("Updating exchange info... Done")
	}
}