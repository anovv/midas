package binance

import (
	"errors"
	"fmt"
	"log"
	"midas/network"
	"midas/common"
	"strconv"
	"encoding/json"
	"net/url"
)

const (
	API_BASE_URL = "https://api.binance.com/"
	API_V1       = API_BASE_URL + "api/v1/"
	API_V3       = API_BASE_URL + "api/v3/"

	TICKER_URI             = "ticker/24hr?symbol=%s"
	TICKERS_URI            = "ticker/allBookTickers"
	DEPTH_URI              = "depth?symbol=%s&limit=%d"
	USER_DATA_STREAM_URI   = "userDataStream"
	ACCOUNT_URI 		   = "account"

	MIN_DEPTH = 5
	MAX_DEPTH = 100
)

// TODO merge with GetAllTickers
func GetAllPairs() ([]*common.CoinPair, error) {
	tickersUri := API_V1 + TICKERS_URI
	respData, err := network.NewHttpRequest(
		"GET",
		tickersUri,
		nil,
		false,
		false)
	if err != nil {
		log.Println("GetAllPairs error:", err)
		return nil, err
	}

	var tickerList []interface{}
	err = json.Unmarshal(respData, &tickerList)
	if err != nil {
		log.Println("GetAllPairs error:", err)
		return nil, err
	}

	var pairs []*common.CoinPair

	for _, tickerInterface := range tickerList {
		tickerMap := tickerInterface.(map[string]interface {})
		pairSymbolInterface := tickerMap["symbol"]
		pairSymbol := pairSymbolInterface.(string)
		pair := common.SymbolToPair(pairSymbol)
		if pair != nil {
			pairs = append(pairs, pair)
		} else {
			log.Println("Unable to parse symbol: " + pairSymbol)
		}
	}

	return pairs, nil
}

func GetAllTickers() (*common.TickersMap, error) {
	tickersUri := API_V1 + TICKERS_URI
	respData, err := network.NewHttpRequest(
		"GET",
		tickersUri,
		nil,
		false,
		false)
	if err != nil {
		log.Println("GetAllTickers error:", err)
		return nil, err
	}

	var tickerList []interface{}
	err = json.Unmarshal(respData, &tickerList)

	if err != nil {
		log.Println("GetAllTickers error:", err)
		return nil, err
	}

	tickers := make(common.TickersMap)

	for _, tickerInterface := range tickerList {
		tickerMap := tickerInterface.(map[string]interface {})
		pairSymbol := tickerMap["symbol"].(string)
		bidPrice, _ := strconv.ParseFloat(tickerMap["bidPrice"].(string), 64)
		askPrice, _ := strconv.ParseFloat(tickerMap["askPrice"].(string), 64)
		bidQty, _ := strconv.ParseFloat(tickerMap["bidQty"].(string), 64)
		askQty, _ := strconv.ParseFloat(tickerMap["askQty"].(string), 64)
		tickers[pairSymbol] = &common.Ticker{
			Symbol: pairSymbol,
			BidPrice: bidPrice,
			AskPrice: askPrice,
			BidQty: bidQty,
			AskQnty: askQty,
		}
	}

	return &tickers, nil
}

func GetDepth(size int, currencyPair string) (*common.Depth, error) {
	if size > MAX_DEPTH {
		size = MAX_DEPTH
	} else if size < MIN_DEPTH {
		size = MIN_DEPTH
	}

	apiUrl := fmt.Sprintf(API_V1+DEPTH_URI, currencyPair, size)

	respData, err := network.NewHttpRequest(
		"GET",
		apiUrl,
		nil,
		false,
		false)
	if err != nil {
		log.Println("GetDepth error:", err)
		return nil, err
	}

	var resp map[string]interface{}
	err = json.Unmarshal(respData, &resp)

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

	depth := new(common.Depth)

	depth.LastUpdateId = lastUpdateId

	for _, bid := range bids {
		_bid := bid.([]interface{})
		amount := common.ToFloat64(_bid[1])
		price := common.ToFloat64(_bid[0])
		dr := common.DepthRecord{Amount: amount, Price: price}
		depth.BidList = append(depth.BidList, dr)
	}

	for _, ask := range asks {
		_ask := ask.([]interface{})
		amount := common.ToFloat64(_ask[1])
		price := common.ToFloat64(_ask[0])
		dr := common.DepthRecord{Amount: amount, Price: price}
		depth.AskList = append(depth.AskList, dr)
	}

	return depth, nil
}

func GetUserDataStreamListenKey() (*string, error) {
	uri := API_V1 + USER_DATA_STREAM_URI
	respData, err := network.NewHttpRequest(
		"POST",
		uri,
		nil,
		true,
		false)
	if err != nil {
		log.Println("GetUserDataStreamListenKey error:", err)
		return nil, err
	}

	var resp map[string]*string
	err = json.Unmarshal(respData, &resp)

	if err != nil {
		log.Println("GetUserDataStreamListenKey error:", err)
		return nil, err
	}

	return resp["listenKey"], nil
}

func PingUserDataStream(listenKey *string) bool {
	uri := API_V1 + USER_DATA_STREAM_URI
	reqData := url.Values{}
	reqData.Set("listenKey", *listenKey)
	_, err := network.NewHttpRequest(
		"PUT",
		uri,
		reqData,
		true,
		false)
	if err != nil {
		log.Println("PingUserDataStream error:", err)
		return false
	}

	return true
}

func CloseUserDataStream(listenKey *string) bool {
	uri := API_V1 + USER_DATA_STREAM_URI
	reqData := url.Values{}
	reqData.Set("listenKey", *listenKey)
	_, err := network.NewHttpRequest(
		"DELETE",
		uri,
		reqData,
		true,
		false)
	if err != nil {
		log.Println("CloseUserDataStream error:", err)
		return false
	}

	return true
}

func GetAccount() (*common.Account, error) {
	accountUri := API_V3 + ACCOUNT_URI
	res, err := network.NewHttpRequest(
		"GET",
		accountUri,
		nil,
		true,
		true)
	if err != nil {
		log.Println("GetAccount error:", err)
		return nil, err
	}

	rawAccount := struct {
		MakerCommission   int64 `json:"makerCommision"`
		TakerCommission  int64 `json:"takerCommission"`
		BuyerCommission  int64 `json:"buyerCommission"`
		SellerCommission int64 `json:"sellerCommission"`
		UpdateTime		 float64 `json:"updateTime"`
		Balances         []struct {
			Asset  string `json:"asset"`
			Free   string `json:"free"`
			Locked string `json:"locked"`
		}
	}{}
	if err := json.Unmarshal(res, &rawAccount); err != nil {
		return nil, err
	}

	acc := &common.Account{
		MakerCommission:  rawAccount.MakerCommission,
		TakerCommission:  rawAccount.TakerCommission,
		BuyerCommission:  rawAccount.BuyerCommission,
		SellerCommission: rawAccount.SellerCommission,
		LastUpdateTs:	 common.TimeFromUnixTimestampFloat(rawAccount.UpdateTime),
		Balances: make([]*common.Balance, 0),
	}
	for _, b := range rawAccount.Balances {
		f := common.ToFloat64(b.Free)
		l := common.ToFloat64(b.Locked)

		acc.Balances = append(acc.Balances, &common.Balance{
			Coin:  common.Coin{
				CoinSymbol: b.Asset,
			},
			Free:   f,
			Locked: l,
		})
	}

	return acc, nil
}
