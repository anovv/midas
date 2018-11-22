package configuration

import (
	"os"
	"io/ioutil"
	"encoding/json"
	"log"
)

const (
	BRAIN_CONFIG_PATH = "brain_config.json"
	EYE_CONFIG_PATH = "eye_config.json"
)

var brainConfig *BrainConfig
var eyeConfig *EyeConfig

type BrainConfig struct {
	ARB_REPORT_UPDATE_THRESHOLD_MICROS int `json:"arb_report_update_threshold_micros"`
	CONNECTION_RECEIVER_PORT int `json:"connection_receiver_port"`
	BASE_PORT int `json:"base_port"`
	FETCH_DELAYS_MICROS map[string]map[string]int `json:"fetch_delays_micros"` // delays per exchange per command
	MYSQL_PASSWORD string `json:"mysql_password"`
	API_KEY string `json:"api_key"`
	API_SECRET string `json:"api_secret"`
}

type EyeConfig struct {
	BRAIN_CONNECTION_RECEIVER_PORT int `json:"brain_connection_receiver_port"`
	BRAIN_ADDRESS string `json:"brain_address"`
}

func ReadBrainConfig() *BrainConfig {
	if brainConfig != nil {
		log.Println("BrainConfig read")
		return brainConfig
	}
	jsonConfigFile, err := os.Open(BRAIN_CONFIG_PATH)
	defer jsonConfigFile.Close()
	if err != nil {
		panic("Unable to open brain config. Make sure there is brain_config.json next to brain_exec binary: " + err.Error())
	}

	byteValue, _ := ioutil.ReadAll(jsonConfigFile)

	err = json.Unmarshal(byteValue, *brainConfig)

	if err != nil {
		panic("Unable to parse brain config: " + err.Error())
	}

	return brainConfig
}

func ReadEyeConfig() *EyeConfig {
	if eyeConfig != nil {
		log.Println("EyeConfig read")
		return eyeConfig
	}
	jsonConfigFile, err := os.Open(EYE_CONFIG_PATH)
	if err != nil {
		panic("Unable to open eye config. Make sure there is eye_config.json next to eye_exec binary: " + err.Error())
	}

	defer jsonConfigFile.Close()

	byteValue, _ := ioutil.ReadAll(jsonConfigFile)

	err = json.Unmarshal(byteValue, *eyeConfig)

	if err != nil {
		panic("Unable to parse eye config: " + err.Error())
	}

	return eyeConfig
}
