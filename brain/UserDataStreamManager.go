package brain

import (
	"fmt"
	"log"
	"github.com/gorilla/websocket"
	"midas/apis/binance"
	"time"
	"encoding/json"
	"midas/common"
)

const KEEP_ALIVE_USER_DATA_STREAM_PERIOD_MINS = 30

const (
	OUTBOUND_ACCOUNT_INFO_EVENT_TYPE = "outboundAccountInfo"
	EXECUTION_REPORT_EVENT_TYPE = "executionReport"
)

var listenKey *string

func StartUserDataStream() {
	listenKey, err := binance.GetUserDataStreamListenKey()
	if err != nil {
		log.Println("Failed to obtain listenKey")
		return
	}
	url := fmt.Sprintf("wss://stream.binance.com:9443/ws/%s", *listenKey)
	log.Println("Connecting to user data stream websocket: ", url)
	c, _, err := websocket.DefaultDialer.Dial(url, nil)
	if err != nil {
		log.Println("Error connecting to user data stream websocket: ", err)
		return
	}

	c.SetCloseHandler(
		func(code int, text string) error {
			restartUserDataStream()
			return nil
		},
	)

	go func() {
		defer c.Close()
		for {
			_, message, err := c.ReadMessage()
			if err != nil {
				log.Println("Error reading from user data stream websocket: ", err)
				return
			}
			handleMsg(message)
		}
	} ()

	// keep alive user data stream
	go func() {
		for {
			if listenKey != nil {
				binance.PingUserDataStream(listenKey)
				log.Println("Pinging user data stream websocket...")
				time.Sleep(time.Duration(KEEP_ALIVE_USER_DATA_STREAM_PERIOD_MINS) * time.Minute)
			}
		}
	} ()
}

func restartUserDataStream() {
	log.Println("Reconnecting to user data stream...")
	binance.CloseUserDataStream(listenKey)
	StartUserDataStream()
}

func handleMsg(message []byte) {
	str := fmt.Sprintf("%s", message)
	log.Println("UserDataStream update: " + str)

	var rawRespEventType map[string]interface{}

	if err := json.Unmarshal(message, &rawRespEventType); err != nil {
		log.Println("Error unmarshaling event type: ", err)
		return
	}

	switch rawRespEventType["e"].(string) {
	case OUTBOUND_ACCOUNT_INFO_EVENT_TYPE:
		rawAccount := struct {
			Type            string  `json:"e"`
			Time            float64 `json:"E"`
			OpenTime        float64 `json:"t"`
			MakerCommission  int64   `json:"m"`
			TakerCommission  int64   `json:"t"`
			BuyerCommission  int64   `json:"b"`
			SellerCommission int64   `json:"s"`
			UpdateTime		float64 `json:"u"`
			Balances        []struct {
				Asset            string `json:"a"`
				Free 			 string `json:"f"`
				Locked           string `json:"l"`
			} `json:"B"`
		}{}

		if err := json.Unmarshal(message, &rawAccount); err != nil {
			log.Println("Error unmarshaling raw account: ", err)
			return
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

		if account == nil || account.LastUpdateTs.Before(acc.LastUpdateTs) {
			account = acc
			//PrintAcc("ws acc update: ")
		}
	case EXECUTION_REPORT_EVENT_TYPE:
		// TODO handle order execution updates
	default:
		return
	}
}

func PrintAcc(msg string) {
	if account != nil {
		out, err := json.Marshal(account)
		if err != nil {
			panic (err)
		}

		fmt.Println(msg + string(out))
	}
}