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

func TestMessage(t *testing.T) {
	t.Run("ParseMessage", func(t *testing.T) {
		t.Run("mint", func(t *testing.T) {
			message := &runes2.Message{
				Fields: map[runes2.Tag][]*big.Int{
					runes2.TagMint: {big.NewInt(2585189), big.NewInt(204)},
				},
			}

			parsedMessage, err := runes2.ParseMessage(sequencereader.New(
				[]*big.Int{big.NewInt(20), big.NewInt(2585189), big.NewInt(20), big.NewInt(204)},
			))
			require.NoError(t, err)
			require.Equal(t, message, parsedMessage)
		})

		t.Run("edict", func(t *testing.T) {
			message := &runes2.Message{
				Edicts: []runes2.Edict{
					{
						RuneID: runes2.RuneID{
							Block: 2585359,
							TxID:  84,
						},
						Amount: big.NewInt(1879),
						Output: 1,
					},
				},
			}

			parsedMessage, err := runes2.ParseMessage(sequencereader.New(
				[]*big.Int{big.NewInt(0), big.NewInt(2585359), big.NewInt(84), big.NewInt(1879), big.NewInt(1)},
			))
			require.NoError(t, err)
			require.Equal(t, message, parsedMessage)
		})
	})

	t.Run("ParseMessage (invalid)", func(t *testing.T) {
		t.Run("invalid edicts group size", func(t *testing.T) {
			_, err := runes2.ParseMessage(sequencereader.New(
				[]*big.Int{big.NewInt(0), big.NewInt(1), big.NewInt(2), big.NewInt(3)},
			))
			require.Error(t, err)
			require.ErrorIs(t, err, runes2.ErrCenotaph)
		})

		t.Run("truncated", func(t *testing.T) {
			_, err := runes2.ParseMessage(sequencereader.New(
				[]*big.Int{big.NewInt(20), big.NewInt(21156847), big.NewInt(20)},
			))
			require.Error(t, err)
			require.ErrorIs(t, err, runes2.ErrTruncated)
		})
	})

	t.Run("ToIntSeq", func(t *testing.T) {
		t.Run("mint", func(t *testing.T) {
			seq := []*big.Int{big.NewInt(20), big.NewInt(2585189), big.NewInt(20), big.NewInt(204)}
			message := &runes2.Message{
				Fields: map[runes2.Tag][]*big.Int{
					runes2.TagMint: {big.NewInt(2585189), big.NewInt(204)},
				},
			}

			require.Equal(t, seq, message.ToIntSeq())
		})

		t.Run("edict", func(t *testing.T) {
			seq := []*big.Int{big.NewInt(0), big.NewInt(2585359), big.NewInt(84), big.NewInt(1879), big.NewInt(1)}
			message := &runes2.Message{
				Edicts: []runes2.Edict{
					{
						RuneID: runes2.RuneID{
							Block: 2585359,
							TxID:  84,
						},
						Amount: big.NewInt(1879),
						Output: 1,
					},
				},
			}

			require.Equal(t, seq, message.ToIntSeq())
		})
	})
}
