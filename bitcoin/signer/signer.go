// Copyright (C) 2024 Creditor Corp. Group.
// See LICENSE for copying information.

package signer

import (
	"bytes"
	"errors"

	"github.com/btcsuite/btcd/btcec/v2"
	"github.com/btcsuite/btcd/btcec/v2/schnorr"
	"github.com/btcsuite/btcd/btcutil/psbt"
	"github.com/btcsuite/btcd/chaincfg"
	"github.com/btcsuite/btcd/txscript"
	"github.com/btcsuite/btcd/wire"
)

// SignTaprootParams defines parameters for SignTaproot method.
type SignTaprootParams struct {
	SerializedPSBT []byte
	Inputs         []int // inputs indexes.
	PrivateKey     *btcec.PrivateKey
}

// SignTaprootMultiParams defines parameters for SignTaprootMulti method.
//
// NOTE: TapScriptPrivateKeys must be in reverse order relatively to public keys in locking script!!!
// Ex.: Locking script pub keys order: {<pub1>, <pub2>, ..., <pubN>}, then private keys order must be: {<prN>, ... <pr2>, <pr1>}.
//
// INFO: Either MasterPrivateKey or TapScriptPrivateKeys must be provided.
type SignTaprootMultiParams struct {
	SerializedPSBT       []byte
	Inputs               []int               // inputs indexes.
	MasterPrivateKey     *btcec.PrivateKey   // primary key which is used to create taproot public key (not tweaked).
	TapScriptPrivateKeys []*btcec.PrivateKey // holds private keys needed to unlock MultiSig tapScript. Key-spend path will be used in case of empty array.
}

// signTaprootInputParams defines parameters for signTaprootInput method.
type signTaprootInputParams struct {
	packet               *psbt.Packet
	input                int
	inputFetcher         txscript.PrevOutputFetcher
	masterPrivateKey     *btcec.PrivateKey
	tapScriptPrivateKeys []*btcec.PrivateKey
}

// Signer provides transaction signing related logic.
type Signer struct {
	networkParams *chaincfg.Params
}

// NewSigner is a constructor for Signer.
func NewSigner(networkParams *chaincfg.Params) *Signer {
	return &Signer{
		networkParams: networkParams,
	}
}

// SignTaproot signs taproot inputs by provided indexes, returns updated serialized PSBT.
func (signer *Signer) SignTaproot(params SignTaprootParams) ([]byte, error) {
	packet, err := psbt.NewFromRawBytes(bytes.NewBuffer(params.SerializedPSBT), false)
	if err != nil {
		return nil, err
	}

	var (
		tx                   = packet.UnsignedTx
		prevOutputFetcherMap = make(map[wire.OutPoint]*wire.TxOut, len(tx.TxIn))
	)
	for idx, in := range packet.Inputs {
		prevOutputFetcherMap[tx.TxIn[idx].PreviousOutPoint] = in.WitnessUtxo
	}

	var prevOutputFetcher = txscript.NewMultiPrevOutFetcher(prevOutputFetcherMap)
	for _, input := range params.Inputs {
		if len(packet.Inputs) <= input {
			return nil, errors.New("invalid input index")
		}

		err = signer.signTaprootInput(signTaprootInputParams{
			packet:               packet,
			input:                input,
			inputFetcher:         prevOutputFetcher,
			masterPrivateKey:     params.PrivateKey,
			tapScriptPrivateKeys: []*btcec.PrivateKey{params.PrivateKey},
		})
		if err != nil {
			return nil, err
		}
	}

	w := bytes.NewBuffer(nil)
	err = packet.Serialize(w)
	if err != nil {
		return nil, err
	}

	return w.Bytes(), nil
}

// SignTaprootMulti signs taproot inputs by provided indexes using 1+ private keys, returns updated serialized PSBT.
func (signer *Signer) SignTaprootMulti(params SignTaprootMultiParams) ([]byte, error) {
	packet, err := psbt.NewFromRawBytes(bytes.NewBuffer(params.SerializedPSBT), false)
	if err != nil {
		return nil, err
	}

	var (
		tx                   = packet.UnsignedTx
		prevOutputFetcherMap = make(map[wire.OutPoint]*wire.TxOut, len(tx.TxIn))
	)
	for idx, in := range packet.Inputs {
		prevOutputFetcherMap[tx.TxIn[idx].PreviousOutPoint] = in.WitnessUtxo
	}

	var prevOutputFetcher = txscript.NewMultiPrevOutFetcher(prevOutputFetcherMap)
	for _, input := range params.Inputs {
		if len(packet.Inputs) <= input {
			return nil, errors.New("invalid input index")
		}

		err = signer.signTaprootInput(signTaprootInputParams{
			packet:               packet,
			input:                input,
			inputFetcher:         prevOutputFetcher,
			masterPrivateKey:     params.MasterPrivateKey,
			tapScriptPrivateKeys: params.TapScriptPrivateKeys,
		})
		if err != nil {
			return nil, err
		}
	}

	w := bytes.NewBuffer(nil)
	err = packet.Serialize(w)
	if err != nil {
		return nil, err
	}

	return w.Bytes(), nil
}

// signTaprootInput signs taproot input with or without witness script with provided private keys.
func (signer *Signer) signTaprootInput(params signTaprootInputParams) error {
	var (
		input       = &params.packet.Inputs[params.input]
		sigHashes   = txscript.NewTxSigHashes(params.packet.UnsignedTx, params.inputFetcher)
		value       = input.WitnessUtxo.Value
		pkScript    = input.WitnessUtxo.PkScript
		sigHashType = input.SighashType
		witness     wire.TxWitness
		err         error
	)

	if len(input.WitnessScript) != 0 {
		var (
			sig  []byte
			tsrd *taprootSignatureRequiredData
		)

		tsrd, err = recoverTaprootSignatureRequiredData(input)
		if err != nil {
			return err
		}

		if len(params.tapScriptPrivateKeys) == 0 {
			if params.masterPrivateKey == nil {
				return errors.New("either master private key or tapScript private keys list was expected")
			}

			input.TaprootKeySpendSig, err = txscript.RawTxInTaprootSignature(
				params.packet.UnsignedTx, sigHashes, params.input, value, pkScript,
				input.TaprootMerkleRoot, sigHashType, params.masterPrivateKey)

			return err
		}

		for _, privateKey := range params.tapScriptPrivateKeys {
			sig, err = txscript.RawTxInTapscriptSignature(
				params.packet.UnsignedTx, sigHashes, params.input,
				value, pkScript, tsrd.tapLeaf, sigHashType, privateKey,
			)
			if err != nil {
				return err
			}

			if len(sig) > 64 {
				sig = sig[:64]
			}
			input.TaprootScriptSpendSig = append(input.TaprootScriptSpendSig, &psbt.TaprootScriptSpendSig{
				XOnlyPubKey: privateKey.PubKey().SerializeCompressed()[1:],
				LeafHash:    tsrd.leafHash,
				Signature:   sig,
				SigHash:     sigHashType,
			})
		}

		return nil
	}

	witness, err = txscript.TaprootWitnessSignature(
		params.packet.UnsignedTx, sigHashes, params.input,
		value, pkScript, sigHashType, params.masterPrivateKey)
	if err != nil {
		return err
	}

	input.TaprootKeySpendSig = witness[0]

	return nil
}

type taprootSignatureRequiredData struct {
	ctrlBlock *txscript.ControlBlock
	tapLeaf   txscript.TapLeaf
	leafHash  []byte
}

// recoverTaprootSignatureRequiredData parses all needed data from PSBT or recover from WitnessScript if not found and updates provided input with it.
func recoverTaprootSignatureRequiredData(input *psbt.PInput) (tsrd *taprootSignatureRequiredData, err error) {
	if len(input.TaprootInternalKey) == 0 {
		return nil, errors.New("taproot internal key is empty")
	}

	var masterPublicKey *btcec.PublicKey
	masterPublicKey, err = schnorr.ParsePubKey(input.TaprootInternalKey)
	if err != nil {
		return nil, err
	}

	tsrd = new(taprootSignatureRequiredData)
	if len(input.TaprootLeafScript) > 0 && input.TaprootLeafScript[0] != nil {
		leafScriptData := input.TaprootLeafScript[0]
		tsrd.tapLeaf = txscript.NewTapLeaf(leafScriptData.LeafVersion, leafScriptData.Script)
		tsrd.ctrlBlock, err = txscript.ParseControlBlock(leafScriptData.ControlBlock)
		if err != nil {
			return nil, err
		}
	} else {
		tsrd.tapLeaf = txscript.NewBaseTapLeaf(input.WitnessScript)
		tapScriptTree := txscript.AssembleTaprootScriptTree(tsrd.tapLeaf)
		ctrlBlock := tapScriptTree.LeafMerkleProofs[0].ToControlBlock(masterPublicKey)
		tsrd.ctrlBlock = &ctrlBlock

		tapLeafScript := &psbt.TaprootTapLeafScript{
			Script:      tsrd.tapLeaf.Script,
			LeafVersion: tsrd.tapLeaf.LeafVersion,
		}
		tapLeafScript.ControlBlock, err = tsrd.ctrlBlock.ToBytes()
		if err != nil {
			return nil, err
		}

		input.TaprootLeafScript = []*psbt.TaprootTapLeafScript{tapLeafScript}
	}
	leafHash := tsrd.tapLeaf.TapHash()
	tsrd.leafHash = leafHash[:]

	if len(input.TaprootMerkleRoot) == 0 {
		input.TaprootMerkleRoot = tsrd.ctrlBlock.RootHash(tsrd.tapLeaf.Script)
	}

	return tsrd, nil
}
