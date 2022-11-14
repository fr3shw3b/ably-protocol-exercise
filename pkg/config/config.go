package config

import (
	"os"
	"strconv"
)

type Config struct {
	SequenceMessageInterval    int
	SessionStateIdleTimeExpiry int
	LogLevel                   string
}

func Load() (*Config, error) {
	intervalStr, intervalExists := os.LookupEnv("SEQUENCE_MESSAGE_INTERVAL")
	if !intervalExists {
		intervalStr = "1"
	}
	sequenceMessageInterval, err := strconv.Atoi(intervalStr)
	if err != nil {
		return nil, err
	}

	expiryStr, expiryExists := os.LookupEnv("SESSION_STATE_IDLE_TIME_EXPIRY")
	if !expiryExists {
		expiryStr = "30"
	}
	sessionStateIdleTimeExpiry, err := strconv.Atoi(
		expiryStr,
	)
	if err != nil {
		return nil, err
	}

	logLevel, logLevelExists := os.LookupEnv("LOG_LEVEL")
	if !logLevelExists {
		logLevel = "info"
	}

	return &Config{
		SequenceMessageInterval:    sequenceMessageInterval,
		SessionStateIdleTimeExpiry: sessionStateIdleTimeExpiry,
		LogLevel:                   logLevel,
	}, nil
}
