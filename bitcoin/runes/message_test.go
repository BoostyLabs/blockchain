// Copyright (C) 2024 Creditor Corp. Group.
// See LICENSE for copying information.

package runes_test

import (
	"math/big"
	"testing"

	"github.com/stretchr/testify/require"

	"blockchain/bitcoin/runes"
	"blockchain/internal/sequencereader"
)

func TestMessage(t *testing.T) {
	t.Run("ParseMessage", func(t *testing.T) {
		t.Run("mint", func(t *testing.T) {
			message := &runes.Message{
				Fields: map[runes.Tag][]*big.Int{
					runes.TagMint: {big.NewInt(2585189), big.NewInt(204)},
				},
			}

			parsedMessage, err := runes.ParseMessage(sequencereader.New(
				[]*big.Int{big.NewInt(20), big.NewInt(2585189), big.NewInt(20), big.NewInt(204)},
			))
			require.NoError(t, err)
			require.Equal(t, message, parsedMessage)
		})

		t.Run("edict", func(t *testing.T) {
			message := &runes.Message{
				Edicts: []runes.Edict{
					{
						RuneID: runes.RuneID{
							Block: 2585359,
							TxID:  84,
						},
						Amount: big.NewInt(1879),
						Output: 1,
					},
				},
			}

			parsedMessage, err := runes.ParseMessage(sequencereader.New(
				[]*big.Int{big.NewInt(0), big.NewInt(2585359), big.NewInt(84), big.NewInt(1879), big.NewInt(1)},
			))
			require.NoError(t, err)
			require.Equal(t, message, parsedMessage)
		})
	})

	t.Run("ParseMessage (invalid)", func(t *testing.T) {
		t.Run("invalid edicts group size", func(t *testing.T) {
			_, err := runes.ParseMessage(sequencereader.New(
				[]*big.Int{big.NewInt(0), big.NewInt(1), big.NewInt(2), big.NewInt(3)},
			))
			require.Error(t, err)
			require.ErrorIs(t, err, runes.ErrCenotaph)
		})

		t.Run("truncated", func(t *testing.T) {
			_, err := runes.ParseMessage(sequencereader.New(
				[]*big.Int{big.NewInt(20), big.NewInt(21156847), big.NewInt(20)},
			))
			require.Error(t, err)
			require.ErrorIs(t, err, runes.ErrTruncated)
		})
	})

	t.Run("ToIntSeq", func(t *testing.T) {
		t.Run("mint", func(t *testing.T) {
			seq := []*big.Int{big.NewInt(20), big.NewInt(2585189), big.NewInt(20), big.NewInt(204)}
			message := &runes.Message{
				Fields: map[runes.Tag][]*big.Int{
					runes.TagMint: {big.NewInt(2585189), big.NewInt(204)},
				},
			}

			require.Equal(t, seq, message.ToIntSeq())
		})

		t.Run("edict", func(t *testing.T) {
			seq := []*big.Int{big.NewInt(0), big.NewInt(2585359), big.NewInt(84), big.NewInt(1879), big.NewInt(1)}
			message := &runes.Message{
				Edicts: []runes.Edict{
					{
						RuneID: runes.RuneID{
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
