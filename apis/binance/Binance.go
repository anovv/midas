package binance

import (
	"errors"
	"fmt"
	"log"
	"midas/network"
	"midas/common"
	"strconv"
	"encoding/json"
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
	ORDER_URI 			   = "order"
	EXCHANGE_INFO_URI 	   = "exchangeInfo"

	MIN_DEPTH = 5
	MAX_DEPTH = 100
)

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
	_, err := network.NewHttpRequest(
		"PUT",
		uri,
		map[string]string{"listenKey": *listenKey},
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
	_, err := network.NewHttpRequest(
		"DELETE",
		uri,
		map[string]string{"listenKey": *listenKey},
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
		Balances: make(map[string]*common.Balance),
	}
	for _, b := range rawAccount.Balances {
		f := common.ToFloat64(b.Free)
		l := common.ToFloat64(b.Locked)

		acc.Balances[b.Asset] = &common.Balance{
			CoinSymbol: b.Asset,
			Free:   f,
			Locked: l,
		}
	}

	return acc, nil
}

// only market or limit
func NewOrder(
	symbol string,
	side common.OrderSide,
	orderType common.OrderType,
	quantity float64,
	price float64,
	clientOrderId string,
	timestamp int64,
	test bool,
	) (*common.ExecutedOrderFullResponse, error) {

	if orderType != common.TypeMarket && orderType != common.TypeLimit {
		panic("NewOrder error: unsupported order type: " + string(orderType))
	}

	params := make(map[string]string)
	params["symbol"] = symbol
	params["side"] = string(side)
	params["type"] = string(orderType)
	if orderType == common.TypeLimit {
		params["timeInForce"] = string(common.IOC)
	}
	params["quantity"] = strconv.FormatFloat(quantity, 'f', -1, 64)
	if orderType == common.TypeLimit {
		params["price"] = strconv.FormatFloat(price, 'f', -1, 64)
	}
	params["timestamp"] = strconv.FormatInt(timestamp, 10)
	if clientOrderId != "" {
		params["newClientOrderId"] = clientOrderId
	}

	var orderUri string
	if test {
		orderUri = API_V3 + ORDER_URI + "/test"
	} else {
		orderUri = API_V3 + ORDER_URI
	}
	res, err := network.NewHttpRequest(
		"POST",
		orderUri,
		params,
		true,
		true)

	if err != nil {
		log.Println("MarketOrder error:", err)
		return nil, err
	}

	rawResponse := struct {
		Symbol        string `json:"symbol"`
		OrderID       int64 `json:"orderId"`
		ClientOrderID string `json:"clientOrderId"`
		TransactTime  float64 `json:"transactTime"`
		Price         string `json:"price"`
		OrigQty       string `json:"origQty"`
		ExecutedQty   string `json:"executedQty"`
		CumulativeQuoteQty string `json:"cummulativeQuoteQty"`
		Status        string `json:"status"`
		TimeInForce   string `json:"timeInForce"`
		Type          string `json:"type"`
		Side          string `json:"side"`
		Fills 		  []struct {
			Price 		string `json:"price"`
			Qty 		string `json:"qty"`
			Commission  string `json:"commission"`
			CommissionAsset string `json:"commissionAsset"`
		}
	}{}

	if err := json.Unmarshal(res, &rawResponse); err != nil {
		log.Println("MarketOrder unmarshaling error:", err)
		return nil, err
	}

	executedOrder := &common.ExecutedOrderFullResponse{
		rawResponse.Symbol,
		rawResponse.OrderID,
		rawResponse.ClientOrderID,
		common.TimeFromUnixTimestampFloat(rawResponse.TransactTime),
		common.ToFloat64(rawResponse.Price),
		common.ToFloat64(rawResponse.OrigQty),
		common.ToFloat64(rawResponse.ExecutedQty),
		common.ToFloat64(rawResponse.CumulativeQuoteQty),
		common.OrderStatus(rawResponse.Status),
		common.TimeInForce(rawResponse.TimeInForce),
		common.OrderType(rawResponse.Type),
		common.OrderSide(rawResponse.Side),
		make([]*common.Fill, 0),
	}

	for _, f := range rawResponse.Fills {
		executedOrder.Fills = append(executedOrder.Fills, &common.Fill{
			common.ToFloat64(f.Price),
			common.ToFloat64(f.Qty),
			common.ToFloat64(f.Commission),
			f.CommissionAsset,
		})
	}

	return executedOrder, nil
}

func GetExchangeInfo() (*common.ExchangeInfo, error) {
	exchangeInfoUri := API_V1 + EXCHANGE_INFO_URI
	res, err := network.NewHttpRequest(
		"GET",
		exchangeInfoUri,
		nil,
		false,
		false)
	if err != nil {
		log.Println("GetExchangeInfo error:", err)
		return nil, err
	}

	var info common.ExchangeInfo

	if err := json.Unmarshal(res, &info); err != nil {
		log.Println("GetExchangeInfo unmarshaling error:", err)
		return nil, err
	}

	return &info, nil
}