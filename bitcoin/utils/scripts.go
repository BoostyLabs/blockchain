// Copyright (C) 2025 Creditor Corp. Group.
// See LICENSE for copying information.

package utils

import (
	"errors"

	"github.com/btcsuite/btcd/btcec/v2"
	"github.com/btcsuite/btcd/btcec/v2/schnorr"
	"github.com/btcsuite/btcd/btcutil/psbt"
	"github.com/btcsuite/btcd/txscript"
)

// NewTaprootMultiSigLeafTapScript generates N of N multi-sig locking script for taproot leaf.
// INFO: Script will have the next format: {<pubKey1> OP_CHECKSIG [<pubKey2> OP_CHECKSIG_ADD [<pubKey3> OP_CHECKSIG_ADD ...]] <signListSize> OP_EQUAL}.
// NOTE: At least 2 private keys for multi-sig script generation is required.
func NewTaprootMultiSigLeafTapScript(privateKeys ...*btcec.PrivateKey) ([]byte, error) {
	if len(privateKeys) < 2 {
		return nil, errors.New("at least 2 private keys are required")
	}
	if len(privateKeys) > 999 {
		return nil, errors.New("max allowed private keys: 999")
	}

	checkSigOp := byte(txscript.OP_CHECKSIG)
	scriptBuilder := txscript.NewScriptBuilder()
	for i, privateKey := range privateKeys {
		scriptBuilder.
			AddData(privateKey.PubKey().SerializeCompressed()[1:]).
			AddOp(checkSigOp)
		if i == 0 {
			checkSigOp = txscript.OP_CHECKSIGADD
		}
	}

	return scriptBuilder.
		AddInt64(int64(len(privateKeys))).
		AddOp(txscript.OP_EQUAL).
		Script()
}

// MustTaprootMultiSigLeafTapScript uses NewTaprootMultiSigLeafTapScript, panics in case of error.
func MustTaprootMultiSigLeafTapScript(privateKeys ...*btcec.PrivateKey) []byte {
	script, err := NewTaprootMultiSigLeafTapScript(privateKeys...)
	if err != nil {
		panic(err)
	}

	return script
}

// NewUnspendableScript builds provably unspendable script (e.g. OP_RETURN) with optional data added after.
// INFO: Def: https://en.bitcoin.it/wiki/OP_RETURN.
func NewUnspendableScript(msg ...byte) ([]byte, error) {
	scriptBuilder := txscript.NewScriptBuilder().AddOp(txscript.OP_RETURN)
	if len(msg) > 0 {
		scriptBuilder.AddData(msg)
	}

	return scriptBuilder.Script()
}

// MustUnspendableScript uses NewUnspendableScript, panics in case of error.
func MustUnspendableScript(msg ...byte) []byte {
	script, err := NewUnspendableScript(msg...)
	if err != nil {
		panic(err)
	}

	return script
}

// NewTapScriptTreeFromRawScripts builds tapScript tree from provided raw leaf scripts.
func NewTapScriptTreeFromRawScripts(leafScripts ...[]byte) (*txscript.IndexedTapScriptTree, error) {
	if len(leafScripts) == 0 {
		return nil, errors.New("no leaf scripts provided")
	}

	var tapLeafs = make([]txscript.TapLeaf, len(leafScripts))
	for i, leafScript := range leafScripts {
		tapLeafs[i] = txscript.NewBaseTapLeaf(leafScript)
	}

	return txscript.AssembleTaprootScriptTree(tapLeafs...), nil
}

// MustTapScriptTreeFromRawScripts uses NewTapScriptTreeFromRawScripts, panics in case of error.
func MustTapScriptTreeFromRawScripts(leafScripts ...[]byte) *txscript.IndexedTapScriptTree {
	tree, err := NewTapScriptTreeFromRawScripts(leafScripts...)
	if err != nil {
		panic(err)
	}

	return tree
}

// UpdatePSBTInputWithTapScriptLeafData updates provided psbt input with all data needed to sign taproot utxo.
func UpdatePSBTInputWithTapScriptLeafData(input *psbt.PInput, tapScriptTree *txscript.IndexedTapScriptTree) error {
	if len(input.TaprootInternalKey) == 0 {
		return errors.New("no taproot internal key provided")
	}
	if len(input.WitnessScript) == 0 {
		return errors.New("no witness script provided")
	}

	tapLeaf := txscript.NewBaseTapLeaf(input.WitnessScript)
	masterPublicKey, err := schnorr.ParsePubKey(input.TaprootInternalKey)
	if err != nil {
		return err
	}

	ctrlBlock := tapScriptTree.LeafMerkleProofs[0].ToControlBlock(masterPublicKey)
	tapLeafScript := &psbt.TaprootTapLeafScript{
		Script:      tapLeaf.Script,
		LeafVersion: tapLeaf.LeafVersion,
	}
	tapLeafScript.ControlBlock, err = ctrlBlock.ToBytes()
	if err != nil {
		return err
	}

	if len(input.TaprootLeafScript) == 0 {
		input.TaprootLeafScript = []*psbt.TaprootTapLeafScript{tapLeafScript}
	}

	if len(input.TaprootMerkleRoot) == 0 {
		input.TaprootMerkleRoot = ctrlBlock.RootHash(tapLeaf.Script)
	}

	return nil
}
