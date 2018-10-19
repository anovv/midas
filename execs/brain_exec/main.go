package main

import (
	"midas/brain"
)

func main() {
	brain.ScheduleTickerUpdates()
	brain.SetupRequestReceiver()
	defer brain.CleanupEyesHandler()
	//InitCoinGraph()
	brain.RunArbDetector()
}
