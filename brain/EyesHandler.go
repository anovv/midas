package brain

import (
	"github.com/pebbe/zmq4"
	"strconv"
	"midas/common"
	"time"
	"encoding/json"
	"log"
)

const (
	INPUT_BUFFER_SIZE = 10000
	TCP_PREFIX = "tcp://*:"
)

type EyeState int
const (
	NOT_READY EyeState = 1
	OUT_READY EyeState = 2
	IN_READY EyeState = 3
	READY EyeState = 4
)

type EyeHandle struct {
	ChannelIn *chan string
	SocketIn *zmq4.Socket
	SocketOut *zmq4.Socket
	EyeId int
	PortPair *PortPair
	EyeState EyeState
}

type PortPair struct {
	PortIn int
	PortOut int
}

type AvgFetchTimeStat struct {
	AvgFetchTimeMicroSeconds int
	NumSamples int
}

// No need for mutex as we simply update this variable with a new map instance on each write
var tickersMap *common.TickersMap

var eyes = make(map[int]*EyeHandle)
var lastConnectedEyeId = -1

var pairsPerExchange = map[string][]string{
	common.BINANCE: {"ETHBTC"},
}

var lastUpdTs = time.Now()
var lastReqSentTs = time.Now()

// TODO merge depth and ticker updates in a single function
func ScheduleDepthUpdates() {
		for exchange, pairList := range pairsPerExchange {
			go func() {
				var pairIndex = 0
				for {
					for _, eyeHandle := range eyes {
						pair := pairList[pairIndex]
						delay := getDelayMicroSeconds(common.DEPTH_REQ, exchange)
						message := common.Message{
							common.DEPTH_REQ,
							map[string]string{
								common.CURRENCY_PAIR: pair,
								common.EXCHANGE: exchange,
							},
							&common.TraceInfo{
								BrainReqSentTs: time.Now(),
							},
						}
						time.Sleep(time.Duration(delay) * time.Microsecond)
						*eyeHandle.ChannelIn<-message.SerializeMessage()
						pairIndex++
						if pairIndex == len(pairList) {
							pairIndex = 0
						}
					}
				}
			} ()
		}
}

// TODO merge depth and ticker updates in a single function
func ScheduleTickerUpdates() {
	for exchange, _ := range pairsPerExchange {
		go func() {
			for {
				for _, eyeHandle := range eyes {
					delay := getDelayMicroSeconds(common.TICKERS_MAP_REQ, exchange)
					message := common.Message{
						common.TICKERS_MAP_REQ,
						map[string]string{
							common.EXCHANGE: exchange,
						},
						&common.TraceInfo{
							BrainReqSentTs: time.Now(),
						},
					}
					time.Sleep(time.Duration(delay) * time.Microsecond)
					*eyeHandle.ChannelIn<-message.SerializeMessage()
				}
			}
		} ()
	}
}

func getDelayMicroSeconds(command string, exchange string) int {
	switch command {
	case common.TICKERS_MAP_REQ:
		numEyes := len(eyes)
		return int(brainConfig.FETCH_DELAYS_MICROS[exchange][command]/numEyes)
	// TODO handle depth
	default:
		return 0
	}
}

func SetupRequestReceiver() {
	go func() {
		requestReceiver, _ := zmq4.NewSocket(zmq4.REP)
		requestReceiver.Bind(TCP_PREFIX + strconv.Itoa(brainConfig.CONNECTION_RECEIVER_PORT))
		for {
			request, error := requestReceiver.Recv(0)
			if error != nil {
				log.Println("Receiver error error: " + error.Error())
				continue
			}
			handleNewConnectionRequest(requestReceiver, request)
		}
	} ()
}

func handleNewConnectionRequest(requestReceiver *zmq4.Socket, requestSerialized string) {
	request := common.DeserializeMessage(requestSerialized)
	if request.Command != common.CONNECT_EYE {
		log.Println("Bad connection request, aborting...")
		return
	}
 	log.Println("New connection request")

	eyeId := lastConnectedEyeId + 1
	var portIn int
	if eyeId == 0 {
		// first connection
		portIn = brainConfig.BASE_PORT
	} else {
		lastEyeHandle := eyes[lastConnectedEyeId]
		portIn = lastEyeHandle.PortPair.PortIn + 2
	}
	portOut := portIn + 1
	portPair := PortPair{portIn, portOut}
	channelIn := make(chan string, INPUT_BUFFER_SIZE)
	socketIn := setupInSocket(portIn, eyeId, &channelIn)
	socketOut := setupOutSocket(portOut, eyeId)
	eyeHandle := EyeHandle{
		&channelIn,
		socketIn,
		socketOut,
		eyeId,
		&portPair,
		NOT_READY,
	}
	eyes[eyeId] = &eyeHandle
	message := common.Message{
		common.CONFIRM_PORTS,
		map[string]string{
			common.PORT_IN: strconv.Itoa(portIn),
			common.PORT_OUT: strconv.Itoa(portOut),
			},
		nil,
	}
	requestReceiver.Send(message.SerializeMessage(), 0)
	lastConnectedEyeId = eyeId
}

func CleanupEyesHandler() {
	for eyeId := range eyes {
		eyeInterface := eyes[eyeId]
		message := common.Message{
			common.KILL_EYE,
			nil,
			nil,
		}
		*eyeInterface.ChannelIn<-message.SerializeMessage()
		// TODO block until channel is empty
		eyeInterface.SocketIn.Close()
		eyeInterface.SocketOut.Close()
	}
}

func setupOutSocket(portOut int, eyeId int) *zmq4.Socket {
	out, _ := zmq4.NewSocket(zmq4.PULL)
	out.Bind(TCP_PREFIX + strconv.Itoa(portOut))
	log.Println("Waiting for eye " + strconv.Itoa(eyeId) + " to confirm out connection on port " + strconv.Itoa(portOut))
	go func(){
		for {
			msg, error := out.Recv(0)
			if error != nil {
				log.Println("Out error: " + error.Error())
				continue
			}
			go func() {
				handleMessage(msg, eyeId)
			} ()
		}
	}()

	return out
}

func setupInSocket(portIn int, eyeId int, channelIn *chan string) *zmq4.Socket {
	in, _ := zmq4.NewSocket(zmq4.PUSH)
	in.Bind(TCP_PREFIX + strconv.Itoa(portIn))
	log.Println("Waiting for eye " + strconv.Itoa(eyeId) + " to confirm in connection on port " + strconv.Itoa(portIn))
	go func(){
		for {
			msg := <-*channelIn
			_ , error := in.Send(msg,0)
			if error != nil {
				log.Println("In error: " + error.Error())
			}
		}
	}()

	return in
}

func handleMessage(messageSerialized string, eyeId int) {
	message := common.DeserializeMessage(messageSerialized)
	command := message.Command
	args := message.Args
	switch command {
	case common.DEPTH_RESP:
		depthSerialized := args[common.DEPTH_SERIALIZED]
		depth := common.DeserializeDepth(depthSerialized)
		dj, _ := json.Marshal(depth)
		log.Println("Received depth update: " + string(dj))

	case common.TICKERS_MAP_RESP:
		// TODO generalize to all
		if lastReqSentTs.After(message.TraceInfo.BrainReqSentTs) {
			log.Println("Frame dropped")
			return
		}
		diff := time.Since(lastUpdTs)
		log.Println("Upd time: " + diff.String())
		lastUpdTs = time.Now()
		lastReqSentTs = message.TraceInfo.BrainReqSentTs

		tickersMapSerialized := args[common.TICKERS_MAP_SERIALIZED]
		tickersMap = common.DeserializeTickersMap(tickersMapSerialized)

	case common.CONF_OUT:
		log.Println("Eye " + strconv.Itoa(eyeId) + " confirmed out")
		eyeState := eyes[eyeId].EyeState
		switch eyeState {
		case NOT_READY:
			eyes[eyeId].EyeState = OUT_READY
		case IN_READY:
			eyes[eyeId].EyeState = READY
			log.Println("Eye " + strconv.Itoa(eyeId) + " is ready")
		}
	case common.CONF_IN:
		log.Println("Eye " + strconv.Itoa(eyeId) + " confirmed in")
		eyeState := eyes[eyeId].EyeState
		switch eyeState {
		case NOT_READY:
			eyes[eyeId].EyeState = IN_READY
		case OUT_READY:
			eyes[eyeId].EyeState = READY
			log.Println("Eye " + strconv.Itoa(eyeId) + " is ready")
		}
	}
}