// Copyright (C) 2024 Creditor Corp. Group.
// See LICENSE for copying information.

package txbuilder

import (
	"fmt"
	"math/big"
)

type balanceErrorType string

const (
	// InsufficientErrorTypeBitcoin defines insufficient bitcoin balance error type.
	InsufficientErrorTypeBitcoin balanceErrorType = "bitcoin"
	// InsufficientErrorTypeRune defines insufficient rune balance error type.
	InsufficientErrorTypeRune balanceErrorType = "rune"
)

// InsufficientError is the error type to describe insufficient balance errors with details.
type InsufficientError struct {
	Type balanceErrorType
	Need *big.Int
	Have *big.Int
}

// NewInsufficientError is a constructor for InsufficientError.
func NewInsufficientError(type_ balanceErrorType, need, have *big.Int) *InsufficientError {
	return &InsufficientError{type_, need, have}
}

// Error returns error description.
func (e *InsufficientError) Error() string {
	if e.Have == nil || e.Need == nil {
		return fmt.Sprintf("insufficient %s balance", e.Type)
	}
	return fmt.Sprintf("insufficient %s balance: Need - %s, Have - %s", e.Type, e.Need, e.Have)
}

// clarify returns formed error with Need and Have values set.
func (e *InsufficientError) clarify(need, have *big.Int) error {
	return &InsufficientError{e.Type, need, have}
}
