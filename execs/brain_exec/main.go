package main

import (
	"midas/brain"
	"midas/logging"
)

func main() {
	initialize()
}

func initialize() {
	logging.CreateTableIfNotExistsMySQL()
	brain.RunUpdateAccountInfo()
	brain.RunUpdateExchangeInfo()
	brain.InitArbDetector()
	brain.ScheduleTickerUpdates()
	brain.SetupRequestReceiver()
	defer brain.CleanupEyesHandler()
	brain.RunArbDetector()
}
