package logger

import (
	"io"
	"log"
	"os"
	"thuhole-go-backend/pkg/consts"
)

func InitLog() {
	logFile, err := os.OpenFile(consts.LogFile, os.O_CREATE|os.O_APPEND|os.O_RDWR, 0666)
	if err != nil {
		panic(err)
	}
	mw := io.MultiWriter(os.Stdout, logFile)
	log.SetOutput(mw)
}
