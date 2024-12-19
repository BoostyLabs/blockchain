// Copyright (C) 2024 Creditor Corp. Group.
// See LICENSE for copying information.

package txbuilder

import (
	"fmt"
	"math/big"
)

type balanceErrorType string

type causerSign string

const (
	// InsufficientErrorTypeBitcoin defines insufficient bitcoin balance error type.
	InsufficientErrorTypeBitcoin balanceErrorType = "bitcoin"
	// InsufficientErrorTypeRune defines insufficient rune balance error type.
	InsufficientErrorTypeRune balanceErrorType = "rune"

	// CauserSender defines that the sender caused this error type.
	CauserSender causerSign = "sender"
	// CauserFeePayer defines that the fee-payer caused this error type.
	CauserFeePayer causerSign = "fee-payer"
)

// InsufficientError is the error type to describe insufficient balance errors with details.
type InsufficientError struct {
	Type   balanceErrorType
	Need   *big.Int
	Have   *big.Int
	Causer causerSign
}

// NewInsufficientError is a constructor for InsufficientError.
func NewInsufficientError(type_ balanceErrorType, need, have *big.Int) *InsufficientError {
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

// clarify returns formed error with Need and Have values set.
func (e *InsufficientError) clarify(need, have *big.Int) *InsufficientError {
	return &InsufficientError{e.Type, need, have, e.Causer}
}

// setCauser updates InsufficientError with provided causer.
func (e *InsufficientError) setCauser(causer causerSign) *InsufficientError {
	e.Causer = causer
	return e
}
