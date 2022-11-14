package sessions

import (
	"fmt"
	"sync"
	"time"

	"github.com/sirupsen/logrus"
)

type InMemoryStoreParams struct {
	ExpireAfterIdleTime int
}

func NewInMemoryStore(params *InMemoryStoreParams, logger *logrus.Logger) SessionStore {
	return &inMemoryStore{
		params:   params,
		sessions: map[string]*internalSessionState{},
		logger:   logger,
	}
}

type inMemoryStore struct {
	mu       sync.Mutex
	params   *InMemoryStoreParams
	sessions map[string]*internalSessionState
	logger   *logrus.Logger
}

type internalSessionState struct {
	clientID     string
	sequence     []uint32
	lastAccessed int
	// We hold an expired property as a soft delete property
	// to prevent clients trying to re-connect for the same client ID
	// after an expiry time has passed.
	// This is to distinguish between a client session that has not yet been
	// created and one that has been discarded.
	// In the future a clean up mechanism should be put in place to free up memory
	// as discarded sessions build up over time.
	expired      bool
	nextIndex    int
	acknowledged []bool
	mu           sync.Mutex
}

func (s *inMemoryStore) Initialise(clientID string, sequence []uint32) (SessionState, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	internalSession, err := s.loadExisting(clientID)
	if err != nil {
		return SessionState{}, err
	}

	if internalSession == nil {
		now := int(time.Now().Unix())
		internalSession = &internalSessionState{
			clientID:     clientID,
			sequence:     sequence,
			lastAccessed: now,
			expired:      false,
			nextIndex:    0,
			acknowledged: make([]bool, len(sequence)),
		}
		s.sessions[clientID] = internalSession
	}

	return SessionState{
		Sequence:     internalSession.sequence,
		Acknowledged: internalSession.acknowledged,
	}, nil
}

func (s *inMemoryStore) Get(clientID string) (SessionState, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	internalSession, err := s.loadExisting(clientID)
	if err != nil {
		return SessionState{}, err
	}

	if internalSession == nil {
		return SessionState{}, fmt.Errorf("no session exists for client id (%s)", clientID)
	}

	return SessionState{
		Sequence:     internalSession.sequence,
		Acknowledged: internalSession.acknowledged,
	}, nil
}

func (s *inMemoryStore) Next(clientID string, offsetOverride int, freshConnection bool) (uint32, int, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	session, err := s.loadExisting(clientID)
	if err != nil {
		return 0, 0, err
	}

	session.mu.Lock()
	defer session.mu.Unlock()

	// offset override takes precedence, this is the client provided
	// offset for the index of the number in the sequence it has not
	// yet received.
	if offsetOverride > -1 && offsetOverride < len(session.sequence) {
		s.logger.Debug("choosing offset override")
		session.nextIndex = offsetOverride + 1
		return session.sequence[offsetOverride], offsetOverride, nil
	}

	// For resilience when clients disconnect and reconnect,
	// the first number in the sequence that has not been acknowledged
	// will take priority over the previously set nextIndex.
	// This is primarily to cover the cases where disconnection occurs when
	// the server believes it has sent a message but the client has not received it,
	// this can occur in the time inbetween updating state in the server and sending
	// the message to the client.
	firstNotAcknowledgedIndex := findFirstFalseIndex(session.acknowledged)
	if freshConnection && firstNotAcknowledgedIndex != session.nextIndex &&
		firstNotAcknowledgedIndex > -1 {
		s.logger.Debug("choosing first not acknowledged index, session.nextIndex: ", session.nextIndex, " firstNotAcknowledgedIndex: ", firstNotAcknowledgedIndex)
		number := session.sequence[firstNotAcknowledgedIndex]
		session.nextIndex = firstNotAcknowledgedIndex + 1
		return number, firstNotAcknowledgedIndex, nil
	}

	if session.nextIndex < len(session.sequence) {
		s.logger.Debug("choosing session.nextIndex + 1")
		index := session.nextIndex
		number := session.sequence[index]
		session.nextIndex += 1
		return number, index, nil
	}

	s.logger.Debug("sequence consumed")
	return 0, 0, fmt.Errorf("sequence consumed for session with client id (%s)", clientID)
}

func (s *inMemoryStore) Ack(clientID string, index int) (bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	session, err := s.loadExisting(clientID)
	if err != nil {
		return index == len(session.sequence)-1, err
	}

	session.mu.Lock()
	defer session.mu.Unlock()

	session.acknowledged[index] = true

	return index == len(session.sequence)-1, nil
}

func (s *inMemoryStore) loadExisting(clientID string) (*internalSessionState, error) {
	session := s.sessions[clientID]
	if session != nil {
		expired := s.checkExpiredAndUpdateIfNeeded(session)
		if expired {
			return nil, fmt.Errorf("session has expired for client id (%s)", clientID)
		}

		return session, nil
	}

	// Indicates a session has not yet been created for a given client ID.
	return nil, nil
}

func (s *inMemoryStore) checkExpiredAndUpdateIfNeeded(session *internalSessionState) bool {
	session.mu.Lock()
	defer session.mu.Unlock()

	if session.expired {
		return true
	}

	now := int(time.Now().Unix())

	if session.lastAccessed+s.params.ExpireAfterIdleTime < now {
		s.logger.Debug("Setting session to expired", session.lastAccessed, s.params.ExpireAfterIdleTime, now)
		session.expired = true
	}
	session.lastAccessed = now

	return session.expired
}

// todo: move into a reusable util function.
func findFirstFalseIndex(list []bool) int {
	i := 0
	found := false
	for !found && i < len(list) {
		found = !list[i]
		if !found {
			i += 1
		}
	}

	if found {
		return i
	}
	return -1
}
