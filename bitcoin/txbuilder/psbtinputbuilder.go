// Copyright (C) 2024 Creditor Corp. Group.
// See LICENSE for copying information.

package txbuilder

import (
	"encoding/hex"
	"errors"

	"github.com/btcsuite/btcd/btcec/v2"
	"github.com/btcsuite/btcd/btcutil"
	"github.com/btcsuite/btcd/btcutil/psbt"
	"github.com/btcsuite/btcd/chaincfg"
	"github.com/btcsuite/btcd/txscript"
)

// ErrPSBTInputBuilder defines errors class for prepare address data method.
var ErrPSBTInputBuilder = errors.New("prepare address data")

const (
	// P2PK defines P2PK (public key) script type over which the address is built.
	P2PK = "P2PK"
	// P2PKH defines P2PK (public key hash) script type over which the address is built.
	P2PKH = "P2PKH"
	// P2SH defines P2SH (script hash) script type over which the address is built.
	P2SH = "P2SH"
	// P2WPKH defines P2WPKH (witness public key hash) script type over which the address is built.
	P2WPKH = "P2WPKH"
	// P2WSH defines P2WSH (witness script hash) script type over which the address is built.
	P2WSH = "P2WSH"
	// P2TR defines P2TR (taproot) script type over which the address is built.
	P2TR = "P2TR"
)

// PSBTInputBuilder is a helping tool to prepare psbt input based on address type.
type PSBTInputBuilder struct {
	params         *chaincfg.Params
	scriptType     string
	address        btcutil.Address
	publicKeyBytes []byte
	publicKey      *btcec.PublicKey
	xOnlyPubKey    []byte
	witnessScript  []byte
	redeemScript   []byte
}

// NewPSBTInputBuilder is a constructor for PSBTInputBuilder.
func NewPSBTInputBuilder(pubKey, address string, networkParams *chaincfg.Params) (pib *PSBTInputBuilder, err error) {
	pib = &PSBTInputBuilder{params: networkParams}

	defer func(err *error) {
		if err != nil && *err != nil {
			*err = errors.Join(ErrPSBTInputBuilder, *err)
		}
	}(&err)

	pib.publicKeyBytes, err = hex.DecodeString(pubKey)
	if err != nil {
		return pib, err
	}

	if len(pib.publicKeyBytes) == 33 {
		pib.xOnlyPubKey = pib.publicKeyBytes[1:]
		pib.publicKey, err = btcec.ParsePubKey(pib.publicKeyBytes)
		if err != nil {
			return pib, err
		}
	} else {
		pib.xOnlyPubKey = pib.publicKeyBytes
	}

	pib.address, err = btcutil.DecodeAddress(address, pib.params)
	if err != nil {
		return pib, err
	}

	switch pib.address.(type) {
	case *btcutil.AddressTaproot:
		pib.scriptType = P2TR
	case *btcutil.AddressWitnessPubKeyHash:
		pib.scriptType = P2WPKH
	case *btcutil.AddressWitnessScriptHash:
		pib.scriptType = P2WSH
	case *btcutil.AddressPubKeyHash:
		pib.scriptType = P2PKH
	case *btcutil.AddressPubKey:
		pib.scriptType = P2PK
	case *btcutil.AddressScriptHash:
		pib.scriptType = P2SH
	default:
		return pib, btcutil.ErrUnknownAddressType
	}

	switch pib.scriptType {
	case P2PK, P2PKH, P2SH:
		pib.redeemScript, err = txscript.PayToAddrScript(pib.address)
	case P2WPKH, P2WSH:
		pib.witnessScript, err = txscript.PayToAddrScript(pib.address)
	}
	if err != nil {
		return pib, err
	}

	return pib, nil
}

// PrepareInput updates input with required data based on address type.
func (pib *PSBTInputBuilder) PrepareInput(input *psbt.PInput) {
	switch pib.scriptType {
	case P2TR:
		input.TaprootInternalKey = pib.xOnlyPubKey
	case P2PK, P2PKH, P2SH:
		input.RedeemScript = pib.redeemScript
	case P2WPKH, P2WSH:
		input.WitnessScript = pib.witnessScript
	}
}

// InputsHelpingKey return InputsHelpingKey for wallet input indexes distinguishing.
func (pib *PSBTInputBuilder) InputsHelpingKey(isForFeePayer bool) InputsHelpingKey {
	switch {
	case isForFeePayer && pib.scriptType == P2TR:
		return FeePayerTaprootInputsHelpingKey
	case !isForFeePayer && pib.scriptType == P2TR:
		return TaprootInputsHelpingKey
	case isForFeePayer:
		return FeePayerPaymentInputsHelpingKey
	default:
		return PaymentInputsHelpingKey
	}
}

// ScriptType returns underlying script type.
func (pib *PSBTInputBuilder) ScriptType() string {
	return pib.scriptType
}
