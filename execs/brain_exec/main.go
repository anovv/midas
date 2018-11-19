package main

import (
	"midas/brain"
	"midas/logging"
)

func main() {
	initialize()
}

func initialize() {
	logging.InitMySQLLogger()
	brain.RunUpdateAccountInfo()
	brain.RunUpdateExchangeInfo()
	brain.ScheduleTickerUpdates()
	brain.SetupRequestReceiver()
	defer brain.CleanupEyesHandler()
	brain.RunArbDetector()
}
