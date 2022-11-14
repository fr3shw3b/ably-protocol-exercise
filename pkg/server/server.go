package server

import (
	"encoding/binary"
	"encoding/json"
	"errors"
	"math/rand"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/fr3shw3b/ably-protocol-exercise/pkg/sessions"
	"github.com/fr3shw3b/ably-protocol-exercise/pkg/utils"
	"github.com/gorilla/websocket"
	"github.com/sirupsen/logrus"
)

type ServerParams struct {
	SequenceMessageInterval int
}

const (
	MaxSequenceNumberValue uint32 = 0xffff
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		// No need for strict CORS checking for this implementation.
		return true
	},
}

type serverImpl struct {
	params *ServerParams
	store  sessions.SessionStore
	logger *logrus.Logger
}

func NewDefaultServer(params *ServerParams, store sessions.SessionStore, logger *logrus.Logger) http.Handler {
	return &serverImpl{
		params,
		store,
		logger,
	}
}

func (s *serverImpl) ServeHTTP(w http.ResponseWriter, r *http.Request) {

	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		s.logger.Error("websockets upgrade error: ", err)
		return
	}

	defer conn.Close()

	query := r.URL.Query()
	clientID := query.Get("clientId")
	if clientID == "" {
		conn.WriteControl(
			websocket.CloseMessage,
			websocket.FormatCloseMessage(utils.CloseCodeMissingClientID, "missing client id"),
			// This deadline could be made configurable.
			time.Now().Add(1*time.Second),
		)
		conn.Close()
		return
	}

	sequenceCountStr := query.Get("sequenceCount")
	sequenceCount, err := deriveSequenceCount(sequenceCountStr)
	if err != nil {
		s.logger.Error("Failed to parse sequenceCount: ", err)
		conn.WriteControl(
			websocket.CloseMessage,
			websocket.FormatCloseMessage(
				utils.CloseCodeInvalidSequenceCount,
				"sequence count must be an integer less than or equal to 0xffff",
			),
			// This deadline could be made configurable.
			time.Now().Add(1*time.Second),
		)
		conn.Close()
		return
	}

	lastReceivedIndexStr := query.Get("lastReceived")
	lastReceived, err := deriveLastReceivedIndex(lastReceivedIndexStr)
	if err != nil {
		s.logger.Error("Failed to parse lastReceived: ", err)
		conn.WriteControl(
			websocket.CloseMessage,
			websocket.FormatCloseMessage(
				utils.CloseCodeInvalidLastReceived,
				"if provided, last received index must be an integer less than or equal to 0xffff",
			),
			// This deadline could be made configurable.
			time.Now().Add(1*time.Second),
		)
		conn.Close()
		return
	}

	// An improvement here could be to first check if a session exists before
	// creating the pseudo-random sequence of numbers.
	sequence := utils.GeneratePseudoRandomSequence(sequenceCount, MaxSequenceNumberValue)
	// If a session exists for the given client id, the sequence provided
	// here will be ignored.
	session, err := s.store.Initialise(clientID, sequence)
	if err != nil {
		s.logger.Error("Failed to initialise session: ", err)
		expiredSession := isExpiredSessionError(err.Error())
		if expiredSession {
			conn.WriteControl(
				websocket.CloseMessage,
				websocket.FormatCloseMessage(
					utils.CloseCodeExpiredSession,
					"session has expired",
				),
				// This deadline could be made configurable.
				time.Now().Add(1*time.Second),
			)
		}
		conn.Close()
	}

	go s.initSequence(conn, clientID, session, lastReceived)

	for {
		_, message, err := conn.ReadMessage()
		if err != nil {
			s.logger.Error("read error:", err)
			break
		}
		s.handleMessage(message, clientID, conn)
	}

}

func (s *serverImpl) initSequence(conn *websocket.Conn, clientID string, session sessions.SessionState, lastReceivedIndex int) {
	next, index, err := s.store.Next(clientID, lastReceivedIndex, true)
	for !isSequenceConsumedError(err) {
		s.logger.Debug("client: ", clientID, " next: ", next, " index: ", index, " error: ", err)
		msg, innerErr := prepareMessage(session, next, index)
		if innerErr != nil {
			// todo: implement a mechanism that handles these errors better.
			s.logger.Error("prepare message error: ", err)
		} else {
			conn.WriteMessage(websocket.BinaryMessage, msg)
		}

		// Only pauses the current goroutine!
		time.Sleep(time.Millisecond * time.Duration(s.params.SequenceMessageInterval))

		next, index, err = s.store.Next(clientID, -1, false)
	}
}

func (s *serverImpl) handleMessage(message []byte, clientID string, conn *websocket.Conn) {
	if message[0] == utils.AcknowledgementPrefix {
		index := binary.LittleEndian.Uint32(message[1:])
		s.logger.Debug("Received index:", message[1:], index, int(index))
		final, err := s.store.Ack(clientID, int(index))
		if err != nil {
			s.logger.Error("failed to persist client acknowledgement: ", err)
		}

		if final {
			conn.WriteControl(
				websocket.CloseMessage,
				websocket.FormatCloseMessage(
					websocket.CloseNormalClosure,
					"sequence complete",
				),
				// This deadline could be made configurable.
				time.Now().Add(1*time.Second),
			)
			conn.Close()
		}
	}
}

func prepareMessage(session sessions.SessionState, next uint32, index int) ([]byte, error) {
	if index < len(session.Sequence)-1 {
		numberInBytes := utils.Uint32ToByteArray([]uint32{next})
		return append([]byte{utils.NumberInSequencePrefix}, numberInBytes...), nil
	}

	finalMessage := utils.SequenceFinalMessage{
		Number:   next,
		Checksum: utils.CreateChecksum(session.Sequence),
	}
	messageBytes, err := json.Marshal(&finalMessage)
	if err != nil {
		return nil, err
	}
	return append([]byte{utils.LastNumberInSequencePrefix}, messageBytes...), nil
}

func isExpiredSessionError(errMessage string) bool {
	// todo: make this cleaner by using custom error structs with custom code
	// properties.
	return strings.HasPrefix(errMessage, "session has expired for client id")
}

func isSequenceConsumedError(err error) bool {
	if err == nil {
		return false
	}

	// todo: make this cleaner by using custom error structs with custom code
	// properties.
	return strings.HasPrefix(err.Error(), "sequence consumed for session with client id")
}

func deriveSequenceCount(queryParam string) (int, error) {
	if queryParam == "" {
		return rand.Intn(int(MaxSequenceNumberValue)), nil
	}

	sequenceCount, err := strconv.Atoi(queryParam)
	if err != nil {
		return 0, err
	}
	if sequenceCount > int(MaxSequenceNumberValue) {
		return 0, errors.New("sequenceCount must be less than or equal to 0xffff")
	}
	return sequenceCount, nil
}

func deriveLastReceivedIndex(queryParam string) (int, error) {
	if queryParam == "" {
		return -1, nil
	}
	lastReceivedIndex, err := strconv.Atoi(queryParam)
	if err != nil {
		return 0, err
	}
	if lastReceivedIndex > int(MaxSequenceNumberValue) {
		return 0, errors.New("lastReceivedIndex must be less than or equal to 0xffff")
	}
	return lastReceivedIndex, nil
}
