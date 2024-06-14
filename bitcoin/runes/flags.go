// Copyright (C) 2024 Creditor Corp. Group.
// See LICENSE for copying information.

package runes

import (
	"math/big"
)

var (
	// FlagEtching defines that the transaction contains an etching.
	FlagEtching = big.NewInt(1)
	// FlagTerms defines that the transaction's etching has open mint terms.
	FlagTerms = new(big.Int).Lsh(big.NewInt(1), 1)
	// FlagTurbo defines that the transaction's etching has set turbo mode.
	FlagTurbo = new(big.Int).Lsh(big.NewInt(1), 2)
	// FlagCenotaph is unrecognized.
	FlagCenotaph = new(big.Int).Lsh(big.NewInt(1), 127)
)

// HasFlag returns true if transmitted value.
func HasFlag(value *big.Int, flag *big.Int) bool {
	return value.Cmp(new(big.Int).Or(value, flag)) == 0
}

// AddFlag adds flag to the value, returns updated value (transmitted value will be mutated).
func AddFlag(value *big.Int, flag *big.Int) *big.Int {
	return value.Or(value, flag)
}
