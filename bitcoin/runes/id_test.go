// Copyright (C) 2024 Creditor Corp. Group.
// See LICENSE for copying information.

package runes_test

import (
	"math/big"
	"testing"

	"github.com/stretchr/testify/require"

	"blockchain/bitcoin/runes"
)

func TestRuneID(t *testing.T) {
	runeID := runes.RuneID{
		Block: 22556689,
		TxID:  15,
	}

	t.Run("from 0 runeID", func(t *testing.T) {
		zeroRuneID := runes.RuneID{Block: 0, TxID: 0}
		require.Equal(t, runeID, zeroRuneID.Next(runeID))
	})

	t.Run("Next (0 delta block)", func(t *testing.T) {
		require.Equal(t, runes.RuneID{Block: 22556689, TxID: 17}, runeID.Next(runes.RuneID{Block: 0, TxID: 2}))
	})

	t.Run("Next (0 delta block and tx)", func(t *testing.T) {
		require.Equal(t, runeID, runeID.Next(runes.RuneID{Block: 0, TxID: 0}))
	})

	t.Run("Next (not 0 delta block)", func(t *testing.T) {
		require.Equal(t, runes.RuneID{Block: 22556690, TxID: 2}, runeID.Next(runes.RuneID{Block: 1, TxID: 2}))
	})

	t.Run("ToIntSeq", func(t *testing.T) {
		seq := []*big.Int{big.NewInt(int64(runeID.Block)), big.NewInt(int64(runeID.TxID))}
		require.Equal(t, seq, runeID.ToIntSeq())
	})

	t.Run("NewRuneIDFromString", func(t *testing.T) {
		tests := []struct {
			input   string
			result  runes.RuneID
			invalid bool
		}{
			{
				input:  "22556689:15",
				result: runeID,
			},
			{
				input:   "2255668915",
				invalid: true,
			},
			{
				input:   "22556689:15F",
				invalid: true,
			},
			{
				input:   "2255pp89:15",
				invalid: true,
			},
			{
				input:   "",
				invalid: true,
			},
		}
		for _, test := range tests {
			parsedRuneID, err := runes.NewRuneIDFromString(test.input)
			if test.invalid {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				require.EqualValues(t, test.result, parsedRuneID)
			}
		}
	})
}
