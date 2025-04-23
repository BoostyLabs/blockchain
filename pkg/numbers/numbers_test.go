// Copyright (C) 2022 Creditor Corp. Group.
// See LICENSE for copying information.

package numbers_test

import (
	"math/big"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/BoostyLabs/blockchain/internal/numbers"
)

func TestNumbers(t *testing.T) {
	negative := big.NewInt(-100)
	zero := big.NewInt(0)
	positive := big.NewInt(100)

	t.Run("IsNegative", func(t *testing.T) {
		require.True(t, numbers.IsNegative(negative))
		require.False(t, numbers.IsNegative(zero))
		require.False(t, numbers.IsNegative(positive))
	})

	t.Run("IsZero", func(t *testing.T) {
		require.False(t, numbers.IsZero(negative))
		require.True(t, numbers.IsZero(zero))
		require.False(t, numbers.IsZero(positive))
	})

	t.Run("IsPositive", func(t *testing.T) {
		require.False(t, numbers.IsPositive(negative))
		require.False(t, numbers.IsPositive(zero))
		require.True(t, numbers.IsPositive(positive))
	})

	t.Run("IsBigger", func(t *testing.T) {
		require.True(t, numbers.IsGreater(positive, negative))
		require.False(t, numbers.IsGreater(negative, positive))
	})

	t.Run("IsLess", func(t *testing.T) {
		require.False(t, numbers.IsLess(positive, negative))
		require.True(t, numbers.IsLess(negative, positive))
	})

	t.Run("IsEqual", func(t *testing.T) {
		require.False(t, numbers.IsEqual(positive, negative))
		require.False(t, numbers.IsEqual(negative, positive))
		require.True(t, numbers.IsEqual(positive, positive))
		require.True(t, numbers.IsEqual(negative, negative))
	})

	t.Run("MaxUint128Value", func(t *testing.T) {
		for i := 0; i < 128; i++ {
			require.EqualValues(t, numbers.MaxUInt128Value.Bit(i), 1)
		}
		require.EqualValues(t, numbers.MaxUInt128Value.Bit(128), 0)
	})

	t.Run("MaxUint256Value", func(t *testing.T) {
		for i := 0; i < 256; i++ {
			require.EqualValues(t, numbers.MaxUInt256Value.Bit(i), 1)
		}
		require.EqualValues(t, numbers.MaxUInt256Value.Bit(256), 0)
	})

	t.Run("MaxMin", func(t *testing.T) {
		bigInts := []*big.Int{big.NewInt(235), big.NewInt(-5158), big.NewInt(56546584), big.NewInt(-46468484)}
		require.EqualValues(t, bigInts[2], numbers.Max(bigInts[0], bigInts[1:]...))
		require.EqualValues(t, bigInts[3], numbers.Min(bigInts[0], bigInts[1:]...))
	})
}
