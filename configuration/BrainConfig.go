package configuration

import (
	"os"
	"io/ioutil"
	"encoding/json"
)

const BRAIN_CONFIG_PATH = "brain_config.json"

type BrainConfig struct {
	TICKERS_UPDATE_PERIOD_MICROS int `json:"tickers_update_period_micros"`
	ARB_REPORT_UPDATE_THRESHOLD_MICROS int `json:"arb_report_update_threshold_micros"`
}

func ReadBrainConfig() *BrainConfig {
	jsonConfigFile, err := os.Open(BRAIN_CONFIG_PATH)
	if err != nil {
		panic("Unable to open brain config. Make sure there is brain_config.json next to binary: " + err.Error())
	}

	defer jsonConfigFile.Close()

	byteValue, _ := ioutil.ReadAll(jsonConfigFile)

	var brainConfig BrainConfig

	err = json.Unmarshal(byteValue, &brainConfig)

	if err != nil {
		panic("Unable to parse brain config: " + err.Error())
	}

	return &brainConfig
}
