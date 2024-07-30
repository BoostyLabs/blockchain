// Copyright (C) 2024 Creditor Corp. Group.
// See LICENSE for copying information.

package signer

import (
	"bytes"
	"errors"

	"github.com/btcsuite/btcd/btcec/v2"
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

// signTaprootInputParams defines parameters for signTaprootInput method.
type signTaprootInputParams struct {
	packet       *psbt.Packet
	input        int
	inputFetcher txscript.PrevOutputFetcher
	privateKey   *btcec.PrivateKey
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
			packet:       packet,
			input:        input,
			inputFetcher: prevOutputFetcher,
			privateKey:   params.PrivateKey,
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

// signTaprootInput signs taproot input with or without witness script.
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
			tapLeaf        = txscript.NewBaseTapLeaf(input.WitnessScript)
			tapScriptTree  = txscript.AssembleTaprootScriptTree(tapLeaf)
			ctrlBlock      = tapScriptTree.LeafMerkleProofs[0].ToControlBlock(params.privateKey.PubKey())
			ctrlBlockBytes []byte
			sig            []byte
			leafHash       = tapLeaf.TapHash()
		)

		ctrlBlockBytes, err = ctrlBlock.ToBytes()
		if err != nil {
			return err
		}

		sig, err = txscript.RawTxInTapscriptSignature(
			params.packet.UnsignedTx, sigHashes, params.input,
			value, pkScript, tapLeaf, sigHashType, params.privateKey,
		)
		if err != nil {
			return err
		}

		if len(sig) > 64 {
			sig = sig[:64]
		}
		input.TaprootScriptSpendSig = []*psbt.TaprootScriptSpendSig{{
			XOnlyPubKey: params.privateKey.PubKey().SerializeCompressed()[1:],
			LeafHash:    leafHash.CloneBytes(),
			Signature:   sig,
			SigHash:     sigHashType,
		}}

		input.TaprootLeafScript = []*psbt.TaprootTapLeafScript{{
			ControlBlock: ctrlBlockBytes,
			Script:       tapLeaf.Script,
			LeafVersion:  tapLeaf.LeafVersion,
		}}

		return nil
	}

	witness, err = txscript.TaprootWitnessSignature(
		params.packet.UnsignedTx, sigHashes, params.input,
		value, pkScript, sigHashType, params.privateKey)
	if err != nil {
		return err
	}

	input.TaprootKeySpendSig = witness[0]

	return nil
}
