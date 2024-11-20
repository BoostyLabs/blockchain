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

	t.Run("MinNameLength", func(t *testing.T) {
		tests := []struct {
			block    uint64
			expected int
		}{{0, 13}, {839999, 13}, {840000, 12}, {857499, 12}, {857500, 11}, {1032500, 1}, {1050000, 0}, {1050001, 0}}
		for _, test := range tests {
			require.EqualValues(t, test.expected, runes.MinNameLength(test.block), "%d -> %d", test.block, test.expected)
		}
	})

	t.Run("MinAtHeight for lengths", func(t *testing.T) {
		tests := []struct {
			block    uint64
			expected int
		}{{0, 13}, {839999, 13}, {840000, 12}, {857499, 12}, {857500, 11}, {1032500, 2}, {1050000, 1}, {1050001, 1}}
		for _, test := range tests {
			runeStr := runes.MinAtHeight(test.block).String()
			require.EqualValues(t, test.expected, len(runeStr), "%d -> %d (%s)", test.block, test.expected, runeStr)
		}
	})

	t.Run("MinAtHeight (Mainnet)", func(t *testing.T) {
		start := runes.ProtocolBlockStart
		end := start + runes.SubsidyHalvingInterval
		interval := runes.UnlockNamePeriod
		tests := []struct {
			height  uint64
			minimum string
		}{
			{0, "AAAAAAAAAAAAA"},
			{start / 2, "AAAAAAAAAAAAA"},
			{start, "ZZYZXBRKWXVA"},
			{start + 1, "ZZXZUDIVTVQA"},
			{end - 1, "A"},
			{end, "A"},
			{end + 1, "A"},
			{1<<32 - 1, "A"},

			{start + interval*0 - 1, "AAAAAAAAAAAAA"},
			{start + interval*0 + 0, "ZZYZXBRKWXVA"},
			{start + interval*0 + 1, "ZZXZUDIVTVQA"},

			{start + interval*1 - 1, "AAAAAAAAAAAA"},
			{start + interval*1 + 0, "ZZYZXBRKWXV"},
			{start + interval*1 + 1, "ZZXZUDIVTVQ"},

			{start + interval*2 - 1, "AAAAAAAAAAA"},
			{start + interval*2 + 0, "ZZYZXBRKWY"},
			{start + interval*2 + 1, "ZZXZUDIVTW"},

			{start + interval*3 - 1, "AAAAAAAAAA"},
			{start + interval*3 + 0, "ZZYZXBRKX"},
			{start + interval*3 + 1, "ZZXZUDIVU"},

			{start + interval*4 - 1, "AAAAAAAAA"},
			{start + interval*4 + 0, "ZZYZXBRL"},
			{start + interval*4 + 1, "ZZXZUDIW"},

			{start + interval*5 - 1, "AAAAAAAA"},
			{start + interval*5 + 0, "ZZYZXBS"},
			{start + interval*5 + 1, "ZZXZUDJ"},

			{start + interval*6 - 1, "AAAAAAA"},
			{start + interval*6 + 0, "ZZYZXC"},
			{start + interval*6 + 1, "ZZXZUE"},

			{start + interval*7 - 1, "AAAAAA"},
			{start + interval*7 + 0, "ZZYZY"},
			{start + interval*7 + 1, "ZZXZV"},

			{start + interval*8 - 1, "AAAAA"},
			{start + interval*8 + 0, "ZZZA"},
			{start + interval*8 + 1, "ZZYA"},

			{start + interval*9 - 1, "AAAA"},
			{start + interval*9 + 0, "ZZZ"},
			{start + interval*9 + 1, "ZZY"},

			{start + interval*10 - 2, "AAC"},
			{start + interval*10 - 1, "AAA"},
			{start + interval*10 + 0, "AAA"},
			{start + interval*10 + 1, "AAA"},

			{start + interval*10 + interval/2, "NA"},

			{start + interval*11 - 2, "AB"},
			{start + interval*11 - 1, "AA"},
			{start + interval*11 + 0, "AA"},
			{start + interval*11 + 1, "AA"},

			{start + interval*11 + interval/2, "N"},

			{start + interval*12 - 2, "B"},
			{start + interval*12 - 1, "A"},
			{start + interval*12 + 0, "A"},
			{start + interval*12 + 1, "A"},
		}

		for _, test := range tests {
			runeStr := runes.MinAtHeight(test.height).String()
			require.EqualValues(t, test.minimum, runeStr, "%d -> %d (%s)", test.height, test.minimum, runeStr)
		}
	})
}
