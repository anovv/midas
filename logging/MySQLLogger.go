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
	FIELD_ARB_CHAIN = "arb_chain"
	FIELD_QTY_BEFORE = "qty_before"
	FIELD_QTY_AFTER = "qty_after"
	FIELD_RELATIVE_PROFIT_PERCENTAGE = "relative_profit_percentage"
	FIELD_LASTED_FOR_MS = "lasted_for_ms"
	FIELD_COIN_A = "coin_a"
	FIELD_COIN_B = "coin_b"
	FIELD_COIN_C = "coin_c"
	FIELD_STARTED_AT = "started_at"
	FIELD_FINISHED_AT = "finished_at"
	FIELD_LASTED_FRAMES = "lasted_frames"
)

const (
	DB_DRIVER = "mysql"
	DB_USER = "root"
	DB_NAME = "midas"
	TABLE_NAME = "arb_state_binance"
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
		FIELD_LASTED_FRAMES +
		") VALUES(?,?,?,?,?,?,?,?,?,?,?)"

	TIMESTAMP_FORMAT = "2006-01-02 15:04:05"
)

// TODO better logger abstraction
func RecordArbStateMySQL(state *arb.State) {
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
		state.NumFrames,
	)
	checkErr(err)
}

func CreateTableIfNotExistsMySQL() {
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

func checkErr(err error) bool {
	if err != nil {
		log.Println("Got error logging to MySQL:")
		log.Println(err.Error())
		return true
	}

	return false
}

