package logging

import (
	_ "github.com/go-sql-driver/mysql"
	"midas/common/arb"
	"database/sql"
	"midas/configuration"
	"log"
	"time"
)

// MySQL field names
const (
	FIELD_ARB_CHAIN                  = "arb_chain"
	FIELD_QTY_BEFORE                 = "qty_before"
	FIELD_QTY_AFTER                  = "qty_after"
	FIELD_RELATIVE_PROFIT_PERCENTAGE = "relative_profit_percentage"
	FIELD_LASTED_FOR_MS              = "lasted_for_ms"
	FIELD_COIN_A                     = "coin_a"
	FIELD_COIN_B        = "coin_b"
	FIELD_COIN_C        = "coin_c"
	FIELD_STARTED_AT    = "started_at"
	FIELD_FINISHED_AT   = "finished_at"
	FIELD_LASTED_FRAMES = "lasted_frames"
	FIELD_SYMBOL_AB     = "symbol_ab"
	FIELD_SIDE_AB       = "side_ab"
	FIELD_TRADE_QTY_AB  = "trade_qty_ab"
	FIELD_ORDER_QTY_AB  = "order_qty_ab"
	FIELD_PRICE_AB      = "price_ab"
	FIELD_SYMBOL_BC     = "symbol_bc"
	FIELD_SIDE_BC       = "side_bc"
	FIELD_TRADE_QTY_BC  = "trade_qty_bc"
	FIELD_ORDER_QTY_BC  = "order_qty_bc"
	FIELD_PRICE_BC      = "price_bc"
	FIELD_SYMBOL_AC     = "symbol_ac"
	FIELD_SIDE_AC       = "side_ac"
	FIELD_TRADE_QTY_AC  = "trade_qty_ac"
	FIELD_ORDER_QTY_AC  = "order_qty_ac"
	FIELD_PRICE_AC      = "price_ac"
	FIELD_BALANCE_A 	= "balance_a"
	FIELD_BALANCE_B 	= "balance_b"
	FIELD_BALANCE_C 	= "balance_c"
)

const (
	ARB_STATE_RECORDS_BUFFER_SIZE = 1000
	DB_DRIVER = "mysql"
	DB_USER = "root"
	DB_NAME = "midas"
	TABLE_NAME = "trade_and_arb_state_binance"
	CREATE_TABLE_QUERY = "CREATE TABLE IF NOT EXISTS " + TABLE_NAME + "(" +
		"id INT(10) NOT NULL AUTO_INCREMENT," +
		FIELD_ARB_CHAIN + " VARCHAR(64)," +
		FIELD_QTY_BEFORE + " FLOAT(16, 8)," +
		FIELD_QTY_AFTER + " FLOAT(16, 8)," +
		FIELD_RELATIVE_PROFIT_PERCENTAGE + " FLOAT(16, 8)," +
		FIELD_LASTED_FOR_MS + " INT(10)," +
		FIELD_COIN_A + " VARCHAR(64)," +
		FIELD_COIN_B + " VARCHAR(64)," +
		FIELD_COIN_C + " VARCHAR(64)," +
		FIELD_STARTED_AT + " TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP," +
		FIELD_FINISHED_AT + " TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP," +
		FIELD_LASTED_FRAMES + " INT(10)," +
		FIELD_SYMBOL_AB + " VARCHAR(64)," +
		FIELD_SYMBOL_BC + " VARCHAR(64)," +
		FIELD_SYMBOL_AC + " VARCHAR(64)," +
		FIELD_SIDE_AB + " VARCHAR(64)," +
		FIELD_SIDE_BC + " VARCHAR(64)," +
		FIELD_SIDE_AC + " VARCHAR(64)," +
		FIELD_TRADE_QTY_AB + " FLOAT(16, 8)," +
		FIELD_TRADE_QTY_BC + " FLOAT(16, 8)," +
		FIELD_TRADE_QTY_AC + " FLOAT(16, 8)," +
		FIELD_ORDER_QTY_AB + " FLOAT(16, 8)," +
		FIELD_ORDER_QTY_BC + " FLOAT(16, 8)," +
		FIELD_ORDER_QTY_AC + " FLOAT(16, 8)," +
		FIELD_PRICE_AB + " FLOAT(16, 8)," +
		FIELD_PRICE_BC + " FLOAT(16, 8)," +
		FIELD_PRICE_AC + " FLOAT(16, 8)," +
		FIELD_BALANCE_A + " FLOAT(16, 8)," +
		FIELD_BALANCE_B + " FLOAT(16, 8)," +
		FIELD_BALANCE_C + " FLOAT(16, 8)," +
		"PRIMARY KEY (id)" +
		");"
	INSERT_ARB_STATE_QUERY = "INSERT INTO " + TABLE_NAME + "(" +
		FIELD_ARB_CHAIN + ","  +
		FIELD_QTY_BEFORE + ","  +
		FIELD_QTY_AFTER + ","  +
		FIELD_RELATIVE_PROFIT_PERCENTAGE + ","  +
		FIELD_LASTED_FOR_MS + ","  +
		FIELD_COIN_A + ","  +
		FIELD_COIN_B + ","  +
		FIELD_COIN_C + ","  +
		FIELD_STARTED_AT + ","  +
		FIELD_FINISHED_AT + "," +
		FIELD_LASTED_FRAMES + "," +
		FIELD_SYMBOL_AB + "," +
		FIELD_SYMBOL_BC + "," +
		FIELD_SYMBOL_AC + "," +
		FIELD_SIDE_AB + "," +
		FIELD_SIDE_BC + "," +
		FIELD_SIDE_AC + "," +
		FIELD_TRADE_QTY_AB + "," +
		FIELD_TRADE_QTY_BC + "," +
		FIELD_TRADE_QTY_AC + "," +
		FIELD_ORDER_QTY_AB + "," +
		FIELD_ORDER_QTY_BC + "," +
		FIELD_ORDER_QTY_AC + "," +
		FIELD_PRICE_AB + "," +
		FIELD_PRICE_BC + "," +
		FIELD_PRICE_AC + "," +
		FIELD_BALANCE_A + "," +
		FIELD_BALANCE_B + "," +
		FIELD_BALANCE_C +
		") VALUES(?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?)"

	TIMESTAMP_FORMAT = "2006-01-02 15:04:05"
)

var stateRecords = make(chan *arb.State, ARB_STATE_RECORDS_BUFFER_SIZE)

// Puts arbState in a queue for async logging
func SubmitArbState(state *arb.State) {
	stateRecords<-state
}

// TODO better logger abstraction
func recordArbStateMySQL(state *arb.State) {
	dbPass := configuration.ReadBrainConfig().MYSQL_PASSWORD
	db, err := sql.Open(DB_DRIVER, DB_USER + ":" + dbPass + "@tcp(127.0.0.1:3306)/" + DB_NAME)
	defer db.Close()

	if checkErr(err) {
		return
	}

	stmt, err := db.Prepare(INSERT_ARB_STATE_QUERY)

	if checkErr(err) {
		return
	}

	arbChain := state.Triangle.CoinA.CoinSymbol + "->" +
		state.Triangle.CoinB.CoinSymbol + "->" +
		state.Triangle.CoinC.CoinSymbol + "->" +
		state.Triangle.CoinA.CoinSymbol
	lastedForMs := int64(state.LastUpdateTs.Sub(state.StartTs)/time.Millisecond)
	_, err = stmt.Exec(
		arbChain,
		state.QtyBefore,
		state.QtyAfter,
		state.ProfitRelative * 100.0,
		lastedForMs,
		state.Triangle.CoinA.CoinSymbol,
		state.Triangle.CoinB.CoinSymbol,
		state.Triangle.CoinC.CoinSymbol,
		state.StartTs.Format(TIMESTAMP_FORMAT),
		state.LastUpdateTs.Format(TIMESTAMP_FORMAT),
		state.GetFrameUpdateCount(),
		state.Orders["AB"].Symbol,
		state.Orders["BC"].Symbol,
		state.Orders["AC"].Symbol,
		string(state.Orders["AB"].Side),
		string(state.Orders["BC"].Side),
		string(state.Orders["AC"].Side),
		state.Orders["AB"].Qty,
		state.Orders["BC"].Qty,
		state.Orders["AC"].Qty,
		state.OrderQtyAB,
		state.OrderQtyBC,
		state.OrderQtyAC,
		state.Orders["AB"].Price,
		state.Orders["BC"].Price,
		state.Orders["AC"].Price,
		state.BalanceA,
		state.BalanceB,
		state.BalanceC,
	)
	checkErr(err)
}

func InitMySQLLogger() {
	createTableIfNotExists()
	startLoggingRoutine()
}

func createTableIfNotExists() {
	dbPass := configuration.ReadBrainConfig().MYSQL_PASSWORD
	db, err := sql.Open(DB_DRIVER, DB_USER + ":" + dbPass + "@tcp(127.0.0.1:3306)/")
	defer db.Close()

	if err != nil {
		panic(err)
	}

	_,err = db.Exec("CREATE DATABASE IF NOT EXISTS " + DB_NAME)
	if err != nil {
		panic(err)
	}

	_,err = db.Exec("USE "+ DB_NAME)
	if err != nil {
		panic(err)
	}

	_,err = db.Exec(CREATE_TABLE_QUERY)
	if err != nil {
		panic(err)
	}
}

func startLoggingRoutine() {
	go func() {
		for {
			state := <-stateRecords
			recordArbStateMySQL(state)
		}
	}()
}

func checkErr(err error) bool {
	if err != nil {
		log.Println("Got error logging to MySQL:")
		log.Println(err.Error())
		return true
	}

	return false
}

