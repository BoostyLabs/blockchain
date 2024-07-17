// Copyright (C) 2024 Creditor Corp. Group.
// See LICENSE for copying information.

package inscriptions

import (
	"encoding/binary"
	"fmt"
	"strconv"
	"strings"

	"github.com/btcsuite/btcd/chaincfg/chainhash"
)

// idSeparator defines separator between TxID and Index in inscription ID.
const idSeparator string = "i"

// ID describes inscription identifier.
type ID struct {
	TxID  *chainhash.Hash // Reveal transaction ID.
	Index uint32          // The index of new inscriptions being inscribed in the reveal transaction.
}

// NewIDFromString parses inscription ID from string.
func NewIDFromString(idStr string) (*ID, error) {
	parts := strings.Split(idStr, idSeparator)
	if len(parts) != 2 {
		return nil, fmt.Errorf("invalid ID format: %s", idStr)
	}

	if len(parts[0]) != chainhash.MaxHashStringSize {
		return nil, fmt.Errorf("invalid TxID format: %s", idStr)
	}

	txID, err := chainhash.NewHashFromStr(parts[0])
	if err != nil {
		return nil, err
	}

	index, err := strconv.ParseUint(parts[1], 10, 32)
	if err != nil {
		return nil, err
	}

	return &ID{TxID: txID, Index: uint32(index)}, nil
}

// NewIDFromDataPush parses inscription ID from script data push.
func NewIDFromDataPush(id []byte) (*ID, error) {
	if len(id) < chainhash.HashSize || len(id) > chainhash.HashSize+4 {
		return nil, fmt.Errorf("invalid TxID format: %x", id)
	}

	txID, err := chainhash.NewHash(id[:chainhash.HashSize])
	if err != nil {
		return nil, err
	}

	var index = make([]byte, 4)
	copy(index, id[chainhash.HashSize:])

	return &ID{TxID: txID, Index: binary.LittleEndian.Uint32(index)}, nil
}

// String returns inscription ID as string.
func (id *ID) String() string {
	return fmt.Sprintf("%s%s%d", id.TxID.String(), idSeparator, id.Index)
}

// IndexLETrailingZerosOmitted returns index as bytes array in little-endian ordering with trailing zeros omitted.
func (id *ID) IndexLETrailingZerosOmitted() []byte {
	data := make([]byte, 4)
	binary.LittleEndian.PutUint32(data, id.Index)
	for lastIdx := 3; lastIdx >= 0; lastIdx-- {
		if data[lastIdx] != 0 {
			return data[:lastIdx+1]
		}
	}

	return []byte{}
}

// IntoDataPush returns ID as bytes for script OP_PUSH.
func (id *ID) IntoDataPush() []byte {
	return append(id.TxID[:], id.IndexLETrailingZerosOmitted()...)
}
