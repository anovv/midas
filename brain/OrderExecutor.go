package brain

import (
	"midas/common/arb"
	"midas/common"
	"time"
	"midas/logging"
	"midas/apis/binance"
	"sync"
	"sync/atomic"
	"strconv"
	"log"
)

const EXECUTION_MODE_TEST = true

var executableTriangles = sync.Map{}
var routineCounter int64 = 0

func SubmitOrders(state *arb.State) {
	// check if there is no active trades for arb with same coins
	// TODO decide if we should also check arb states with diff prices/timestamps
	if _, loaded := executableTriangles.LoadOrStore(state.Triangle.Key, true); loaded {
		return
	}
	// async schedule 3 trades
	log.Println("Submitting orders for " + state.Triangle.Key)
	now := time.Now()
	ts := common.UnixMillis(now)
	atomic.AddInt64(&routineCounter, 3)

	for _, order := range state.Orders {
		// TODO time sleep 1 ms?
		go func() {
			// generate order Id
			clientOrderId := order.Symbol + "_" + strconv.FormatInt(common.UnixMillis(now), 10)
			// get balances and log
			logging.QueueEvent(&logging.Event{
				EventType:logging.EventTypeOrderStatusChange,
				Value: &common.OrderStatusChangeEvent{
					OrderStatus: common.StatusNew,
					ArbStateId: state.Id,
					ClientOrderId: clientOrderId,
					Symbol: order.Symbol,
					Side: order.Side,
					Type: order.Type,
					Price: order.Price,
					OrigQty: order.Qty,
					ExecutedQty: 0.0,
					CumulativeQuoteQty: 0.0,
					TimeInForce: common.IOC,
					Fills: make([]*common.Fill, 0),
					ErrorMessage: "",
					TransactTime: time.Now(),
					BalanceA: account.Balances[state.Triangle.CoinA.CoinSymbol].Free,
					BalanceB: account.Balances[state.Triangle.CoinB.CoinSymbol].Free,
					BalanceC: account.Balances[state.Triangle.CoinC.CoinSymbol].Free,
				},
			})
			res, err := binance.NewOrder(
				order.Symbol,
				order.Side,
				common.TypeLimit,
				order.Qty,
				order.Price,
				clientOrderId,
				ts,
				EXECUTION_MODE_TEST,
			)

			// TODO proper wait for balance to be updated
			// TODO panic if not updated?
			// TODO discrepancies in balances logging
			// TODO THIS ALSO MAKES US IGNORE POTENTIAL ARB OPPS FOR EXECUTION
			time.Sleep(time.Duration(50) * time.Millisecond)

			// get balances and log
			if err == nil {
				log.Println("Order " + res.Symbol + " is executed")
				logging.QueueEvent(&logging.Event{
					EventType: logging.EventTypeOrderStatusChange,
					Value: &common.OrderStatusChangeEvent{
						OrderStatus:        res.Status,
						ArbStateId:         state.Id,
						ClientOrderId:      res.ClientOrderID,
						Symbol:             res.Symbol,
						Side:               res.Side,
						Type:               res.Type,
						Price:              res.Price,
						OrigQty:            res.OrigQty,
						ExecutedQty:        res.ExecutedQty,
						CumulativeQuoteQty: res.CumulativeQuoteQty,
						TimeInForce:        res.TimeInForce,
						Fills:              res.Fills,
						ErrorMessage:       "",
						TransactTime:       res.TransactTime,
						BalanceA:           account.Balances[state.Triangle.CoinA.CoinSymbol].Free,
						BalanceB:           account.Balances[state.Triangle.CoinB.CoinSymbol].Free,
						BalanceC:           account.Balances[state.Triangle.CoinC.CoinSymbol].Free,
					},
				})
			} else {
				log.Println("Order " + order.Symbol + " error: " + err.Error())
				logging.QueueEvent(&logging.Event{
					EventType: logging.EventTypeOrderStatusChange,
					Value: &common.OrderStatusChangeEvent{
						OrderStatus:        common.StatusError,
						ArbStateId:         state.Id,
						ClientOrderId: clientOrderId,
						Symbol: order.Symbol,
						Side: order.Side,
						Type: order.Type,
						Price: order.Price,
						OrigQty: order.Qty,
						ExecutedQty: 0.0,
						CumulativeQuoteQty: 0.0,
						TimeInForce: common.IOC,
						Fills: make([]*common.Fill, 0),
						ErrorMessage: err.Error(),
						TransactTime: time.Now(),
						BalanceA: account.Balances[state.Triangle.CoinA.CoinSymbol].Free,
						BalanceB: account.Balances[state.Triangle.CoinB.CoinSymbol].Free,
						BalanceC: account.Balances[state.Triangle.CoinC.CoinSymbol].Free,
					},
				})
			}

			// Increase ts for next trade
			atomic.AddInt64(&ts, 1)

			// remove from active orders
			atomic.AddInt64(&routineCounter, -1)
			if routineCounter == 0 {
				executableTriangles.Delete(state.Triangle.Key)
			}
		}()
	}
}