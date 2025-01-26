package utils

import (
	"log"
	"os"
)

var logger = log.New(os.Stdout, "LOG: ", log.LstdFlags)

func LogInfo(message string, v ...any) {
	logger.Printf("[INFO] "+message, v...)
}

func LogError(err error) {
	logger.Println("[ERROR]", err)
}

func LogErrorStr(err string) {
	logger.Println("[ERROR]", err)
}
