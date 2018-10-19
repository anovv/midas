package common

import (
	"encoding/json"
)

type Message struct {
	Command string
	Args map[string]string
}

// Commands
const (
	DEPTH_REQ  = "depth_req"
	DEPTH_RESP  = "depth_resp"
	TICKERS_MAP_REQ = "tickers_map_req"
	TICKERS_MAP_RESP = "tickers_map_resp"
	CONNECT_EYE  = "connect_eye"
	CONFIRM_PORTS  = "confirm_ports"
	KILL_EYE  = "kill_eye"
	CONF_IN  = "conf_in"
	CONF_OUT  = "conf_out"
)

// Argument keys
const (
	PORT_IN                 = "port_in"
	PORT_OUT                = "port_out"
	CURRENCY_PAIR           = "currency_pair"
	EXCHANGE                = "exchange"
	DEPTH_SERIALIZED        = "depth_serialized"
	TICKERS_MAP_SERIALIZED        = "tickers_map_serialized"
	FETCH_TIME_MICROSECONDS = "fetch_time_microseconds"
)

func (message *Message) SerializeMessage() string {
	out, err := json.Marshal(message)
	if err != nil {
		panic (err)
	}

	return string(out)
}

func DeserializeMessage(messageJson string) *Message {
	var message *Message
	err := json.Unmarshal([]byte(messageJson), &message)
	if err != nil {
		panic(err)
	}

	return message
}

