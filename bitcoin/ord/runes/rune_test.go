// Copyright (C) 2024 Creditor Corp. Group.
// See LICENSE for copying information.

package runes_test

import (
	"errors"
	"math/big"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/BoostyLabs/blockchain/bitcoin/ord/runes"
)

func TestRunes(t *testing.T) {
	t.Run("conversions", func(t *testing.T) {
		tests := []struct {
			num *big.Int
			str string
		}{
			{big.NewInt(0), "A"},
			{big.NewInt(1), "B"},
			{big.NewInt(2), "C"},
			{big.NewInt(3), "D"},
			{big.NewInt(4), "E"},
			{big.NewInt(5), "F"},
			{big.NewInt(6), "G"},
			{big.NewInt(7), "H"},
			{big.NewInt(8), "I"},
			{big.NewInt(9), "J"},
			{big.NewInt(10), "K"},
			{big.NewInt(11), "L"},
			{big.NewInt(12), "M"},
			{big.NewInt(13), "N"},
			{big.NewInt(14), "O"},
			{big.NewInt(15), "P"},
			{big.NewInt(16), "Q"},
			{big.NewInt(17), "R"},
			{big.NewInt(18), "S"},
			{big.NewInt(19), "T"},
			{big.NewInt(20), "U"},
			{big.NewInt(21), "V"},
			{big.NewInt(22), "W"},
			{big.NewInt(23), "X"},
			{big.NewInt(24), "Y"},
			{big.NewInt(25), "Z"},
			{big.NewInt(26), "AA"},
			{big.NewInt(27), "AB"},
			{big.NewInt(51), "AZ"},
			{big.NewInt(52), "BA"},
			{runes.MaxU128, "BCGDENLQRQWDSLRUGSNLBTMFIJAV"},
		}
		for _, test := range tests {
			runeFromStr, err := runes.NewRuneFromString(test.str)
			require.NoError(t, err)
			runeFromNum, err := runes.NewRuneFromNumber(test.num)
			require.NoError(t, err)
			require.Equal(t, runeFromStr.Value(), test.num, "str: "+test.str)
			require.Equal(t, runeFromNum.String(), test.str, "num: "+test.num.String())
		}
	})

	t.Run("NewRuneFromString", func(t *testing.T) {
		var (
			errSymb         = errors.New("invalid symbol in the rune")
			errU128Overflow = errors.New("value overflows uint128")
		)
		tests := []struct {
			str string
			err error
		}{
			{"A", nil},
			{"B", nil},
			{"AB", nil},
			{"BA", nil},
			{"AZNF", nil},
			{"Aok", errSymb},
			{"TP3", errSymb},
			{"ORNV_", errSymb},
			{"OR V", errSymb},
			{"OR2V", errSymb},
			{"123", errSymb},
			{"ABCDEFGHIJKLMNOPQRSTUVWXYZ", nil},
			{"ZZZZZZZZZZZZZZZZZZZZZZZZZZZZ", errU128Overflow}, // uint128 overflow.
		}
		for _, test := range tests {
			_, err := runes.NewRuneFromString(test.str)
			require.Equal(t, test.err, err)
		}
	})

	t.Run("StringWithSeparator", func(t *testing.T) {
		tests := []struct {
			rawRune      string
			spacer       string
			spacers      uint32
			expectedRune string
		}{
			{
				rawRune:      "ABCDEFGHIJKLMNOPQRSTUVWXYZ",
				spacer:       "_",
				spacers:      0b00000000_10010010_01001001_00100100,
				expectedRune: "ABC_DEF_GHI_JKL_MNO_PQR_STU_VWX_YZ",
			},
			{
				rawRune:      "ABCDEFGHIJKLMNOPQRSTUVWXYZ",
				spacer:       "",
				spacers:      0b00000000_10010010_01001001_00100100,
				expectedRune: "ABC•DEF•GHI•JKL•MNO•PQR•STU•VWX•YZ",
			},
			{
				rawRune:      "HELLOTESTRUNE",
				spacer:       " ",
				spacers:      0b00000000_00000000_00010001_00010000,
				expectedRune: "HELLO TEST RUNE",
			},
			{
				rawRune:      "HELLOTESTRUNE",
				spacer:       "\\/",
				spacers:      0b00000000_00000000_00010001_00010000,
				expectedRune: "HELLO\\/TEST\\/RUNE",
			},
		}
		for _, test := range tests {
			rune_, err := runes.NewRuneFromString(test.rawRune)
			require.NoError(t, err)
			require.EqualValues(t, test.expectedRune, rune_.StringWithSeparator(test.spacers, test.spacer), test.rawRune)
		}

	})
}
