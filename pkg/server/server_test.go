package server

import (
	"net/http/httptest"
	"net/url"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/fr3shw3b/ably-protocol-exercise/pkg/client"
	"github.com/fr3shw3b/ably-protocol-exercise/pkg/sessions"
	"github.com/sirupsen/logrus"
)

func Test_server_produces_sequence_of_numbers_and_client_processes_them_successfully(t *testing.T) {

	logger := createLogger()

	server := createTestServer()
	defer server.Close()

	serverURL, err := url.Parse(server.URL)
	if err != nil {
		t.Error(err)
		t.FailNow()
	}
	host := serverURL.Hostname()
	port, _ := strconv.Atoi(serverURL.Port())

	clientParams := &client.ClientParams{
		ServerHost:            host,
		ServerPort:            port,
		SendLastReceivedIndex: true,
		MaxReconnectAttempts:  100,
		SequenceCount:         200,
	}
	client := client.NewDefaultClient(clientParams, logger)
	err = client.Connect()
	if err != nil {
		t.Error(err)
		t.FailNow()
	}

	result := client.Result()
	if result.Error != nil {
		t.Error("result contained error: ", result.Error)
		t.FailNow()
	}

	if !result.Success {
		t.Error("did not succeed, result.Success was false")
		t.FailNow()
	}

	if result.Checksum != result.ServerChecksum {
		t.Error("expected checksums from client and server to match")
	}
}

func Test_failure_due_to_missing_client_id(t *testing.T) {
	logger := createLogger()

	server := createTestServer()
	defer server.Close()

	serverURL, err := url.Parse(server.URL)
	if err != nil {
		t.Error(err)
		t.FailNow()
	}
	host := serverURL.Hostname()
	port, _ := strconv.Atoi(serverURL.Port())

	overrideClientID := ""
	clientParams := &client.ClientParams{
		ServerHost:            host,
		ServerPort:            port,
		SendLastReceivedIndex: true,
		MaxReconnectAttempts:  100,
		SequenceCount:         200,
		OverrideClientID:      &overrideClientID,
	}
	client := client.NewDefaultClient(clientParams, logger)
	err = client.Connect()
	if err != nil {
		t.Error(err)
		t.FailNow()
	}

	result := client.Result()
	if result.Error == nil {
		t.Error("result does not contain an error when one was expected")
		t.FailNow()
	}

	if !strings.HasSuffix(result.Error.Error(), "code[CloseCodeMissingClientID(4002)] reason: missing client id") {
		t.Error("expected error to be a 4002 missing client id but received: ", result.Error)
	}

	if result.Success {
		t.Error("expected result.Success to be false, received true")
		t.FailNow()
	}
}

func Test_failure_due_to_invalid_sequence_count(t *testing.T) {
	logger := createLogger()

	server := createTestServer()
	defer server.Close()

	serverURL, err := url.Parse(server.URL)
	if err != nil {
		t.Error(err)
		t.FailNow()
	}
	host := serverURL.Hostname()
	port, _ := strconv.Atoi(serverURL.Port())

	clientParams := &client.ClientParams{
		ServerHost:            host,
		ServerPort:            port,
		SendLastReceivedIndex: true,
		MaxReconnectAttempts:  100,
		// Max size for sequence count is 0xffff.
		SequenceCount: 0xffff1,
	}
	client := client.NewDefaultClient(clientParams, logger)
	err = client.Connect()
	if err != nil {
		t.Error(err)
		t.FailNow()
	}

	result := client.Result()
	if result.Error == nil {
		t.Error("result does not contain an error when one was expected")
		t.FailNow()
	}

	if !strings.HasSuffix(
		result.Error.Error(),
		"code[CloseCodeInvalidSequenceCount(4003)] reason: "+
			"sequence count must be an integer less than or equal to 0xffff",
	) {
		t.Error("expected error to be a 4003 invalid sequence count but received: ", result.Error)
	}

	if result.Success {
		t.Error("expected result.Success to be false, received true")
		t.FailNow()
	}
}

func Test_failure_due_to_invalid_last_received_index(t *testing.T) {
	logger := createLogger()

	server := createTestServer()
	defer server.Close()

	serverURL, err := url.Parse(server.URL)
	if err != nil {
		t.Error(err)
		t.FailNow()
	}
	host := serverURL.Hostname()
	port, _ := strconv.Atoi(serverURL.Port())

	// Max size for last received index is 0xffff.
	overrideLastReceivedIndex := 0xffff2
	clientParams := &client.ClientParams{
		ServerHost:                host,
		ServerPort:                port,
		SendLastReceivedIndex:     true,
		MaxReconnectAttempts:      100,
		SequenceCount:             200,
		OverrideLastReceivedIndex: &overrideLastReceivedIndex,
	}
	client := client.NewDefaultClient(clientParams, logger)
	err = client.Connect()
	if err != nil {
		t.Error(err)
		t.FailNow()
	}

	result := client.Result()
	if result.Error == nil {
		t.Error("result does not contain an error when one was expected")
		t.FailNow()
	}

	if !strings.HasSuffix(
		result.Error.Error(),
		"code[CloseCodeInvalidLastReceived(4004)] reason: if provided, "+
			"last received index must be an integer less than or equal to 0xffff",
	) {
		t.Error("expected error to be a 4003 invalid sequence count but received: ", result.Error)
	}

	if result.Success {
		t.Error("expected result.Success to be false, received true")
		t.FailNow()
	}
}

func Test_server_handles_concurrent_clients(t *testing.T) {
	logger := createLogger()

	server := createTestServer()
	defer server.Close()

	serverURL, err := url.Parse(server.URL)
	if err != nil {
		t.Error(err)
		t.FailNow()
	}
	host := serverURL.Hostname()
	port, _ := strconv.Atoi(serverURL.Port())

	resultChan := make(chan client.Result, 60)
	for i := 0; i < 30; i += 1 {
		go func(outputChan chan client.Result) {
			clientParams := &client.ClientParams{
				ServerHost:            host,
				ServerPort:            port,
				SendLastReceivedIndex: true,
				MaxReconnectAttempts:  100,
				SequenceCount:         200,
			}
			client := client.NewDefaultClient(clientParams, logger)
			err = client.Connect()
			if err != nil {
				t.Error(err)
			}

			result := client.Result()
			outputChan <- result
		}(resultChan)
	}

	collectedResults := []client.Result{}
	for len(collectedResults) < 30 {
		select {
		case result := <-resultChan:
			collectedResults = append(collectedResults, result)
		case <-time.After(60 * time.Second):
			t.Error("timed out waiting for result from concurrent clients")
			t.FailNow()
		}
	}

	for i := 0; i < 30; i += 1 {
		result := collectedResults[i]
		if result.Error != nil {
			t.Error("result contained error: ", result.Error)
			t.FailNow()
		}

		if !result.Success {
			t.Error("did not succeed, result.Success was false")
			t.FailNow()
		}

		if result.Checksum != result.ServerChecksum {
			t.Error("expected checksums from client and server to match")
		}
	}
}

func createTestServer() *httptest.Server {
	logger := createLogger()

	storeParams := &sessions.InMemoryStoreParams{
		ExpireAfterIdleTime: 30,
	}
	store := sessions.NewInMemoryStore(storeParams, logger)
	serverParams := &ServerParams{
		// 5 milliseconds interval to send each number
		// in the sequence to speed up tests.
		SequenceMessageInterval: 5,
	}
	server := NewDefaultServer(serverParams, store, logger)

	return httptest.NewServer(server)
}

func createLogger() *logrus.Logger {
	customFormatter := new(logrus.TextFormatter)
	customFormatter.TimestampFormat = "2006-01-02T15:04:05.999999999Z07:00"
	customFormatter.FullTimestamp = true
	logger := logrus.New()
	logger.SetLevel(logrus.DebugLevel)
	logger.SetFormatter(customFormatter)
	return logger
}
