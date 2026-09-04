package config

import (
	"os"
	"strconv"
	"time"
)

type Config struct {
	AMQPURL      string
	MgmtURL      string
	ConsoleURL   string
	Prefetch     int
	WriteLogWait time.Duration
}

func Load() Config {
	prefetch, _ := strconv.Atoi(os.Getenv("PREFETCH"))
	writeLogWait, _ := strconv.Atoi(os.Getenv("WRITE_LOG_WAIT"))
	return Config{
		AMQPURL:      os.Getenv("AMQP_URL"),
		MgmtURL:      os.Getenv("RABBIT_MGMT_URL"),
		ConsoleURL:   os.Getenv("CONSOLE_URL"),
		Prefetch:     prefetch,
		WriteLogWait: time.Duration(writeLogWait) * time.Second,
	}
}
