// Copyright (C) 2024 Creditor Corp. Group.
// See LICENSE for copying information.

package runes_test

import (
	"math/big"
	"testing"

	"github.com/stretchr/testify/require"

	"blockchain/bitcoin/runes"
)

func TestTags(t *testing.T) {
	t.Run("Equal", func(t *testing.T) {
		require.True(t, runes.TagBody.Equal(big.NewInt(int64(runes.TagBody))))
		require.Equal(t, runes.TagBody, runes.TagBody)
		require.False(t, runes.TagFlags.Equal(big.NewInt(int64(runes.TagBody))))
		require.NotEqual(t, runes.TagBody, runes.TagPointer)
	})

	t.Run("BigInt", func(t *testing.T) {
		require.Equal(t, big.NewInt(int64(runes.TagBody)), runes.TagBody.BigInt())
		require.Equal(t, big.NewInt(int64(runes.TagMint)), runes.TagMint.BigInt())
		require.NotEqual(t, big.NewInt(int64(runes.TagPointer)), runes.TagCap.BigInt())
		require.NotEqual(t, big.NewInt(int64(runes.TagNop)), runes.TagDivisibility.BigInt())
	})
}
