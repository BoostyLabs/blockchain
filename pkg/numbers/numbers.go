// Copyright (C) 2022 Creditor Corp. Group.
// See LICENSE for copying information.

package numbers

import (
	"math/big"
)

// Zero defines 0 number.
const Zero = 0

// ZeroBigInt defies 0 as *big.Int type.
var ZeroBigInt = big.NewInt(0)

// OneBigInt defies 1 as *big.Int type.
var OneBigInt = big.NewInt(1)

// MaxUInt128Value defines maximum value of uint128 type.
var MaxUInt128Value = new(big.Int).Sub(new(big.Int).Lsh(OneBigInt, 128), OneBigInt)

// MaxUInt256Value defines maximum value of uint256 type.
var MaxUInt256Value = new(big.Int).Sub(new(big.Int).Lsh(OneBigInt, 256), OneBigInt)

// IsNegative returns true if the number is less than zero.
func IsNegative(num *big.Int) bool {
	return num.Sign() < Zero
}

// IsPositive returns true if the number is grater than zero.
func IsPositive(num *big.Int) bool {
	return num.Sign() > Zero
}

// IsZero returns true if the number is zero.
func IsZero(num *big.Int) bool {
	return num.Sign() == Zero
}

// IsGreater returns true is a > b.
func IsGreater(a, b *big.Int) bool {
	return a.Cmp(b) > Zero
}

// IsEqual returns true is a = b.
func IsEqual(a, b *big.Int) bool {
	return a.Cmp(b) == Zero
}

// IsLess returns true is a < b.
func IsLess(a, b *big.Int) bool {
	return a.Cmp(b) < Zero
}

// Max returns the largest value from provided.
func Max(a *big.Int, b ...*big.Int) *big.Int {
	maxValue := a
	for _, el := range b {
		if IsGreater(el, maxValue) {
			maxValue = el
		}
	}

	return maxValue
}

// Min returns the least value from provided.
func Min(a *big.Int, b ...*big.Int) *big.Int {
	minValue := a
	for _, el := range b {
		if IsLess(el, minValue) {
			minValue = el
		}
	}

	return minValue
}
