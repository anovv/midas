package logging

import (
	"os"
	"log"
)

const (
	FILE_PATH = "logs.txt"
)

func LogLineToFile(line string) {
	f, err := os.OpenFile(FILE_PATH, os.O_APPEND|os.O_WRONLY|os.O_CREATE, 0600)
	if err != nil {
		log.Println("Can't open logging file: " + err.Error())
		return
	}

	defer f.Close()

	if _, err = f.WriteString(line + "\n"); err != nil {
		log.Println("Can't log to file: " + err.Error())
	}
}