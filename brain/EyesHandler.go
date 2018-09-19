package brain

import (
	. "github.com/pebbe/zmq4"
	"fmt"
	"strconv"
	. "midas/common"
	"time"
	//"encoding/json"
	"encoding/json"
)

const (
	INPUT_BUFFER_SIZE = 10000
	CONNECTION_RECEIVER_PORT = 5500
	BASE_PORT = 5501
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
	SocketIn *Socket
	SocketOut *Socket
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

var eyes = make(map[int]*EyeHandle)
var lastConnectedEyeId = -1

var pairsPerExchange = map[string][]string{
	BINANCE: {"ETHBTC"},
}

var avgFetchTimePerExchange = map[string]*AvgFetchTimeStat{
	BINANCE: {
		0,
		0,
	},
}

var rateLimitDelaysPerExchange = map[string]int{
	BINANCE: int(0.05 * 1000 * 1000), // limit = 1200 reqs/min, delay in microseconds
}

func ScheduleUpdates() {
		for exchange, pairList := range pairsPerExchange {
			go func() {
				var pairIndex = 0
				for {
					for eyeId, eyeHandle := range eyes {
						pair := pairList[pairIndex]
						delay := getDelayMicroSeconds(exchange, pair, eyeId)
						message := Message{
							DEPTH_REQ,
							map[string]string{
								CURRENCY_PAIR: pair,
								EXCHANGE: exchange,
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

func getDelayMicroSeconds(exchange string, pair string, eyeId int) int {
	numEyes := len(eyes)
	numPairs := len(pairsPerExchange[exchange])
	if numEyes <= numPairs {
		return rateLimitDelaysPerExchange[exchange]
	}

	avgFetchTime := avgFetchTimePerExchange[exchange].AvgFetchTimeMicroSeconds

	return rateLimitDelaysPerExchange[exchange] + int(avgFetchTime * numPairs/numEyes)
}

func SetupRequestReceiver() {
	requestReceiver, _ := NewSocket(REP)
	requestReceiver.Bind(TCP_PREFIX + strconv.Itoa(CONNECTION_RECEIVER_PORT))
	for {
		request, error := requestReceiver.Recv(0)
		if error != nil {
			fmt.Println("Receiver error error: " + error.Error())
			continue
		}
		handleNewConnectionRequest(requestReceiver, request)
	}
}

func handleNewConnectionRequest(requestReceiver *Socket, requestSerialized string) {
	request := DeserializeMessage(requestSerialized)
	if request.Command != CONNECT_EYE {
		fmt.Println("Bad connection request, aborting...")
		return
	}
 	fmt.Println("New connection request")

	eyeId := lastConnectedEyeId + 1
	var portIn int
	if eyeId == 0 {
		// first connection
		portIn = BASE_PORT
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
	message := Message{
		CONFIRM_PORTS,
		map[string]string{
			PORT_IN: strconv.Itoa(portIn),
			PORT_OUT: strconv.Itoa(portOut),
			},
	}
	requestReceiver.Send(message.SerializeMessage(), 0)
	lastConnectedEyeId = eyeId
}

func CleanupEyesHandler() {
	for eyeId := range eyes {
		eyeInterface := eyes[eyeId]
		message := Message{
			KILL_EYE,
			nil,
		}
		*eyeInterface.ChannelIn<-message.SerializeMessage()
		// TODO block until channel is empty
		eyeInterface.SocketIn.Close()
		eyeInterface.SocketOut.Close()
	}
}

func setupOutSocket(portOut int, eyeId int) *Socket {
	out, _ := NewSocket(PULL)
	out.Bind(TCP_PREFIX + strconv.Itoa(portOut))
	fmt.Println("Waiting for eye " + strconv.Itoa(eyeId) + " to confirm out connection on port " + strconv.Itoa(portOut))
	go func(){
		for {
			msg, error := out.Recv(0)
			if error != nil {
				fmt.Println("Out error: " + error.Error())
				continue
			}
			go func() {
				handleMessage(msg, eyeId)
			} ()
		}
	}()

	return out
}

func setupInSocket(portIn int, eyeId int, channelIn *chan string) *Socket {
	in, _ := NewSocket(PUSH)
	in.Bind(TCP_PREFIX + strconv.Itoa(portIn))
	fmt.Println("Waiting for eye " + strconv.Itoa(eyeId) + " to confirm in connection on port " + strconv.Itoa(portIn))
	go func(){
		for {
			msg := <-*channelIn
			_ , error := in.Send(msg,0)
			if error != nil {
				fmt.Println("In error: " + error.Error())
			}
		}
	}()

	return in
}

func handleMessage(messageSerialized string, eyeId int) {
	message := DeserializeMessage(messageSerialized)
	command := message.Command
	args := message.Args
	switch command {
	case DEPTH_RESP:
		depthSerialized := args[DEPTH_SERIALIZED]
		depth := DeserializeDepth(depthSerialized)
		dj, _ := json.Marshal(depth)
		fmt.Println("Received depth update: " + string(dj))
		fetchTimeMicroSeconds, _ := strconv.Atoi(args[FETCH_TIME_MICROSECONDS])
		exchange := args[EXCHANGE]
		// Update avg fetch
		// TODO make atomic/sync
		avgFetchTimeStat := avgFetchTimePerExchange[exchange]
		// TODO Math could be wrong here (check what "/" does)
		newAvg := int((avgFetchTimeStat.AvgFetchTimeMicroSeconds * avgFetchTimeStat.NumSamples + fetchTimeMicroSeconds)/(avgFetchTimeStat.NumSamples + 1))
		avgFetchTimeStat.AvgFetchTimeMicroSeconds = newAvg
		avgFetchTimeStat.NumSamples = avgFetchTimeStat.NumSamples + 1

	case CONF_OUT:
		fmt.Println("Eye " + strconv.Itoa(eyeId) + " confirmed out")
		eyeState := eyes[eyeId].EyeState
		switch eyeState {
		case NOT_READY:
			eyes[eyeId].EyeState = OUT_READY
		case IN_READY:
			eyes[eyeId].EyeState = READY
			fmt.Println("Eye " + strconv.Itoa(eyeId) + " is ready")
		}
	case CONF_IN:
		fmt.Println("Eye " + strconv.Itoa(eyeId) + " confirmed in")
		eyeState := eyes[eyeId].EyeState
		switch eyeState {
		case NOT_READY:
			eyes[eyeId].EyeState = IN_READY
		case OUT_READY:
			eyes[eyeId].EyeState = READY
			fmt.Println("Eye " + strconv.Itoa(eyeId) + " is ready")
		}
	}
}