// Copyright (C) 2024 Creditor Corp. Group.
// See LICENSE for copying information.

package txbuilder

import (
	"errors"
)

// ErrUnknownInputsHelpingKey defines that inputs help keys is unknown.
var ErrUnknownInputsHelpingKey = errors.New("unknown inputs help keys")

// InputsHelpingKey defines type for additional data in PSBT Unknowns field
// to distinguish input types and their indexes.
type InputsHelpingKey byte

const (
	// TaprootInputsHelpingKey defines key for taproot inputs.
	TaprootInputsHelpingKey InputsHelpingKey = 0x10
	// PaymentInputsHelpingKey defines key for payment (btc) inputs.
	PaymentInputsHelpingKey InputsHelpingKey = 0x20
	// FeePayerTaprootInputsHelpingKey defines key for taproot inputs for fee payer.
	FeePayerTaprootInputsHelpingKey InputsHelpingKey = 0x11
	// FeePayerPaymentInputsHelpingKey defines key for payment (btc) inputs for fee payer.
	FeePayerPaymentInputsHelpingKey InputsHelpingKey = 0x21
)

// InputsHelpingKeyFromBytes parses bytes array into InputsHelpingKey if any.
func InputsHelpingKeyFromBytes(b []byte) (InputsHelpingKey, error) {
	if len(b) != 1 {
		return 0, ErrUnknownInputsHelpingKey
	}

	switch b[0] {
	case TaprootInputsHelpingKey.Byte():
		return TaprootInputsHelpingKey, nil
	case PaymentInputsHelpingKey.Byte():
		return PaymentInputsHelpingKey, nil
	case FeePayerTaprootInputsHelpingKey.Byte():
		return FeePayerTaprootInputsHelpingKey, nil
	case FeePayerPaymentInputsHelpingKey.Byte():
		return FeePayerPaymentInputsHelpingKey, nil
	}

	return 0, ErrUnknownInputsHelpingKey
}

// Byte returns InputsHelpingKey as byte.
func (k InputsHelpingKey) Byte() byte {
	return byte(k)
}

// Bytes returns InputsHelpingKey as bytes array.
func (k InputsHelpingKey) Bytes() []byte {
	return []byte{byte(k)}
}
