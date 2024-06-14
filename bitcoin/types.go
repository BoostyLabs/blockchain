// Copyright (C) 2024 Creditor Corp. Group.
// See LICENSE for copying information.

package bitcoin

import (
	"errors"
	"math/big"

	"blockchain/bitcoin/ord/runes"
)

var (
	// ErrInsufficientNativeBalance describes that there is not enough native balance to cover the payment.
	ErrInsufficientNativeBalance = errors.New("insufficient native balance")
	// ErrInsufficientRuneBalance describes that there is not enough rune balance to cover the payment.
	ErrInsufficientRuneBalance = errors.New("insufficient rune balance")
	// ErrInvalidUTXOAmount describes that there was invalid UTXO amount transmitted.
	ErrInvalidUTXOAmount = errors.New("invalid UTXO amount")
)

// UTXO describes unspent transaction output data.
type UTXO struct {
	TxHash  string
	Index   uint32   // output index in transaction outputs.
	Amount  *big.Int // in Satoshi.
	Script  []byte   // ScriptPubKey.
	Address string   // output recipient address.
	Runes   []RuneUTXO
}

// RuneUTXO describes linked to UTXO runes transaction.
type RuneUTXO struct {
	RuneID runes.RuneID
	Amount *big.Int // in rune units.
}
