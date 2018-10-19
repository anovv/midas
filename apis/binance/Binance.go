package binance

import (
	"errors"
	"fmt"
	"log"
	"net/http"
	. "midas/network"
	. "midas/common"
	"strconv"
)

const (
	API_BASE_URL = "https://api.binance.com/"
	API_V1       = API_BASE_URL + "api/v1/"

	TICKER_URI             = "ticker/24hr?symbol=%s"
	TICKERS_URI            = "ticker/allBookTickers"
	DEPTH_URI              = "depth?symbol=%s&limit=%d"

	MIN_DEPTH = 5
	MAX_DEPTH = 100
)

type Binance struct {
	accessKey,
	secretKey string
	httpClient *http.Client
}

func New(client *http.Client, api_key, secret_key string) *Binance {
	return &Binance{api_key, secret_key, client}
}

func (bn *Binance) GetAllPairs() ([]*CoinPair, error) {
	tickersUri := API_V1 + TICKERS_URI
	tickerList, err := HttpGetList(bn.httpClient, tickersUri)

	if err != nil {
		log.Println("GetAllPairs error:", err)
		return nil, err
	}

	var pairs []*CoinPair

	for _, tickerInterface := range tickerList {
		tickerMap := tickerInterface.(map[string]interface {})
		pairSymbolInterface := tickerMap["symbol"]
		pairSymbol := pairSymbolInterface.(string)
		pair := SymbolToPair(pairSymbol)
		if pair != nil {
			pairs = append(pairs, pair)
		} else {
			log.Println("Unable to parse symbol: " + pairSymbol)
		}
	}

	return pairs, nil
}

func (bn *Binance) GetAllTickers() (*TickersMap, error) {
	tickersUri := API_V1 + TICKERS_URI
	tickerList, err := HttpGetList(bn.httpClient, tickersUri)

	if err != nil {
		log.Println("GetAllTickers error:", err)
		return nil, err
	}

	tickers := make(TickersMap)

	for _, tickerInterface := range tickerList {
		tickerMap := tickerInterface.(map[string]interface {})
		pairSymbol := tickerMap["symbol"].(string)
		bidPrice, _ := strconv.ParseFloat(tickerMap["bidPrice"].(string), 64)
		askPrice, _ := strconv.ParseFloat(tickerMap["askPrice"].(string), 64)
		bidQty, _ := strconv.ParseFloat(tickerMap["bidQty"].(string), 64)
		askQty, _ := strconv.ParseFloat(tickerMap["askQty"].(string), 64)
		tickers[pairSymbol] = &Ticker{
			Symbol: pairSymbol,
			BidPrice: bidPrice,
			AskPrice: askPrice,
			BidQty: bidQty,
			AskQnty: askQty,
		}
	}

	return &tickers, nil
}

func (bn *Binance) GetDepth(size int, currencyPair string) (*Depth, error) {
	if size > MAX_DEPTH {
		size = MAX_DEPTH
	} else if size < MIN_DEPTH {
		size = MIN_DEPTH
	}

	apiUrl := fmt.Sprintf(API_V1+DEPTH_URI, currencyPair, size)
	log.Println("API Url:", apiUrl)
	resp, err := HttpGetMap(bn.httpClient, apiUrl)
	if err != nil {
		log.Println("GetDepth error:", err)
		return nil, err
	}

	if _, isok := resp["code"]; isok {
		return nil, errors.New(resp["msg"].(string))
	}

	lastUpdateId := resp["lastUpdateId"].(float64)
	bids := resp["bids"].([]interface{})
	asks := resp["asks"].([]interface{})

	depth := new(Depth)

	depth.LastUpdateId = lastUpdateId

	for _, bid := range bids {
		_bid := bid.([]interface{})
		amount := ToFloat64(_bid[1])
		price := ToFloat64(_bid[0])
		dr := DepthRecord{Amount: amount, Price: price}
		depth.BidList = append(depth.BidList, dr)
	}

	for _, ask := range asks {
		_ask := ask.([]interface{})
		amount := ToFloat64(_ask[1])
		price := ToFloat64(_ask[0])
		dr := DepthRecord{Amount: amount, Price: price}
		depth.AskList = append(depth.AskList, dr)
	}

	return depth, nil
}
