// Copyright (C) 2024 Creditor Corp. Group.
// See LICENSE for copying information.

package runes_test

import (
	"math/big"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/BoostyLabs/blockchain/bitcoin/ord/runes"
)

func TestFlags(t *testing.T) {
	etchingAndTerms := new(big.Int).Or(runes.FlagEtching, runes.FlagTerms)
	etchingAndTurbo := new(big.Int).Or(runes.FlagEtching, runes.FlagTurbo)
	termsAndTurbo := new(big.Int).Or(runes.FlagTerms, runes.FlagTurbo)
	all := new(big.Int).Or(etchingAndTerms, etchingAndTurbo)
	none := big.NewInt(0)

	t.Run("Has", func(t *testing.T) {
		require.True(t, runes.HasFlag(etchingAndTerms, runes.FlagEtching))
		require.True(t, runes.HasFlag(etchingAndTerms, runes.FlagTerms))
		require.False(t, runes.HasFlag(etchingAndTerms, runes.FlagTurbo))

		require.True(t, runes.HasFlag(etchingAndTurbo, runes.FlagEtching))
		require.True(t, runes.HasFlag(etchingAndTurbo, runes.FlagTurbo))
		require.False(t, runes.HasFlag(etchingAndTurbo, runes.FlagTerms))

		require.True(t, runes.HasFlag(termsAndTurbo, runes.FlagTerms))
		require.True(t, runes.HasFlag(termsAndTurbo, runes.FlagTurbo))
		require.False(t, runes.HasFlag(termsAndTurbo, runes.FlagEtching))

		require.True(t, runes.HasFlag(all, runes.FlagEtching))
		require.True(t, runes.HasFlag(all, runes.FlagTerms))
		require.True(t, runes.HasFlag(all, runes.FlagTurbo))

		require.False(t, runes.HasFlag(none, runes.FlagEtching))
		require.False(t, runes.HasFlag(none, runes.FlagTerms))
		require.False(t, runes.HasFlag(none, runes.FlagTurbo))
	})

	t.Run("Add", func(t *testing.T) {
		// none flags.
		fl := runes.AddFlag(new(big.Int).Set(none), none)
		require.False(t, runes.HasFlag(fl, runes.FlagEtching))
		require.False(t, runes.HasFlag(fl, runes.FlagTerms))
		require.False(t, runes.HasFlag(fl, runes.FlagTurbo))

		// add etching to previous.
		fl = runes.AddFlag(fl, runes.FlagEtching)
		require.True(t, runes.HasFlag(fl, runes.FlagEtching))
		require.False(t, runes.HasFlag(fl, runes.FlagTerms))
		require.False(t, runes.HasFlag(fl, runes.FlagTurbo))

		// add etching and turbo to previous.
		fl = runes.AddFlag(fl, etchingAndTurbo)
		require.True(t, runes.HasFlag(fl, runes.FlagEtching))
		require.False(t, runes.HasFlag(fl, runes.FlagTerms))
		require.True(t, runes.HasFlag(fl, runes.FlagTurbo))

		// add nothing to previous.
		fl = runes.AddFlag(fl, none)
		require.True(t, runes.HasFlag(fl, runes.FlagEtching))
		require.False(t, runes.HasFlag(fl, runes.FlagTerms))
		require.True(t, runes.HasFlag(fl, runes.FlagTurbo))

		// add all to previous.
		fl = runes.AddFlag(fl, all)
		require.True(t, runes.HasFlag(fl, runes.FlagEtching))
		require.True(t, runes.HasFlag(fl, runes.FlagTerms))
		require.True(t, runes.HasFlag(fl, runes.FlagTurbo))
	})
}
