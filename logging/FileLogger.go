package logging

import (
	"os"
	"log"
)

const (
	LOGS_FILE_PATH = "logs.txt"
	ARB_STATES_FILE_PATH = "arb_states.txt"
)

func LogLineToFile(line string, file string) {
	f, err := os.OpenFile(file, os.O_APPEND|os.O_WRONLY|os.O_CREATE, 0600)
	if err != nil {
		log.Println("Can't open logging file: " + err.Error())
		return
	}

	defer f.Close()

	if _, err = f.WriteString(line + "\n"); err != nil {
		log.Println("Can't log to file: " + err.Error())
	}
}