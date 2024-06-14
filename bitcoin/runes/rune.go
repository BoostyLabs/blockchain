// Copyright (C) 2024 Creditor Corp. Group.
// See LICENSE for copying information.

package runes

import (
	"errors"
	"math/big"
)

// DefaultSpacer defines default spacer for Rune name.
const DefaultSpacer = "•"

// base26 defines 26 as *big.Int.
var base26 = big.NewInt(26)

// one predefines one as *big.Int.
var one = big.NewInt(1)

// MaxU128 defines maximum value of uint128 integer type.
var MaxU128 = new(big.Int).Sub(new(big.Int).Lsh(one, 128), one)

// intToChar defines conversion rules from integers to chars.
var intToChar = map[int64]byte{
	0:  'A',
	1:  'B',
	2:  'C',
	3:  'D',
	4:  'E',
	5:  'F',
	6:  'G',
	7:  'H',
	8:  'I',
	9:  'J',
	10: 'K',
	11: 'L',
	12: 'M',
	13: 'N',
	14: 'O',
	15: 'P',
	16: 'Q',
	17: 'R',
	18: 'S',
	19: 'T',
	20: 'U',
	21: 'V',
	22: 'W',
	23: 'X',
	24: 'Y',
	25: 'Z',
}

// Rune defines rune names and encodes as modified base-26 integers.
type Rune struct {
	value *big.Int
}

// NewRuneFromString creates new Rune from string name.
// TODO: Add constructor for Rune name with spacer.
func NewRuneFromString(runeStr string) (*Rune, error) {
	var value = big.NewInt(0)
	for i, c := range runeStr {
		if i > 0 {
			value.Add(value, one)
		}
		value = value.Mul(value, base26)
		if c < 'A' || c > 'Z' {
			return nil, errors.New("invalid symbol in the rune")
		}
		value = value.Add(value, big.NewInt(int64(c)-'A'))
	}

	if value.Cmp(MaxU128) > 0 {
		return nil, errors.New("value overflows uint128")
	}

	return &Rune{value: value}, nil
}

// NewRuneFromNumber creates new Rune from number.
func NewRuneFromNumber(number *big.Int) (*Rune, error) {
	if number.Cmp(MaxU128) > 0 || number.Sign() < 0 {
		return nil, errors.New("invalid number")
	}

	return &Rune{value: number}, nil
}

// Value returns Rune name as number.
func (r *Rune) Value() *big.Int {
	return r.value
}

// String returns Rune name as string.
func (r *Rune) String() string {
	var value = new(big.Int).Set(r.value)
	if value.Cmp(MaxU128) == 0 {
		return "BCGDENLQRQWDSLRUGSNLBTMFIJAV"
	}

	value = value.Add(value, one)
	var symbol string
	for value.Sign() > 0 {
		valueSubOne := new(big.Int).Sub(value, one)
		idx := new(big.Int).Mod(valueSubOne, base26)

		symbol = string(intToChar[idx.Int64()]) + symbol

		value = valueSubOne.Div(valueSubOne, base26)
	}

	return symbol
}

// StringWithSeparator returns Rune name as string with provides spacer.
// If spacer is an empty strung, DefaultSpacer will be used.
func (r *Rune) StringWithSeparator(spacers uint32, spacer string) string {
	rune_ := r.String()

	if spacer == "" {
		spacer = DefaultSpacer
	}

	symbol := ""
	for idx, char := range rune_ {
		symbol += string(char)

		if idx < len(rune_)-1 && spacers&(1<<idx) != 0 {
			symbol += spacer
		}
	}

	return symbol
}
