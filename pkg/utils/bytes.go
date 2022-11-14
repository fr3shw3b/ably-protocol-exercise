package utils

import "encoding/binary"

func Uint32ToByteArray(input []uint32) []byte {
	allBytes := []byte{}
	for _, number := range input {
		numberInBytes := make([]byte, 4)
		binary.LittleEndian.PutUint32(numberInBytes, number)
		allBytes = append(allBytes, numberInBytes...)
	}
	return allBytes
}

func ByteArrayToSingleUint32(input []byte) uint32 {
	return binary.LittleEndian.Uint32(input)
}
