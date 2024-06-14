// Copyright (C) 2022 Creditor Corp. Group.
// See LICENSE for copying information.

package reverse

// Bytes takes a bytes as argument and return the reverse of bytes.
func Bytes(value []byte) []byte {
	for i, j := 0, len(value)-1; i < j; i, j = i+1, j-1 {
		value[i], value[j] = value[j], value[i]
	}

	return value
}
