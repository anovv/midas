package main

import (
	"midas/brain"
)

func main() {
	brain.ScheduleTickerUpdates()
	brain.SetupRequestReceiver()
	brain.CleanupEyesHandler()
	//InitCoinGraph()
	//brain.RunArbDetector()
}
