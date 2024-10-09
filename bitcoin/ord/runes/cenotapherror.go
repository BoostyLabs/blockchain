// Copyright (C) 2024 Creditor Corp. Group.
// See LICENSE for copying information.

package runes

const (
	// PointerCenotaphErrorType describes invalid pointer values.
	PointerCenotaphErrorType byte = 1
	// EtchingCenotaphErrorType describes invalid etching values.
	EtchingCenotaphErrorType byte = 2
	// MintCenotaphErrorType describes invalid mint values.
	MintCenotaphErrorType byte = 3
	// EdictsCenotaphErrorType describes invalid edict values.
	EdictsCenotaphErrorType byte = 4
)

// CenotaphError provides wide description of the cenotaph.
type CenotaphError struct {
	type_   byte
	message string
}

func (e *CenotaphError) Error() string {
	return e.message
}

// Type returns cenotaph error type.
func (e *CenotaphError) Type() byte {
	return e.type_
}
