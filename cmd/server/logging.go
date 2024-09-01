package main

import (
	"io"
	"log"
	"os"
)

const LOG_FILE = "simple-rtmp-restreamer.log"

func setupLogger() {
	logFile, err := os.OpenFile(LOG_FILE, os.O_APPEND|os.O_RDWR|os.O_CREATE, 0644)
	if err != nil {
		log.Panic(err)
	}
	mw := io.MultiWriter(os.Stdout, logFile)

	log.SetFlags(log.LstdFlags)
	log.SetOutput(mw)
}
