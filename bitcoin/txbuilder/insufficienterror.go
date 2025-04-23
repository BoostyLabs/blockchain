// Copyright (C) 2024 Creditor Corp. Group.
// See LICENSE for copying information.

package txbuilder

import (
	"fmt"
	"math/big"
)

type BalanceErrorType string

type CauserSign string

const (
	// InsufficientErrorTypeBitcoin defines insufficient bitcoin balance error type.
	InsufficientErrorTypeBitcoin BalanceErrorType = "bitcoin"
	// InsufficientErrorTypeRune defines insufficient rune balance error type.
	InsufficientErrorTypeRune BalanceErrorType = "rune"

	// CauserSender defines that the sender caused this error type.
	CauserSender CauserSign = "sender"
	// CauserFeePayer defines that the fee-payer caused this error type.
	CauserFeePayer CauserSign = "fee-payer"
)

// InsufficientError is the error type to describe insufficient balance errors with details.
type InsufficientError struct {
	Type   BalanceErrorType
	Need   *big.Int
	Have   *big.Int
	Causer CauserSign
}

// NewInsufficientError is a constructor for InsufficientError.
func NewInsufficientError(type_ BalanceErrorType, need, have *big.Int) *InsufficientError {
	return &InsufficientError{type_, need, have, ""}
}

// Error returns error description.
func (e *InsufficientError) Error() string {
	var errMsg = fmt.Sprintf("insufficient %s balance", e.Type)

	if e.Have == nil || e.Need == nil {
		errMsg += fmt.Sprintf(": Need - %s, Have - %s", e.Need, e.Have)
	}

	if e.Causer != "" {
		errMsg += " (" + string(e.Causer) + ")"
	}

	return errMsg
}

// Is implements comparator method for [errors] package.
func (e *InsufficientError) Is(target error) bool {
	return e.Error() == target.Error()
}

// Clarify returns formed error with Need and Have values set.
func (e *InsufficientError) Clarify(need, have *big.Int) *InsufficientError {
	return &InsufficientError{e.Type, need, have, e.Causer}
}

// SetCauser updates InsufficientError with provided causer.
func (e *InsufficientError) SetCauser(causer CauserSign) *InsufficientError {
	e.Causer = causer
	return e
}
