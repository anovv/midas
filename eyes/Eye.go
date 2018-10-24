package eyes

import (
	"midas/common"
	"github.com/pebbe/zmq4"
	"sync"
	"strconv"
	"midas/apis/binance"
	"time"
	"log"
	"midas/configuration"
)

const (
	TCP_PREFIX    = "tcp://"
	INPUT_BUFFER_SIZE = 10000
)

var eyeConfig = configuration.ReadEyeConfig()

var channelIn = make(chan string, INPUT_BUFFER_SIZE)
var socketIn *zmq4.Socket
var socketOut *zmq4.Socket
var portIn int
var portOut int
var setupWg sync.WaitGroup

func SetupEye() {
	requestSetupMetadata()
	socketIn = setupInSocket()
	socketOut = setupOutSocket()

	setupWg.Wait()
	log.Println("Killed eye")
}

func requestSetupMetadata() {
	socket, _ := zmq4.NewSocket(zmq4.REQ)
	address := TCP_PREFIX + eyeConfig.BRAIN_ADDRESS + ":" + strconv.Itoa(eyeConfig.BRAIN_CONNECTION_RECEIVER_PORT)
	log.Println("Requesting metadata from brain at address: " + address)
	socket.Connect(address)
	// TODO proper message
	message := common.Message{
		common.CONNECT_EYE,
		nil,
		nil,
		"",
	}
	socket.Send(message.SerializeMessage(), 0)
	resp, _ := socket.Recv(0)
	response := common.DeserializeMessage(resp)
	if response.Command != common.CONFIRM_PORTS {
		log.Println("Bad port confirmation message from Brain, aborting...")
		return
	}
	args := response.Args
	// Brain port_in == Eye port_out
	log.Println("Received metadata: Port In: " + args[common.PORT_OUT] + " Port out: " + args[common.PORT_IN])
	portIn, _ = strconv.Atoi(args[common.PORT_OUT])
	portOut, _ = strconv.Atoi(args[common.PORT_IN])
}

func CleanupEye() {
	socketOut.Close()
	socketIn.Close()
}

func setupOutSocket() *zmq4.Socket {
	out, _ := zmq4.NewSocket(zmq4.PULL)
	address := TCP_PREFIX + eyeConfig.BRAIN_ADDRESS + ":" + strconv.Itoa(portOut)
	log.Println("Eye: connecting out to: " + address)
	out.Connect(address)
	message := common.Message{
		common.CONF_OUT,
		nil,
		nil,
		"",
	}
	socketIn.Send(message.SerializeMessage(), 0)
	setupWg.Add(1)
	go func(){
		for {
			msg, _ := out.Recv(0)
			handleMessage(msg)
		}
	}()

	return out
}

func setupInSocket() *zmq4.Socket {
	in, _ := zmq4.NewSocket(zmq4.PUSH)
	address := TCP_PREFIX + eyeConfig.BRAIN_ADDRESS + ":" + strconv.Itoa(portIn)
	log.Println("Eye: connecting in to: " + address)
	in.Connect(address)
	message := common.Message{
		common.CONF_IN,
		nil,
		nil,
		"",
	}
	in.Send(message.SerializeMessage(), 0)
	go func(){
		for {
			msg := <-channelIn
			in.Send(msg,0)
		}
	}()

	return in
}

func handleMessage(messageSerialized string) {
	log.Println("Eye received: " + messageSerialized)
	message := common.DeserializeMessage(messageSerialized)
	command := message.Command
	switch command {
	case common.KILL_EYE:
		setupWg.Done()
	case common.DEPTH_REQ:
		args := message.Args
		pair := args[common.CURRENCY_PAIR]
		exchange := args[common.EXCHANGE]

		if exchange != common.BINANCE {
			log.Println("Unsupported exchange " + exchange)
			return
		}

		go func() {
			tStart := time.Now()
			depth, err := binance.GetDepth(100, pair)
			var serialized string
			var errMsg string
			if err != nil {
				log.Println("Error fetching depth")
				serialized = ""
				errMsg = err.Error()
			} else {
				serialized = depth.Serialize()
				errMsg = ""
			}
			tEnd := time.Now()
			delta := tEnd.Sub(tStart)
			log.Println("Depth fetched in " + delta.String())

			response := common.Message{
				common.DEPTH_RESP,
				map[string]string{
					common.DEPTH_SERIALIZED:        serialized,
					common.CURRENCY_PAIR:           pair,
					common.EXCHANGE:                exchange,
				},
				&common.TraceInfo{
					BrainReqSentTs: message.TraceInfo.BrainReqSentTs,
					EyeReqSentTs: tStart,
					EyeRespReceivedTs: tEnd,
				},
				errMsg,
			}

			channelIn<-response.SerializeMessage()
		} ()

	case common.TICKERS_MAP_REQ:
		args := message.Args
		exchange := args[common.EXCHANGE]

		if exchange != common.BINANCE {
			log.Println("Unsupported exchange " + exchange)
			return
		}

		go func() {
			tStart := time.Now()
			tickers, err := binance.GetAllTickers()
			var serialized string
			var errMsg string
			if err != nil {
				log.Println("Error fetching tickers")
				serialized = ""
				errMsg = err.Error()
			} else {
				serialized = tickers.Serialize()
				errMsg = ""
			}
			tEnd := time.Now()
			delta := tEnd.Sub(tStart) // nanosec
			log.Println("Tickers fetched in " + delta.String())

			response := common.Message{
				common.TICKERS_MAP_RESP,
				map[string]string{
					common.TICKERS_MAP_SERIALIZED:	serialized,
					common.EXCHANGE:				exchange,
				},
				&common.TraceInfo{
					BrainReqSentTs: message.TraceInfo.BrainReqSentTs,
					EyeReqSentTs: tStart,
					EyeRespReceivedTs: tEnd,
				},
				errMsg,
			}

			channelIn<-response.SerializeMessage()
		} ()
	}
}
