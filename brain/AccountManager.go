package brain

import (
	"midas/common"
	"time"
	"midas/apis/binance"
)

var account *common.Account

const UPDATE_PERIOD_MINS = 1

func RunUpdateAccountInfo() {
	StartUserDataStream()
	//go func() {
		for {
			acc, _ := binance.GetAccount()
			if acc != nil && (account == nil || account.LastUpdateTs.Before(acc.LastUpdateTs)) {
				account = acc
				PrintAcc("poll acc update ")
			}
			time.Sleep(time.Duration(UPDATE_PERIOD_MINS) * time.Minute)
		}
	//}()
}
