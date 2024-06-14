// Copyright (C) 2024 Creditor Corp. Group.
// See LICENSE for copying information.

package runes

import (
	"math/big"
)

// Etching defines values to create new rune.
type Etching struct {
	Divisibility *byte
	Premine      *big.Int
	Rune         *Rune
	Spacers      *uint32
	Symbol       *rune
	Terms        *Terms
	Turbo        bool
}

// Terms defines additional Etching parameters.
type Terms struct {
	Amount      *big.Int
	Cap         *big.Int
	HeightStart *uint64
	HeightEnd   *uint64
	OffsetStart *uint64
	OffsetEnd   *uint64
}
