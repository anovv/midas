package main

import (
	"midas/apis/binance"
	"midas/brain"
	"time"
	"midas/logging"
	//"midas/common"
	//"fmt"
	"midas/common"
	"fmt"
)

func main() {
	initialize()

	coinPair := common.CoinPair{
		PairSymbol: "XLMBTC",
	}

	ts := common.UnixMillis(time.Now())
	clientOrderId := fmt.Sprintf("%s%d", coinPair.PairSymbol, ts)

	for i := 0; i < 20; i++ {
		tStart := time.Now()
		res, _ := binance.NewOrder(
			coinPair,
			common.SideSell,
			common.TypeLimit,
			100.0,
			0.00004069,
			clientOrderId,
			ts,
			true,
		)

		fmt.Printf("%+v\n", res)
		tEnd := time.Now()
		delta := tEnd.Sub(tStart)

		fmt.Printf("Executed in " + delta.String())

		time.Sleep(1 * time.Duration(1 * time.Second))
	}
}

func initialize() {
	logging.CreateTableIfNotExistsMySQL()
	brain.RunUpdateAccountInfo()
	brain.RunUpdateExchangeInfo()
	//brain.InitArbDetector()
	//brain.ScheduleTickerUpdates()
	//brain.SetupRequestReceiver()
	//defer brain.CleanupEyesHandler()
	//brain.RunArbDetector()
}
