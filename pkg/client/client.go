package client

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"strconv"
	"sync"
	"time"

	"github.com/cenkalti/backoff/v4"
	"github.com/fr3shw3b/ably-protocol-exercise/pkg/utils"
	"github.com/google/uuid"
	"github.com/gorilla/websocket"
	"github.com/sirupsen/logrus"
)

type Result struct {
	Checksum       string
	ServerChecksum string
	Success        bool
	Error          error
}

type ClientParams struct {
	ServerHost            string
	ServerPort            int
	SendLastReceivedIndex bool
	MaxReconnectAttempts  int
	SequenceCount         int
	// The following are primarily for providing a programmable interface
	// for automated tests to simulate failure.
	// These are references to allow for nil checks
	// so we can skip while allowing an empty string
	// or 0 as valid inputs.
	OverrideClientID          *string
	OverrideLastReceivedIndex *int
}

type clientImpl struct {
	params   *ClientParams
	session  *sessionState
	wsClient *websocket.Conn
	logger   *logrus.Logger
}

type sessionState struct {
	clientID                 string
	sequenceReceived         []uint32
	lastReceivedIndex        int
	receivedCompleteSequence bool
	success                  bool
	finalErr                 error
	serverChecksum           string
	mu                       sync.Mutex
}

func NewDefaultClient(params *ClientParams, logger *logrus.Logger) Client {
	return &clientImpl{params: params, session: &sessionState{
		// Ensure we initialise last received as -1, otherwise it will be 0
		// which is the default empty value and therefore the first message will be skipped.
		lastReceivedIndex: -1,
	}, wsClient: nil, logger: logger}
}

func (c *clientImpl) Connect() error {
	id := uuid.New()
	if c.params.OverrideClientID != nil {
		c.session.clientID = *c.params.OverrideClientID
	} else {
		c.session.clientID = id.String()
	}

	return c.connect()
}

func (c *clientImpl) connect() error {

	err := backoff.Retry(c.retryConnect, backoff.WithMaxRetries(
		backoff.NewExponentialBackOff(),
		uint64(c.params.MaxReconnectAttempts),
	))
	if err != nil {
		return err
	}

	go c.handleMessages()
	return nil
}

func (c *clientImpl) handleMessages() {
	for c.session.finalErr == nil && !c.session.receivedCompleteSequence {
		_, message, err := c.wsClient.ReadMessage()
		if err != nil {
			c.logger.Debug("read message error: ", err)
			c.wsClient.Close()
			break
		} else {
			c.handleMessage(message)
		}
	}
}

func (c *clientImpl) handleMessage(message []byte) {
	c.logger.Debug("Received message: ", message)
	if message[0] == utils.NumberInSequencePrefix {
		c.handleMessageInSequence(message[1:])
	} else if message[0] == utils.LastNumberInSequencePrefix {
		c.handleLastMessageInSequence(message[1:])
	}
}

func (c *clientImpl) handleMessageInSequence(message []byte) {
	c.session.mu.Lock()
	defer c.session.mu.Unlock()

	sequenceNumber := utils.ByteArrayToSingleUint32(message)
	c.session.sequenceReceived = append(c.session.sequenceReceived, sequenceNumber)
	newIndex := len(c.session.sequenceReceived) - 1
	c.session.lastReceivedIndex = newIndex
	c.logger.Debug(
		"len sequence received: ", len(c.session.sequenceReceived), " uint32: ", uint32(len(c.session.sequenceReceived)-1),
	)
	c.wsClient.WriteMessage(websocket.BinaryMessage, append(
		[]byte{utils.AcknowledgementPrefix},
		utils.Uint32ToByteArray([]uint32{uint32(newIndex)})...,
	))
}

func (c *clientImpl) handleLastMessageInSequence(message []byte) {
	c.session.mu.Lock()
	defer c.session.mu.Unlock()

	finalMessage := &utils.SequenceFinalMessage{}
	err := json.Unmarshal(message, finalMessage)
	// Failure to parse the last message should be deemed one of the
	// possible final errors.
	if err != nil {
		c.session.finalErr = err
		c.session.success = false
		return
	}

	c.session.sequenceReceived = append(c.session.sequenceReceived, finalMessage.Number)
	clientChecksum := utils.CreateChecksum(c.session.sequenceReceived)
	if clientChecksum != finalMessage.Checksum {
		c.session.finalErr = fmt.Errorf(
			"client checksum %s does not match one from server %s",
			clientChecksum,
			finalMessage.Checksum,
		)
		c.session.success = false
	} else {
		c.session.success = true
	}
	newIndex := len(c.session.sequenceReceived) - 1
	c.session.lastReceivedIndex = newIndex
	c.session.serverChecksum = finalMessage.Checksum
	c.session.receivedCompleteSequence = true

	c.logger.Debug(
		"len sequence received: ", len(c.session.sequenceReceived), " uint32: ", uint32(len(c.session.sequenceReceived)-1),
	)
	// Perhaps this isn't necessary as the server will be closing after sending
	// the final number in the sequence with the checksum.
	c.wsClient.WriteMessage(websocket.BinaryMessage, append(
		[]byte{utils.AcknowledgementPrefix},
		utils.Uint32ToByteArray([]uint32{uint32(newIndex)})...,
	))
}

func (c *clientImpl) retryConnect() error {
	// todo: support TLS.
	url := c.buildUrl()

	wsClient, _, err := websocket.DefaultDialer.Dial(url, nil)
	if err != nil {
		return err
	}
	wsClient.SetCloseHandler(c.closeHandler)
	c.wsClient = wsClient
	return nil
}

func (c *clientImpl) closeHandler(code int, text string) error {
	c.session.mu.Lock()
	defer c.session.mu.Unlock()

	// Implement the default close handler behaviour and then try to reconnect
	// if not complete and the connection was not closed due to known client issues.
	message := websocket.FormatCloseMessage(code, "")
	c.wsClient.WriteControl(websocket.CloseMessage, message, time.Now().Add(time.Second))

	// We only try to reconnect on unexpected closures before the full sequence has
	// been received by the client.
	finishedProcessing := c.session.finalErr == nil && !c.session.receivedCompleteSequence
	if !utils.IsKnownClientErrorCode(code) && finishedProcessing && text != "sequence complete" {
		// Do not let retrying the connection block the close handler,
		// we need to free up the WebSocket connection to complete clean up.
		go c.connect()
	}

	if utils.IsKnownClientErrorCode(code) {
		c.session.finalErr = fmt.Errorf(
			"client error: code[%s(%d)] reason: %s",
			utils.CloseCodeName(code),
			code,
			text,
		)
	}

	return nil
}

func (c *clientImpl) buildUrl() string {
	c.session.mu.Lock()
	defer c.session.mu.Unlock()

	q := url.Values{
		"clientId": {c.session.clientID},
	}
	if c.params.SequenceCount > -1 {
		q.Set("sequenceCount", strconv.Itoa(c.params.SequenceCount))
	}
	if c.params.SendLastReceivedIndex && c.session.lastReceivedIndex > -1 {
		q.Set("lastReceived", strconv.Itoa(c.session.lastReceivedIndex))
	} else if c.params.SendLastReceivedIndex && c.params.OverrideLastReceivedIndex != nil {
		q.Set("lastReceived", strconv.Itoa(*c.params.OverrideLastReceivedIndex))
	}

	url := url.URL{
		// todo: support TLS.
		Scheme:   "ws",
		Host:     fmt.Sprintf("%s:%d", c.params.ServerHost, c.params.ServerPort),
		RawQuery: q.Encode(),
	}
	return url.String()
}

func (c *clientImpl) Close() error {
	return c.wsClient.Close()
}

func (c *clientImpl) Result() Result {
	// Make deadline configurable.
	deadline := time.Now().Add(300 * time.Second) // 5 minutes
	// todo: improve communication by using channels.
	for {
		if c.session.finalErr != nil || c.session.receivedCompleteSequence || time.Now().After(deadline) {
			break
		}
	}

	timedOut := c.session.finalErr == nil && !c.session.receivedCompleteSequence
	if timedOut {
		return Result{Error: errors.New("timed out after 300 seconds waiting to receive full sequence")}
	}

	return Result{
		Checksum:       utils.CreateChecksum(c.session.sequenceReceived),
		ServerChecksum: c.session.serverChecksum,
		Error:          c.session.finalErr,
		Success:        c.session.success,
	}
}
