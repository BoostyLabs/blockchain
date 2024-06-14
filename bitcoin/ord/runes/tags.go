// Copyright (C) 2024 Creditor Corp. Group.
// See LICENSE for copying information.

package runes

import (
	"math/big"
)

// Tag defines tag type for untyped message parsing.
type Tag byte

const (
	// TagBody defines Body tag.
	TagBody Tag = 0
	// TagFlags defines Flags tag.
	TagFlags Tag = 2
	// TagRune defines Rune tag.
	TagRune Tag = 4
	// TagPremine defines Premine tag.
	TagPremine Tag = 6
	// TagCap defines Cap tag.
	TagCap Tag = 8
	// TagAmount defines Amount tag.
	TagAmount Tag = 10
	// TagHeightStart defines HeightStart tag.
	TagHeightStart Tag = 12
	// TagHeightEnd defines HeightEnd tag.
	TagHeightEnd Tag = 14
	// TagOffsetStart defines OffsetStart tag.
	TagOffsetStart Tag = 16
	// TagOffsetEnd defines OffsetEnd tag.
	TagOffsetEnd Tag = 18
	// TagMint defines Mint tag.
	TagMint Tag = 20
	// TagPointer defines Pointer tag.
	TagPointer Tag = 22
	// TagCenotaph defines Cenotaph tag.
	TagCenotaph Tag = 126

	// TagDivisibility defines Divisibility tag.
	TagDivisibility Tag = 1
	// TagSpacers defines Spacers tag.
	TagSpacers Tag = 3
	// TagSymbol defines Symbol tag.
	TagSymbol Tag = 5
	// TagNop defines Nop tag.
	TagNop Tag = 127
)

// Equal returns true if tags are equal by value.
func (t Tag) Equal(val *big.Int) bool {
	return uint64(t) == val.Uint64()
}

// BigInt returns Tag as big.Int.
func (t Tag) BigInt() *big.Int {
	return new(big.Int).SetUint64(uint64(t))
}
