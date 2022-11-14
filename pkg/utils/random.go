package utils

import "math/rand"

func GeneratePseudoRandomSequence(size int, maxNumber uint32) []uint32 {
	sequence := make([]uint32, size)
	for i := range sequence {
		sequence[i] = Uuint32Random(0, maxNumber)
	}
	return sequence
}

func Uuint32Random(min uint32, max uint32) uint32 {
	value := rand.Uint32()
	value %= (max - min)
	value += min
	return value
}
