package common

import "time"

type OrderStatus string

type OrderType string

type OrderSide string

type TimeInForce string

type OrderRespType string

var (
	StatusNew             = OrderStatus("NEW")
	StatusPartiallyFilled = OrderStatus("PARTIALLY_FILLED")
	StatusFilled          = OrderStatus("FILLED")
	StatusCanceled       = OrderStatus("CANCELED")
	StatusPendingCancel   = OrderStatus("PENDING_CANCEL")
	StatusRejected        = OrderStatus("REJECTED")
	StatusExpired         = OrderStatus("EXPIRED")

	TypeLimit  = OrderType("LIMIT")
	TypeMarket = OrderType("MARKET")

	SideBuy  = OrderSide("BUY")
	SideSell = OrderSide("SELL")

	IOC = TimeInForce("IOC")
	GTC = TimeInForce("GTC")

	RespTypeAck = OrderRespType("ACK")
	RespTypeResult = OrderRespType("RESULT")
	RespTypeFull = OrderRespType("FULL")

)

type OrderRequest struct {
	Symbol string
	Side OrderSide
	Type OrderType
	Qty float64
	Price float64
}

func (or OrderRequest) String() string {
	return "[Side: " + string(or.Side) +
		" | Symbol: " + or.Symbol +
			" | Type: " + string(or.Type) +
				" | Qty: " + FloatToString(or.Qty) +
					" | Price: " + FloatToString(or.Price) + "]"
}

type ExecutedOrderFullResponse struct {
	Symbol        string
	OrderID       int64
	ClientOrderID string
	TransactTime  time.Time
	Price         float64
	OrigQty       float64
	ExecutedQty   float64
	CumulativeQuoteQty float64
	Status        OrderStatus
	TimeInForce   TimeInForce
	Type          OrderType
	Side          OrderSide
	Fills 		  []*Fill
}

type Fill struct {
	Price 		float64
	Qty 		float64
	Commission  float64
	CommissionAsset string
}