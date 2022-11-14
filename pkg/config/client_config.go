package config

import (
	"os"
	"strconv"
)

type ClientConfig struct {
	SendLastReceivedIndex bool
	MaxReconnectAttempts  int
	LogLevel              string
}

func LoadForClient() (*ClientConfig, error) {
	lastReceivedStr, lastReceivedExists := os.LookupEnv("SEND_LAST_RECEIVED_INDEX")
	if !lastReceivedExists {
		lastReceivedStr = "false"
	}
	sendLastReceived, err := strconv.ParseBool(lastReceivedStr)
	if err != nil {
		return nil, err
	}

	maxReconnectStr, maxReconnectExists := os.LookupEnv("MAX_RECONNECTION_ATTEMPTS")
	if !maxReconnectExists {
		maxReconnectStr = "100"
	}
	maxReconnectAttempts, err := strconv.Atoi(
		maxReconnectStr,
	)
	if err != nil {
		return nil, err
	}

	logLevel, logLevelExists := os.LookupEnv("LOG_LEVEL")
	if !logLevelExists {
		logLevel = "info"
	}

	return &ClientConfig{
		SendLastReceivedIndex: sendLastReceived,
		MaxReconnectAttempts:  maxReconnectAttempts,
		LogLevel:              logLevel,
	}, nil
}
