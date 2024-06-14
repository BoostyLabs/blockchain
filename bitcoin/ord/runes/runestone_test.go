// Copyright (C) 2024 Creditor Corp. Group.
// See LICENSE for copying information.

package runes_test

import (
	runes2 "blockchain/bitcoin/ord/runes"
	"encoding/hex"
	"math/big"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestRunestone(t *testing.T) {
	t.Run("parse script data", func(t *testing.T) {
		t.Run("edict only", func(t *testing.T) {
			runestone := &runes2.Runestone{
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

			data, err := hex.DecodeString("6a5d09008fe69d0154d70e01")
			require.NoError(t, err)

			parsedRunestone, err := runes2.ParseRunestone(data)
			require.NoError(t, err)
			require.Equal(t, runestone, parsedRunestone)
		})

		t.Run("mint only", func(t *testing.T) {
			runestone := &runes2.Runestone{
				Mint: &runes2.RuneID{
					Block: 2585189,
					TxID:  204,
				},
			}

			data, err := hex.DecodeString("6a5d0814e5e49d0114cc01")
			require.NoError(t, err)

			parsedRunestone, err := runes2.ParseRunestone(data)
			require.NoError(t, err)
			require.Equal(t, runestone, parsedRunestone)
		})

		t.Run("mint with pointer", func(t *testing.T) {
			pointer := uint32(1)
			runestone := &runes2.Runestone{
				Mint: &runes2.RuneID{
					Block: 2584240,
					TxID:  130,
				},
				Pointer: &pointer,
			}

			data, err := hex.DecodeString("6a5d0a14b0dd9d011482011601")
			require.NoError(t, err)

			parsedRunestone, err := runes2.ParseRunestone(data)
			require.NoError(t, err)
			require.Equal(t, runestone, parsedRunestone)
		})

		t.Run("pointer only", func(t *testing.T) {
			pointer := uint32(14)
			runestone := &runes2.Runestone{
				Pointer: &pointer,
			}

			data, err := hex.DecodeString("6a5d02160e")
			require.NoError(t, err)

			parsedRunestone, err := runes2.ParseRunestone(data)
			require.NoError(t, err)
			require.Equal(t, runestone, parsedRunestone)
		})

		t.Run("etching only", func(t *testing.T) {
			divisibility := byte(10)
			spasers := uint32(0)
			symbol := rune(77)
			rune_, err := runes2.NewRuneFromNumber(big.NewInt(104114246938590))
			require.NoError(t, err)

			runestone := &runes2.Runestone{
				Etching: &runes2.Etching{
					Divisibility: &divisibility,
					Premine:      big.NewInt(210000000),
					Rune:         rune_,
					Spacers:      &spasers,
					Symbol:       &symbol,
				},
			}

			data, err := hex.DecodeString("6a5d15010a0201030004dedfd1e58fd617054d0680b19164")
			require.NoError(t, err)

			parsedRunestone, err := runes2.ParseRunestone(data)
			require.NoError(t, err)
			require.Equal(t, runestone, parsedRunestone)
		})

		t.Run("etching with pointer", func(t *testing.T) {
			divisibility := byte(4)
			spasers := uint32(256)
			symbol := rune(36)
			pointer := uint32(1)
			rune_, err := runes2.NewRuneFromNumber(big.NewInt(1490942589659574650))
			require.NoError(t, err)

			runestone := &runes2.Runestone{
				Etching: &runes2.Etching{
					Divisibility: &divisibility,
					Premine:      big.NewInt(100000000),
					Rune:         rune_,
					Spacers:      &spasers,
					Symbol:       &symbol,
				},
				Pointer: &pointer,
			}

			data, err := hex.DecodeString("6a5d1a020104fae2a3e9ac8cb9d814010403800205240680c2d72f1601")
			require.NoError(t, err)

			parsedRunestone, err := runes2.ParseRunestone(data)
			require.NoError(t, err)
			require.Equal(t, runestone, parsedRunestone)
		})

		t.Run("invalid edict", func(t *testing.T) {
			data, err := hex.DecodeString("6a5d09008fe69d0154d70e0115")
			require.NoError(t, err)

			_, err = runes2.ParseRunestone(data)
			require.Error(t, err)
			require.ErrorContains(t, err, "EOF")
		})
	})

	t.Run("data into script", func(t *testing.T) {
		t.Run("edict only", func(t *testing.T) {
			script := "6a5d09008fe69d0154d70e01"
			runestone := &runes2.Runestone{
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

			data, err := runestone.IntoScript()
			require.NoError(t, err)
			require.Equal(t, script, hex.EncodeToString(data))
		})

		t.Run("mint only", func(t *testing.T) {
			script := "6a5d0814e5e49d0114cc01"
			runestone := &runes2.Runestone{
				Mint: &runes2.RuneID{
					Block: 2585189,
					TxID:  204,
				},
			}

			data, err := runestone.IntoScript()
			require.NoError(t, err)
			require.Equal(t, script, hex.EncodeToString(data))
		})

		t.Run("mint with pointer", func(t *testing.T) {
			script := "6a5d0a14b0dd9d011482011601"
			pointer := uint32(1)
			runestone := &runes2.Runestone{
				Mint: &runes2.RuneID{
					Block: 2584240,
					TxID:  130,
				},
				Pointer: &pointer,
			}

			data, err := runestone.IntoScript()
			require.NoError(t, err)
			require.Equal(t, script, hex.EncodeToString(data))
		})

		t.Run("pointer only", func(t *testing.T) {
			script := "6a5d02160e"
			pointer := uint32(14)
			runestone := &runes2.Runestone{
				Pointer: &pointer,
			}

			data, err := runestone.IntoScript()
			require.NoError(t, err)
			require.Equal(t, script, hex.EncodeToString(data))
		})

		t.Run("etching only", func(t *testing.T) {
			script := "6a5d15010a0201030004dedfd1e58fd617054d0680b19164"
			divisibility := byte(10)
			spasers := uint32(0)
			symbol := rune(77)
			rune_, err := runes2.NewRuneFromNumber(big.NewInt(104114246938590))
			require.NoError(t, err)

			runestone := &runes2.Runestone{
				Etching: &runes2.Etching{
					Divisibility: &divisibility,
					Premine:      big.NewInt(210000000),
					Rune:         rune_,
					Spacers:      &spasers,
					Symbol:       &symbol,
				},
			}

			data, err := runestone.IntoScript()
			require.NoError(t, err)
			require.Equal(t, script, hex.EncodeToString(data))
		})

		t.Run("etching with pointer", func(t *testing.T) {
			divisibility := byte(4)
			spasers := uint32(256)
			symbol := rune(36)
			pointer := uint32(1)
			rune_, err := runes2.NewRuneFromNumber(big.NewInt(1490942589659574650))
			require.NoError(t, err)

			runestone := &runes2.Runestone{
				Etching: &runes2.Etching{
					Divisibility: &divisibility,
					Premine:      big.NewInt(100000000),
					Rune:         rune_,
					Spacers:      &spasers,
					Symbol:       &symbol,
				},
				Pointer: &pointer,
			}

			data, err := hex.DecodeString("6a5d1a020104fae2a3e9ac8cb9d814010403800205240680c2d72f1601")
			require.NoError(t, err)

			parsedRunestone, err := runes2.ParseRunestone(data)
			require.NoError(t, err)
			require.Equal(t, runestone, parsedRunestone)
		})
	})

	t.Run("bytes to integer sequence", func(t *testing.T) {
		t.Run("mint", func(t *testing.T) {
			tSeq := []*big.Int{big.NewInt(20), big.NewInt(2585189), big.NewInt(20), big.NewInt(204)}
			data, err := hex.DecodeString("14e5e49d0114cc01")
			require.NoError(t, err)

			seq, err := runes2.PayloadIntoIntSequence(data)
			require.NoError(t, err)
			require.Equal(t, tSeq, seq)
		})

		t.Run("edict", func(t *testing.T) {
			tSeq := []*big.Int{big.NewInt(0), big.NewInt(2585359), big.NewInt(84), big.NewInt(1879), big.NewInt(1)}
			data, err := hex.DecodeString("008fe69d0154d70e01")
			require.NoError(t, err)

			seq, err := runes2.PayloadIntoIntSequence(data)
			require.NoError(t, err)
			require.Equal(t, tSeq, seq)
		})

		// TODO: Add tests.
	})

	t.Run("integer sequence into bytes", func(t *testing.T) {
		t.Run("mint", func(t *testing.T) {
			seq := []*big.Int{big.NewInt(20), big.NewInt(2585189), big.NewInt(20), big.NewInt(204)}
			payload := "14e5e49d0114cc01"

			data, err := runes2.IntSequenceIntoPayload(seq)
			require.NoError(t, err)
			require.Equal(t, payload, hex.EncodeToString(data))
		})

		t.Run("edict", func(t *testing.T) {
			seq := []*big.Int{big.NewInt(0), big.NewInt(2585359), big.NewInt(84), big.NewInt(1879), big.NewInt(1)}
			payload := "008fe69d0154d70e01"

			data, err := runes2.IntSequenceIntoPayload(seq)
			require.NoError(t, err)
			require.Equal(t, payload, hex.EncodeToString(data))
		})

		// TODO: Add tests.
	})

	t.Run("IsPossibleRunestone", func(t *testing.T) {
		tests := []struct {
			script string
			mustBe bool
		}{
			{"6a5d09008fe69d0154d70e01", true},
			{"6a5d0814e5e49d0114cc01", true},
			{"6a5d0a14b0dd9d011482011601", true},
			{"6a5d02160e", true},
			{"6a5d15010a0201030004dedfd1e58fd617054d0680b19164", true},
			{"6a5d1a020104fae2a3e9ac8cb9d814010403800205240680c2d72f1601", true},
			{"", false},
			{"10", false},
			{"0231", false},
			{"6a5d1a", false},
			{"6a5dff00", false},
			{"6affff00", false},
			{"ffffff00", false},
			{"ff5d1a00", false},
			{"6a5d1a00", true},
		}
		for _, test := range tests {
			script, err := hex.DecodeString(test.script)
			require.NoError(t, err)
			require.Equal(t, test.mustBe, runes2.IsPossibleRunestone(script))
		}
	})

	t.Run("IsValid...", func(t *testing.T) {
		tests := []struct {
			script         string
			isValidEtching bool
			isValidMint    bool
			isValidEdicts  bool
		}{
			{"6a5d09008fe69d0154d70e01", false, false, true},
			{"6a5d0814e5e49d0114cc01", false, true, false},
			{"6a5d0a14b0dd9d011482011601", false, true, false},
			{"6a5d02160e", false, false, false},
			{"6a5d15010a0201030004dedfd1e58fd617054d0680b19164", true, false, false},
			{"6a5d1a020104fae2a3e9ac8cb9d814010403800205240680c2d72f1601", true, false, false},
		}
		for _, test := range tests {
			script, err := hex.DecodeString(test.script)
			require.NoError(t, err)

			runestone, err := runes2.ParseRunestone(script)
			require.NoError(t, err)
			require.Equal(t, test.isValidEtching, runestone.IsValidEtching(2))
			require.Equal(t, test.isValidMint, runestone.IsValidMint(2))
			require.Equal(t, test.isValidEdicts, runestone.IsValidEdicts(2))
		}
	})
}
