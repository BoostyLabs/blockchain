// Copyright (C) 2024 Creditor Corp. Group.
// See LICENSE for copying information.

package runes_test

import (
	"errors"
	"math/big"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/BoostyLabs/blockchain/bitcoin/ord/runes"
	"github.com/BoostyLabs/blockchain/internal/numbers"
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

	t.Run("MaxUInt128 name", func(t *testing.T) {
		val := big.NewInt(20)
		rune_, err := runes.NewRuneFromNumber(val)
		require.NoError(t, err)

		val.Set(numbers.MaxUInt128Value)

		require.EqualValues(t, "BCGDENLQRQWDSLRUGSNLBTMFIJAV", rune_.String())
	})

	t.Run("NewRuneFromString", func(t *testing.T) {
		var (
			errSymb         = errors.New("invalid symbol in the rune")
			errU128Overflow = errors.New("value overflows uint128")
			errReserved     = errors.New("reserved name")
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
			{"ABACDEFGHIJKLMNOPQRSTUVWXYZ", errReserved},      // > AAAAAAAAAAAAAAAAAAAAAAAAAAA.
			{"ZZZZZZZZZZZZZZZZZZZZZZZZZZZZ", errU128Overflow}, // uint128 overflow.
		}
		for _, test := range tests {
			_, err := runes.NewRuneFromString(test.str)
			require.Equal(t, test.err, err)
		}
	})

	t.Run("NewRuneFromStringWithSpacer", func(t *testing.T) {
		var (
			rune_  *runes.Rune
			spacer uint32
			err    error
		)
		tests := []struct {
			runeWithSpacer string
			spacer         rune
			spacers        uint32
			expectedRune   string
		}{
			{
				runeWithSpacer: "ABC_DEF_GHI_JKL_MNO_PQR_STU_VWX_YZ",
				spacer:         '_',
				spacers:        0b00000000_10010010_01001001_00100100,
				expectedRune:   "ABCDEFGHIJKLMNOPQRSTUVWXYZ",
			},
			{
				runeWithSpacer: "ABC•DEF•GHI•JKL•MNO•PQR•STU•VWX•YZ",
				spacers:        0b00000000_10010010_01001001_00100100,
				expectedRune:   "ABCDEFGHIJKLMNOPQRSTUVWXYZ",
			},
			{
				runeWithSpacer: "HELLO TEST RUNE",
				spacer:         ' ',
				spacers:        0b00000000_00000000_00000001_00010000,
				expectedRune:   "HELLOTESTRUNE",
			},
			{
				runeWithSpacer: "HE\\LLO\\TEST\\RUN\\E",
				spacer:         '\\',
				spacers:        0b00000000_00000000_00001001_00010010,
				expectedRune:   "HELLOTESTRUNE",
			},
		}
		for _, test := range tests {
			if test.spacer == 0 {
				rune_, spacer, err = runes.NewRuneFromStringWithSpacer(test.runeWithSpacer)
			} else {
				rune_, spacer, err = runes.NewRuneFromStringWithSpacer(test.runeWithSpacer, test.spacer)
			}
			require.NoError(t, err)
			require.EqualValues(t, test.spacers, spacer)
			require.EqualValues(t, test.expectedRune, rune_.String(), test.expectedRune)
			require.EqualValues(t, test.expectedRune, rune_.String(), test.expectedRune)
		}
	})

	t.Run("StringWithSeparator", func(t *testing.T) {
		tests := []struct {
			rawRune      string
			spacer       rune
			spacers      uint32
			expectedRune string
		}{
			{
				rawRune:      "ABCDEFGHIJKLMNOPQRSTUVWXYZ",
				spacer:       '_',
				spacers:      0b00000000_10010010_01001001_00100100,
				expectedRune: "ABC_DEF_GHI_JKL_MNO_PQR_STU_VWX_YZ",
			},
			{
				rawRune:      "ABCDEFGHIJKLMNOPQRSTUVWXYZ",
				spacers:      0b00000000_10010010_01001001_00100100,
				expectedRune: "ABC•DEF•GHI•JKL•MNO•PQR•STU•VWX•YZ",
			},
			{
				rawRune:      "HELLOTESTRUNE",
				spacer:       ' ',
				spacers:      0b00000000_00000000_00000001_00010000,
				expectedRune: "HELLO TEST RUNE",
			},
			{
				rawRune:      "HELLOTESTRUNE",
				spacer:       '\\',
				spacers:      0b00000000_00000000_00001001_00010010,
				expectedRune: "HE\\LLO\\TEST\\RUN\\E",
			},
		}
		for _, test := range tests {
			rune_, err := runes.NewRuneFromString(test.rawRune)
			require.NoError(t, err)
			if test.spacer == 0 {
				require.EqualValues(t, test.expectedRune, rune_.StringWithSeparator(test.spacers), test.rawRune)
			} else {
				require.EqualValues(t, test.expectedRune, rune_.StringWithSeparator(test.spacers, test.spacer), test.rawRune)
			}
		}
	})

	t.Run("RuneReserve", func(t *testing.T) {
		tests := []struct {
			block    uint64
			tx       uint32
			expected string
		}{
			{0, 0, "AAAAAAAAAAAAAAAAAAAAAAAAAAA"},
			{0, 1, "AAAAAAAAAAAAAAAAAAAAAAAAAAB"},
			{100, 1, "AAAAAAAAAAAAAAAAAACBMITDVSR"},
			{1<<64 - 1, 1<<32 - 1, "ZZZZZZZZZZZZZZZZZZZZZZZZZZ"},
		}
		for _, test := range tests {
			require.EqualValues(t, test.expected, runes.RuneReserve(runes.RuneID{Block: test.block, TxID: test.tx}).String())
		}
	})

	t.Run("RuneReserve", func(t *testing.T) {
		tests := []struct {
			block    uint64
			expected int
		}{{0, 13}, {839999, 13}, {840000, 12}, {857499, 12}, {857500, 11}, {1032500, 1}, {1050000, 0}, {1050001, 0}}
		for _, test := range tests {
			require.EqualValues(t, test.expected, runes.MinNameLength(test.block), "%d -> %d", test.block, test.expected)
		}
	})
}
