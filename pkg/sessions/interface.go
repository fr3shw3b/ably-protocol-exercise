package sessions

type SessionStore interface {
	// Initialises a session and returns a read-only copy of
	// session state.
	Initialise(clientID string, sequence []uint32) (SessionState, error)
	// Should produce a read-only copy of session state.
	Get(clientID string) (SessionState, error)
	// Gets the next number in the sequence to send to the client.
	Next(clientID string, offsetOverride int, freshConnection bool) (uint32, int, error)
	// Registers an acknowledgement from the client for a given index
	// in the sequence.
	// The first return value is whether or not the acknowledged
	// index is the final one in the sequence.
	Ack(clientID string, index int) (bool, error)
}

type SessionState struct {
	Sequence     []uint32
	Acknowledged []bool
}
