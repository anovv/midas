package brain

import (
	"midas/common/arb"
	"midas/common"
	"time"
	"midas/logging"
	"midas/apis/binance"
	"sync/atomic"
	"strconv"
	"log"
)

const EXECUTION_MODE_TEST = true

// TODO implement rate limiter
//type RateLimiter struct {
//	numTriggers int
//	lastStart time.Time
//	mux sync.Mutex
//}
//
//func (rl *RateLimiter) ShouldReject() bool {
//	rl.mux.Lock()
//	defer rl.mux.Unlock()
//
//	should := rl.numTriggers > 3 && time.Since(rl.lastStart) < time.Duration(10) * time.Second
//
//	if should {
//		return true
//	} else {
//		return false
//	}
//}
//
//func (rl *RateLimiter) Trigger() {
//	rl.mux.Lock()
//	defer rl.mux.Unlock()
//
//	if rl.numTriggers == 0 || rl.numTriggers > 3 {
//		rl.numTriggers = 0
//		rl.lastStart = time.Now()
//	}
//	rl.numTriggers++
//}

//TODO IMPLEMENT THIS
//type ExecutableStates struct {
//	coins sync.Map
//	mux sync.Mutex
//}
//
//func (ec *ExecutableStates) HasOverlapOrStore(state *arb.State) bool {
//	ec.mux.Lock()
//	defer ec.mux.Unlock()
//	_, hasA := ec.coins.LoadOrStore(state.Triangle.CoinA.CoinSymbol, true)
//	_, hasB := ec.coins.LoadOrStore(state.Triangle.CoinB.CoinSymbol, true)
//	_, hasC := ec.coins.LoadOrStore(state.Triangle.CoinC.CoinSymbol, true)
//
//	return hasA || hasB || hasC
//}
//
//func (ec *ExecutableStates) Delete(state *arb.State) {
//	ec.mux.Lock()
//	defer ec.mux.Unlock()
//	ec.coins.Delete(state.Triangle.CoinA.CoinSymbol)
//	ec.coins.Delete(state.Triangle.CoinB.CoinSymbol)
//	ec.coins.Delete(state.Triangle.CoinC.CoinSymbol)
//}
//
//var atomicExecutingStates = &ExecutableStates{
//	coins: sync.Map{},
//	mux: sync.Mutex{},
//}

var routineCounter int64 = 0
var isBusy = false

func ScheduleOrderExecutionIfNeeded(state *arb.State) {
	if !shouldExecute(state) {
		return
	}

	// async schedule 3 trades
	log.Println("Started execution for " + state.Id)
	now := time.Now()
	ts := common.UnixMillis(now)
	atomic.AddInt64(&routineCounter, 3)

	for _, orderRequest := range state.Orders {
		go func(request *common.OrderRequest) {
			// generate order Id
			clientOrderId := request.Symbol + "_" + strconv.FormatInt(common.UnixMillis(now), 10)
			// get balances and log
			logging.QueueEvent(&logging.Event{
				EventType:logging.EventTypeOrderStatusChange,
				Value: &common.OrderStatusChangeEvent{
					OrderStatus: common.StatusNew,
					ArbStateId: state.Id,
					ClientOrderId: clientOrderId,
					Symbol: request.Symbol,
					Side: request.Side,
					Type: request.Type,
					Price: request.Price,
					OrigQty: request.Qty,
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
				request.Symbol,
				request.Side,
				common.TypeLimit,
				request.Qty,
				request.Price,
				clientOrderId,
				ts,
				EXECUTION_MODE_TEST,
			)

			// TODO proper wait for balance to be updated
			// TODO panic if not updated?
			// TODO discrepancies in balances logging among arb currencies
			// TODO THIS ALSO MAKES US IGNORE POTENTIAL ARB OPPS FOR EXECUTION

			BIG_FUCKING_DELAY_DELET_THIS := time.Duration(100) * time.Millisecond
			time.Sleep(BIG_FUCKING_DELAY_DELET_THIS)

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
				log.Println("Order " + request.Symbol + " error: " + err.Error())
				// TODO report min notional reason
				logging.QueueEvent(&logging.Event{
					EventType: logging.EventTypeOrderStatusChange,
					Value: &common.OrderStatusChangeEvent{
						OrderStatus:        common.StatusError,
						ArbStateId:         state.Id,
						ClientOrderId: clientOrderId,
						Symbol: request.Symbol,
						Side: request.Side,
						Type: request.Type,
						Price: request.Price,
						OrigQty: request.Qty,
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

			// remove from active orders
			atomic.AddInt64(&routineCounter, -1)
			if routineCounter == 0 {
				log.Println("Finished execution for " + state.Id)
				isBusy = false
			}
		}(orderRequest)

		// Increase ts for next trade
		// TODO time sleep 1 ms for ts to increase properly?
		ts++
	}
}

func shouldExecute(state *arb.State) bool {
	// check frame count first
	if state.GetFrameUpdateCount() < 2 {
		return false
	}

	// TODO check if there is no active trades for arb with same coins
	// TODO decide if we should also check arb states with diff prices/timestamps
	if isBusy {
		return false
	}
	isBusy = true

	if state.ScheduledForExecution {
		return false
	}
	state.ScheduledForExecution = true

	for _, orderRequest := range state.Orders {
		check := FilterCheck(orderRequest.Symbol, orderRequest.Qty, orderRequest.Price)
		if check != common.FilterCheckOk {

			msg := string(check)
			if check == common.FilterCheckMinNotional && state.UsesAllBalance {
				msg += "_ALL_BALANCE"
			}

			log.Println(state.Id + " is dropped. Did not pass " + msg + " for pair " + orderRequest.Symbol + " Price: " + common.FloatToString(orderRequest.Price) + " Qty: " + common.FloatToString(orderRequest.Qty) + " | Tick size: " + common.FloatToString(GetTickSize(orderRequest.Symbol)) + " | Min price: " + common.FloatToString(GetMinPrice(orderRequest.Symbol)) + " | Min notional: " + common.FloatToString(GetMinNotional(orderRequest.Symbol)) + " | Step size: " + common.FloatToString(GetStepSize(orderRequest.Symbol)))
			isBusy = false
			return false
		}
	}

	if state.ProfitRelative <= 0.0001 {
		log.Println(state.Id + " is dropped. Profit is too low")
		isBusy = false
		return false
	}

	return true
}