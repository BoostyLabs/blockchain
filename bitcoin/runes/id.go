// Copyright (C) 2024 Creditor Corp. Group.
// See LICENSE for copying information.

package runes

import (
	"fmt"
	"math/big"
	"strconv"
	"strings"
)

// RuneID defined the id of the rune.
type RuneID struct {
	Block uint64
	TxID  uint32
}

// NewRuneIDFromString returns RuneID parsed from string.
func NewRuneIDFromString(s string) (RuneID, error) {
	data := strings.Split(s, ":")
	if len(data) != 2 {
		return RuneID{}, fmt.Errorf("invalid rune id format: %s", s)
	}

	block, err := strconv.ParseUint(data[0], 10, 64)
	if err != nil {
		return RuneID{}, err
	}

	txID, err := strconv.ParseUint(data[1], 10, 32)
	if err != nil {
		return RuneID{}, err
	}

	return RuneID{Block: block, TxID: uint32(txID)}, nil
}

// Next produces next RuneID from delta encoding.
func (id *RuneID) Next(delta RuneID) RuneID {
	if delta.Block == 0 {
		return RuneID{Block: id.Block, TxID: id.TxID + delta.TxID}
	}

	return RuneID{Block: id.Block + delta.Block, TxID: delta.TxID}
}

// Set is a copying setter, sets runeID values to id.
func (id *RuneID) Set(runeID RuneID) {
	id.Block = runeID.Block
	id.TxID = runeID.TxID
}

// String returns RuneID as string.
func (id *RuneID) String() string {
	return fmt.Sprintf("%d:%d", id.Block, id.TxID)
}

// ToIntSeq returns RuneID as integer sequence.
func (id *RuneID) ToIntSeq() []*big.Int {
	return []*big.Int{big.NewInt(int64(id.Block)), big.NewInt(int64(id.TxID))}
}
