package eyes

import (
	//. "midas/apis/binance"
	. "midas/common"
	. "github.com/pebbe/zmq4"
	//"net/http"
	//"encoding/json"
	"sync"
	"fmt"
	"strconv"
	//"strings"
	"midas/apis/binance"
	"net/http"
	"time"
)

//type EyeCallback func(dep *Depth, err error) ()
//func (eye *Eye) See(callback EyeCallback) {
//
//	bn := binance.New(http.DefaultClient, "", "")
//
//	for {
//		dep, err := bn.GetDepth(100, ETH_BTC)
//		//depJson, _ := json.Marshal(dep)
//		callback(dep, err)
//	}
//}
const (
	TCP_PREFIX    = "tcp://"
	BRAIN_ADDRESS = "localhost"
	BRAIN_CONNECTION_RECEIVER_PORT = 5500
	INPUT_BUFFER_SIZE = 10000
)

var channelIn = make(chan string, INPUT_BUFFER_SIZE)
var socketIn *Socket
var socketOut *Socket
var portIn int
var portOut int
var setupWg sync.WaitGroup

var bn *binance.Binance

func SetupEye() {
	requestSetupMetadata()
	socketIn = setupInSocket()
	socketOut = setupOutSocket()

	setupWg.Wait()
	fmt.Println("Killed eye")
}

func requestSetupMetadata() {
	socket, _ := NewSocket(REQ)
	address := TCP_PREFIX + BRAIN_ADDRESS + ":" + strconv.Itoa(BRAIN_CONNECTION_RECEIVER_PORT)
	fmt.Println("Requesting metadata from brain at address: " + address)
	socket.Connect(address)
	// TODO proper message
	message := Message{
		CONNECT_EYE,
		nil,
	}
	socket.Send(message.SerializeMessage(), 0)
	resp, _ := socket.Recv(0)
	response := DeserializeMessage(resp)
	if response.Command != CONFIRM_PORTS {
		fmt.Println("Bad port confirmation message from Brain, aborting...")
		return
	}
	args := response.Args
	// Brain port_in == Eye port_out
	fmt.Println("Received metadata: Port In: " + args[PORT_OUT] + " Port out: " + args[PORT_IN])
	portIn, _ = strconv.Atoi(args[PORT_OUT])
	portOut, _ = strconv.Atoi(args[PORT_IN])
}

func CleanupEye() {
	socketOut.Close()
	socketIn.Close()
}

func setupOutSocket() *Socket {
	out, _ := NewSocket(PULL)
	address := TCP_PREFIX + BRAIN_ADDRESS + ":" + strconv.Itoa(portOut)
	fmt.Println("Eye: connecting out to: " + address)
	out.Connect(address)
	message := Message{
		CONF_OUT,
		nil,
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

func setupInSocket() *Socket {
	in, _ := NewSocket(PUSH)
	address := TCP_PREFIX + BRAIN_ADDRESS + ":" + strconv.Itoa(portIn)
	fmt.Println("Eye: connecting in to: " + address)
	in.Connect(address)
	message := Message{
		CONF_IN,
		nil,
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
	fmt.Println("Eye received: " + messageSerialized)
	message := DeserializeMessage(messageSerialized)
	command := message.Command
	switch command {
	case KILL_EYE:
		setupWg.Done()
	case DEPTH_REQ:
		args := message.Args
		pair := args[CURRENCY_PAIR]
		exchange := args[EXCHANGE]

		if exchange != BINANCE {
			fmt.Println("Unsupported exchange " + exchange)
			return
		}

		if bn == nil {
			bn = binance.New(http.DefaultClient, "", "")
		}
		go func() {
			tStart := time.Now()
			depth, err := bn.GetDepth(100, pair)
			if err != nil {
				fmt.Println("Error fetching depth")
				return
			}
			delta := time.Since(tStart) // nanosec
			fetchTimeMicroSeconds := strconv.Itoa(int(delta/1000)) // microsec
			fmt.Println("Depth fetched in " + fetchTimeMicroSeconds + " microseconds")

			response := Message{
				DEPTH_RESP,
				map[string]string{
					DEPTH_SERIALIZED:        depth.Serialize(),
					FETCH_TIME_MICROSECONDS: fetchTimeMicroSeconds,
					CURRENCY_PAIR:           pair,
					EXCHANGE:                exchange,
				},
			}

			channelIn<-response.SerializeMessage()
		} ()
	}
}
