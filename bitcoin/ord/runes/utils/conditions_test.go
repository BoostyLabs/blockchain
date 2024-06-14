// Copyright (C) 2024 Creditor Corp. Group.
// See LICENSE for copying information.

package utils_test

import (
	"blockchain/bitcoin/ord/runes/utils"
	"errors"
	"math/big"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestConditions(t *testing.T) {
	t.Run("IfLen", func(t *testing.T) {
		var fn = func() error {
			return nil
		}

		var fnErr = func() error {
			return errors.New("error")
		}

		t.Run("Condition true", func(t *testing.T) {
			require.True(t, utils.IfLen(make([]*big.Int, 4), 4).Then(fn).Ok())
		})

		t.Run("Condition false", func(t *testing.T) {
			require.False(t, utils.IfLen(make([]*big.Int, 3), 4).Then(fn).Ok())
			require.False(t, utils.IfLen(make([]*big.Int, 5), 4).Then(fn).Ok())
		})

		t.Run("error", func(t *testing.T) {
			require.NoError(t, utils.IfLen(make([]*big.Int, 4), 4).Then(fn).Error())
			require.NoError(t, utils.IfLen(make([]*big.Int, 5), 1).Then(fnErr).Error())
			require.Error(t, utils.IfLen(make([]*big.Int, 5), 5).Then(fnErr).Error())
		})
	})
}
