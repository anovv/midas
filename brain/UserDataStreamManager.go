package brain

import (
	"fmt"
	"log"
	"github.com/gorilla/websocket"
	"midas/apis/binance"
	"time"
)

const KEEP_ALIVE_USER_DATA_STREAM_PERIOD_MINS = 30

var listenKey *string

// TODO return channel?
// TODO combine ws with polling account info
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
	// TODO parse message based on type
	// https://github.com/binance-exchange/binance-official-api-docs/blob/master/user-data-stream.md
}