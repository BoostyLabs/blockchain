// Copyright (C) 2025 Creditor Corp. Group.
// See LICENSE for copying information.

package utils

import (
	"github.com/btcsuite/btcd/btcec/v2"
	"github.com/btcsuite/btcd/btcec/v2/schnorr"
	"github.com/btcsuite/btcd/btcutil"
	"github.com/btcsuite/btcd/chaincfg"
	"github.com/btcsuite/btcd/txscript"
)

// NewTaprootAddressWithMultiSig generates taproot address with one leaf tapScript that holds multi-sig locking script build on provided privateKeys.
// NOTE: At least 2 private keys for multi-sig script generation is required.
func NewTaprootAddressWithMultiSig(chainParams *chaincfg.Params, masterPrivateKey *btcec.PrivateKey, privateKeys ...*btcec.PrivateKey) (*btcutil.AddressTaproot, error) {
	leafTapScript, err := NewTaprootMultiSigLeafTapScript(privateKeys...)
	if err != nil {
		return nil, err
	}

	return NewTaprootAddressFromScripts(chainParams, masterPrivateKey, leafTapScript)
}

// MustTaprootAddressWithMultiSig uses NewTaprootAddressWithMultiSig, panics in case of error.
func MustTaprootAddressWithMultiSig(chainParams *chaincfg.Params, masterPrivateKey *btcec.PrivateKey, privateKeys ...*btcec.PrivateKey) *btcutil.AddressTaproot {
	address, err := NewTaprootAddressWithMultiSig(chainParams, masterPrivateKey, privateKeys...)
	if err != nil {
		panic(err)
	}

	return address
}

// NewTaprootAddressFromScripts generates taproot address with tree built from provided leaf scripts.
func NewTaprootAddressFromScripts(chainParams *chaincfg.Params, masterPrivateKey *btcec.PrivateKey, leafScripts ...[]byte) (*btcutil.AddressTaproot, error) {
	tapScriptTree, err := NewTapScriptTreeFromRawScripts(leafScripts...)
	if err != nil {
		return nil, err
	}

	tapScriptRootHash := tapScriptTree.RootNode.TapHash()
	outputKey := txscript.ComputeTaprootOutputKey(masterPrivateKey.PubKey(), tapScriptRootHash[:])

	return btcutil.NewAddressTaproot(schnorr.SerializePubKey(outputKey), chainParams)
}

// MustTaprootAddressFromScripts uses NewTaprootAddressFromScripts, panics in case of error.
func MustTaprootAddressFromScripts(chainParams *chaincfg.Params, masterPrivateKey *btcec.PrivateKey, leafScripts ...[]byte) *btcutil.AddressTaproot {
	address, err := NewTaprootAddressFromScripts(chainParams, masterPrivateKey, leafScripts...)
	if err != nil {
		panic(err)
	}

	return address
}
