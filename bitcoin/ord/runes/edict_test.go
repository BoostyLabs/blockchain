// Copyright (C) 2024 Creditor Corp. Group.
// See LICENSE for copying information.

package runes_test

import (
	runes2 "blockchain/bitcoin/ord/runes"
	"math/big"
	"testing"

	"github.com/stretchr/testify/require"

	"blockchain/internal/sequencereader"
)

func TestEdicts(t *testing.T) {
	t.Run("ParseEdictsFromIntSeq (single)", func(t *testing.T) {
		edicts := []runes2.Edict{
			{
				RuneID: runes2.RuneID{
					Block: 2585359,
					TxID:  84,
				},
				Amount: big.NewInt(1879),
				Output: 1,
			},
		}
		payload := sequencereader.New(
			[]*big.Int{big.NewInt(2585359), big.NewInt(84), big.NewInt(1879), big.NewInt(1)},
		)
		parsedEdicts, err := runes2.ParseEdictsFromIntSeq(payload)
		require.NoError(t, err)
		require.Len(t, parsedEdicts, 1)
		require.Equal(t, edicts, parsedEdicts)
	})

	t.Run("ParseEdictsFromIntSeq (many)", func(t *testing.T) {
		edicts := []runes2.Edict{
			{ // base edict.
				RuneID: runes2.RuneID{
					Block: 2585359,
					TxID:  84,
				},
				Amount: big.NewInt(1879),
				Output: 1,
			},
			{ // 0 blocks delta, 16 tx delta.
				RuneID: runes2.RuneID{
					Block: 2585359,
					TxID:  100,
				},
				Amount: big.NewInt(2000),
				Output: 2,
			},
			{ // 0 blocks delta, 0 tx delta.
				RuneID: runes2.RuneID{
					Block: 2585359,
					TxID:  100,
				},
				Amount: big.NewInt(3000),
				Output: 3,
			},
			{ // 1085 blocks delta, 12 tx.
				RuneID: runes2.RuneID{
					Block: 2586444,
					TxID:  12,
				},
				Amount: big.NewInt(10052),
				Output: 4,
			},
		}
		payload := sequencereader.New(
			[]*big.Int{
				big.NewInt(2585359), big.NewInt(84), big.NewInt(1879), big.NewInt(1),
				big.NewInt(0), big.NewInt(16), big.NewInt(2000), big.NewInt(2),
				big.NewInt(0), big.NewInt(0), big.NewInt(3000), big.NewInt(3),
				big.NewInt(1085), big.NewInt(12), big.NewInt(10052), big.NewInt(4),
			},
		)
		parsedEdicts, err := runes2.ParseEdictsFromIntSeq(payload)
		require.NoError(t, err)
		require.Len(t, parsedEdicts, 4)
		require.Equal(t, edicts, parsedEdicts)
	})

	t.Run("ParseEdictsFromIntSeq (invalid length)", func(t *testing.T) {
		payload := sequencereader.New(
			[]*big.Int{big.NewInt(2585359), big.NewInt(84), big.NewInt(1879), big.NewInt(1), big.NewInt(0)},
		)
		_, err := runes2.ParseEdictsFromIntSeq(payload)
		require.Error(t, err)
		require.ErrorIs(t, err, runes2.ErrCenotaph)
	})

	t.Run("EdictsToIntSeq", func(t *testing.T) {
		edict := runes2.Edict{
			RuneID: runes2.RuneID{
				Block: 12,
				TxID:  2,
			},
			Amount: big.NewInt(1000),
			Output: 1,
		}
		seq := []*big.Int{big.NewInt(12), big.NewInt(2), big.NewInt(1000), big.NewInt(1)}
		require.Equal(t, seq, edict.ToIntSeq())
	})

	t.Run("SortEdicts", func(t *testing.T) {
		edicts := []runes2.Edict{
			{
				RuneID: runes2.RuneID{
					Block: 12,
					TxID:  2,
				},
				Amount: big.NewInt(1000),
				Output: 1,
			},
			{
				RuneID: runes2.RuneID{
					Block: 9,
					TxID:  13,
				},
				Amount: big.NewInt(1200),
				Output: 3,
			},
			{
				RuneID: runes2.RuneID{
					Block: 9,
					TxID:  12,
				},
				Amount: big.NewInt(10000),
				Output: 4,
			},
			{
				RuneID: runes2.RuneID{
					Block: 13,
					TxID:  45,
				},
				Amount: big.NewInt(100),
				Output: 3,
			},
		}

		sortedEdicts := []runes2.Edict{
			{
				RuneID: runes2.RuneID{
					Block: 9,
					TxID:  12,
				},
				Amount: big.NewInt(10000),
				Output: 4,
			},
			{
				RuneID: runes2.RuneID{
					Block: 9,
					TxID:  13,
				},
				Amount: big.NewInt(1200),
				Output: 3,
			},
			{
				RuneID: runes2.RuneID{
					Block: 12,
					TxID:  2,
				},
				Amount: big.NewInt(1000),
				Output: 1,
			},
			{
				RuneID: runes2.RuneID{
					Block: 13,
					TxID:  45,
				},
				Amount: big.NewInt(100),
				Output: 3,
			},
		}

		runes2.SortEdicts(edicts)
		require.Equal(t, sortedEdicts, edicts)
	})

	t.Run("SortEdicts", func(t *testing.T) {
		sortedEdicts := []runes2.Edict{
			{
				RuneID: runes2.RuneID{
					Block: 9,
					TxID:  12,
				},
				Amount: big.NewInt(10000),
				Output: 4,
			},
			{
				RuneID: runes2.RuneID{
					Block: 9,
					TxID:  13,
				},
				Amount: big.NewInt(1200),
				Output: 3,
			},
			{
				RuneID: runes2.RuneID{
					Block: 12,
					TxID:  2,
				},
				Amount: big.NewInt(1000),
				Output: 1,
			},
			{
				RuneID: runes2.RuneID{
					Block: 13,
					TxID:  45,
				},
				Amount: big.NewInt(100),
				Output: 3,
			},
		}

		deltaEdicts := []runes2.Edict{
			{ // first edict.
				RuneID: runes2.RuneID{
					Block: 9,
					TxID:  12,
				},
				Amount: big.NewInt(10000),
				Output: 4,
			},
			{ // 0 block delta, 1 tx delta.
				RuneID: runes2.RuneID{
					Block: 0,
					TxID:  1,
				},
				Amount: big.NewInt(1200),
				Output: 3,
			},
			{ // 3 block delta, no delta for tx.
				RuneID: runes2.RuneID{
					Block: 3,
					TxID:  2,
				},
				Amount: big.NewInt(1000),
				Output: 1,
			},
			{ // 1 block delta, no delta for tx.
				RuneID: runes2.RuneID{
					Block: 1,
					TxID:  45,
				},
				Amount: big.NewInt(100),
				Output: 3,
			},
		}

		require.Equal(t, deltaEdicts, runes2.UseDelta(sortedEdicts))
	})

	t.Run("EdictsToIntSeq", func(t *testing.T) {
		edicts := []runes2.Edict{
			{
				RuneID: runes2.RuneID{
					Block: 12,
					TxID:  2,
				},
				Amount: big.NewInt(1000),
				Output: 1,
			},
			{
				RuneID: runes2.RuneID{
					Block: 9,
					TxID:  13,
				},
				Amount: big.NewInt(1200),
				Output: 3,
			},
			{
				RuneID: runes2.RuneID{
					Block: 9,
					TxID:  12,
				},
				Amount: big.NewInt(10000),
				Output: 4,
			},
			{
				RuneID: runes2.RuneID{
					Block: 13,
					TxID:  45,
				},
				Amount: big.NewInt(100),
				Output: 3,
			},
		}

		seq := []*big.Int{
			big.NewInt(9), big.NewInt(12), big.NewInt(10000), big.NewInt(4),
			big.NewInt(0), big.NewInt(1), big.NewInt(1200), big.NewInt(3),
			big.NewInt(3), big.NewInt(2), big.NewInt(1000), big.NewInt(1),
			big.NewInt(1), big.NewInt(45), big.NewInt(100), big.NewInt(3),
		}

		require.Equal(t, seq, runes2.EdictsToIntSeq(edicts))
	})
}
