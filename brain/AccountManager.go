package brain

import (
	"midas/common"
	"time"
	"midas/apis/binance"
	"log"
)

var account *common.Account

const ACCOUNT_UPDATE_PERIOD_MIN = 1

func RunUpdateAccountInfo() {
	StartUserDataStream()
	updateAccountInfo()
	go func() {
		for {
			time.Sleep(time.Duration(ACCOUNT_UPDATE_PERIOD_MIN) * time.Minute)
			updateAccountInfo()
		}
	}()
}

func updateAccountInfo() {
	log.Println("Updating account info...")
	acc, _ := binance.GetAccount()
	if acc != nil && (account == nil || account.LastUpdateTs.Before(acc.LastUpdateTs)) {
		account = acc
		log.Println("Updating account info... Done")
	}
}
