package main

import (
	//"midas/apis/binance"
	"midas/brain"
	//"time"
	"midas/logging"
	//"midas/common"
	//"fmt"
	//"midas/common"
	//"fmt"
	//"midas/common"
	//"time"
	//"fmt"
)

func main() {
	initialize()

	//symbol := "EOSBTC"
	//
	//ts := common.UnixMillis(time.Now())
	//clientOrderId := fmt.Sprintf("%s%d", symbol, ts)
	//
	//for i := 0; i < 20; i++ {
	//	tStart := time.Now()
	//	res, _ := binance.NewOrder(
	//		symbol,
	//		common.SideSell,
	//		common.TypeMarket,
	//		100.0,
	//		0.0,
	//		clientOrderId,
	//		ts,
	//		true,
	//	)
	//
	//	fmt.Printf("%+v\n", res)
	//	tEnd := time.Now()
	//	delta := tEnd.Sub(tStart)
	//
	//	fmt.Printf("Executed in " + delta.String())
	//
	//	time.Sleep(1 * time.Duration(1 * time.Second))
	//}
}

func initialize() {
	logging.CreateTableIfNotExistsMySQL()
	brain.RunUpdateAccountInfo()
	brain.RunUpdateExchangeInfo()
	//brain.InitArbDetector()
	brain.RebalancePortfolio()
	//brain.ScheduleTickerUpdates()
	//brain.SetupRequestReceiver()
	//defer brain.CleanupEyesHandler()
	//brain.RunArbDetector()
}
