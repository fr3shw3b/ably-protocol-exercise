package utils

// Custom WebSocket close codes.
// https://www.rfc-editor.org/rfc/rfc6455#section-7.4.2
const (
	CloseCodeExpiredSession       int = 4001
	CloseCodeMissingClientID      int = 4002
	CloseCodeInvalidSequenceCount int = 4003
	CloseCodeInvalidLastReceived  int = 4004
)

// Message prefixes.
const (
	NumberInSequencePrefix     uint8 = 0x1
	AcknowledgementPrefix      uint8 = 0x2
	LastNumberInSequencePrefix uint8 = 0x3
)

type SequenceFinalMessage struct {
	Number   uint32 `json:"number"`
	Checksum string `json:"checksum"`
}

func IsKnownClientErrorCode(code int) bool {
	return code == CloseCodeExpiredSession ||
		code == CloseCodeMissingClientID ||
		code == CloseCodeInvalidSequenceCount ||
		code == CloseCodeInvalidLastReceived
}

var codeNameMap = map[int]string{
	CloseCodeExpiredSession:       "CloseCodeExpiredSession",
	CloseCodeMissingClientID:      "CloseCodeMissingClientID",
	CloseCodeInvalidSequenceCount: "CloseCodeInvalidSequenceCount",
	CloseCodeInvalidLastReceived:  "CloseCodeInvalidLastReceived",
}

func CloseCodeName(code int) string {
	name, exists := codeNameMap[code]
	if exists {
		return name
	}
	return "UnknownCode"
}
