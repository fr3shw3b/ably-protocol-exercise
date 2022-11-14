package clientapp

import (
	"fmt"
	"log"

	"github.com/fr3shw3b/ably-protocol-exercise/pkg/client"
	"github.com/fr3shw3b/ably-protocol-exercise/pkg/config"
	"github.com/joho/godotenv"
	"github.com/sirupsen/logrus"
)

func Run(serverHost string, serverPort int, sequenceCount int) error {
	err := godotenv.Load(".env.client")
	if err != nil {
		log.Fatal("Failed to load environment variables: ", err)
	}

	conf, err := config.LoadForClient()
	if err != nil {
		log.Fatal("Failed to load configuration for client: ", err)
	}

	customFormatter := new(logrus.TextFormatter)
	customFormatter.TimestampFormat = "2006-01-02T15:04:05.999999999Z07:00"
	customFormatter.FullTimestamp = true
	logger := logrus.New()
	logger.SetFormatter(customFormatter)
	logLevel, err := logrus.ParseLevel(conf.LogLevel)
	if err != nil {
		logLevel = logrus.InfoLevel
	}
	logger.SetLevel(logLevel)

	// The client implementation is currently limited to run as a one-off client-side
	// connection/session, in the future this could be expanded to manage multiple connections
	// with a single client implementation.
	clientInstance := client.NewDefaultClient(
		&client.ClientParams{
			ServerHost:            serverHost,
			ServerPort:            serverPort,
			SequenceCount:         sequenceCount,
			SendLastReceivedIndex: conf.SendLastReceivedIndex,
			MaxReconnectAttempts:  conf.MaxReconnectAttempts,
		},
		logger,
	)

	err = clientInstance.Connect()
	if err != nil {
		return err
	}
	defer clientInstance.Close()

	result := clientInstance.Result()
	printResult(result)
	return nil
}

func printResult(result client.Result) {
	fmt.Print("Result\n____________\n\n\n")
	fmt.Printf("Client-side Checksum: %s\n", result.Checksum)
	fmt.Printf("Server-provided Checksum: %s\n", result.ServerChecksum)
	fmt.Printf("Successful: %v\n", result.Success)
	if result.Error != nil {
		fmt.Printf("Error: %s\n", result.Error)
	}
}
