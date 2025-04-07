// Copyright (C) 2024 Creditor Corp. Group.
// See LICENSE for copying information.

package signer_test

import (
	"bytes"
	"encoding/hex"
	"testing"

	"github.com/btcsuite/btcd/btcec/v2"
	"github.com/btcsuite/btcd/btcec/v2/schnorr"
	"github.com/btcsuite/btcd/btcutil"
	"github.com/btcsuite/btcd/btcutil/psbt"
	"github.com/btcsuite/btcd/chaincfg"
	"github.com/btcsuite/btcd/chaincfg/chainhash"
	"github.com/btcsuite/btcd/txscript"
	"github.com/btcsuite/btcd/wire"
	"github.com/stretchr/testify/require"

	"github.com/BoostyLabs/blockchain/bitcoin/ord/inscriptions"
	"github.com/BoostyLabs/blockchain/bitcoin/ord/runes"
	"github.com/BoostyLabs/blockchain/bitcoin/signer"
	"github.com/BoostyLabs/blockchain/bitcoin/utils"
)

func TestSigner(t *testing.T) {
	s := signer.NewSigner(&chaincfg.MainNetParams)

	privKey, err := btcec.NewPrivateKey()
	require.NoError(t, err)

	pubKey := privKey.PubKey()

	tx := wire.NewMsgTx(2)
	tx.AddTxIn(wire.NewTxIn(wire.NewOutPoint(mustHash("5aa4e4e957b467d07413aa75cdab5e4ce9ff2b714cd81b6af0e90bfee5ff070c"), 0), nil, nil))
	tx.AddTxOut(wire.NewTxOut(43000, mustHex("512015ae9a1bdfb273684b8c1107cc2dccf51f2235d8c79fe8b8e6555ad826415011")))

	t.Run("tap script", func(t *testing.T) {
		rr, _ := runes.NewRuneFromString("HELLO")
		insc := inscriptions.Inscription{Rune: rr, Body: make([]byte, 21)}

		inscriptionScript, err := insc.IntoScriptForWitness(pubKey.SerializeCompressed()[1:])
		require.NoError(t, err)

		inscriptionAddrStr, err := insc.IntoAddress(hex.EncodeToString(pubKey.SerializeCompressed()), &chaincfg.MainNetParams)
		require.NoError(t, err)

		inscriptionAddr, err := btcutil.DecodeAddress(inscriptionAddrStr, &chaincfg.MainNetParams)
		require.NoError(t, err)

		inscriptionAddrScript, err := txscript.PayToAddrScript(inscriptionAddr)
		require.NoError(t, err)

		packet, err := psbt.NewFromUnsignedTx(tx)
		require.NoError(t, err)

		packet.Inputs[0].WitnessUtxo = wire.NewTxOut(43000, inscriptionAddrScript)
		packet.Inputs[0].SighashType = txscript.SigHashAll
		packet.Inputs[0].TaprootInternalKey = pubKey.SerializeCompressed()[1:]
		packet.Inputs[0].WitnessScript = inscriptionScript

		packetBytes := bytes.NewBuffer(nil)
		err = packet.Serialize(packetBytes)
		require.NoError(t, err)

		signedPSBTBytes, err := s.SignTaproot(signer.SignTaprootParams{
			SerializedPSBT: packetBytes.Bytes(),
			Inputs:         []int{0},
			PrivateKey:     privKey,
		})
		require.NoError(t, err)

		signedPSBT, err := psbt.NewFromRawBytes(bytes.NewReader(signedPSBTBytes), false)
		require.NoError(t, err)
		require.NoError(t, psbt.Finalize(signedPSBT, 0))

		signedTx, err := psbt.Extract(signedPSBT)
		require.NoError(t, err)

		prevFetcher := txscript.NewCannedPrevOutputFetcher(copyBytes(packet.Inputs[0].WitnessUtxo.PkScript), packet.Inputs[0].WitnessUtxo.Value)
		sigHashes := txscript.NewTxSigHashes(signedTx, prevFetcher)

		vm, err := txscript.NewEngine(
			inscriptionAddrScript, signedTx, 0, txscript.StandardVerifyFlags,
			nil, sigHashes, 43000, prevFetcher,
		)
		require.NoError(t, err)
		require.NoError(t, vm.Execute())
	})

	t.Run("simple taproot", func(t *testing.T) {
		taprootAddr, err := btcutil.NewAddressTaproot(schnorr.SerializePubKey(txscript.ComputeTaprootKeyNoScript(pubKey)),
			&chaincfg.MainNetParams)
		require.NoError(t, err)

		taprootAddrAddrScript, err := txscript.PayToAddrScript(taprootAddr)
		require.NoError(t, err)

		packet, err := psbt.NewFromUnsignedTx(tx)
		require.NoError(t, err)

		packet.Inputs[0].WitnessUtxo = wire.NewTxOut(43000, taprootAddrAddrScript)
		packet.Inputs[0].SighashType = txscript.SigHashAll
		packet.Inputs[0].TaprootInternalKey = pubKey.SerializeCompressed()[1:]

		packetBytes := bytes.NewBuffer(nil)
		err = packet.Serialize(packetBytes)
		require.NoError(t, err)

		signedPSBTBytes, err := s.SignTaproot(signer.SignTaprootParams{
			SerializedPSBT: packetBytes.Bytes(),
			Inputs:         []int{0},
			PrivateKey:     privKey,
		})
		require.NoError(t, err)

		signedPSBT, err := psbt.NewFromRawBytes(bytes.NewReader(signedPSBTBytes), false)
		require.NoError(t, err)
		require.NoError(t, psbt.Finalize(signedPSBT, 0))

		signedTx, err := psbt.Extract(signedPSBT)
		require.NoError(t, err)

		prevFetcher := txscript.NewCannedPrevOutputFetcher(copyBytes(packet.Inputs[0].WitnessUtxo.PkScript), packet.Inputs[0].WitnessUtxo.Value)
		sigHashes := txscript.NewTxSigHashes(signedTx, prevFetcher)

		vm, err := txscript.NewEngine(
			taprootAddrAddrScript, signedTx, 0, txscript.StandardVerifyFlags,
			nil, sigHashes, 43000, prevFetcher,
		)
		require.NoError(t, err)
		require.NoError(t, vm.Execute())
	})
}

func TestSignerMulti(t *testing.T) {
	chainParams := &chaincfg.MainNetParams
	s := signer.NewSigner(chainParams)

	var (
		masterPrivateKey,
		tapScriptPrivateKey1, tapScriptPrivateKey2,
		tapScriptPrivateKey3, tapScriptPrivateKey4,
		invalidPrivateKey1, invalidPrivateKey2 *btcec.PrivateKey
		err error
	)
	for _, privateKeyP := range []**btcec.PrivateKey{
		&masterPrivateKey, &tapScriptPrivateKey1, &tapScriptPrivateKey2,
		&tapScriptPrivateKey3, &tapScriptPrivateKey4, &invalidPrivateKey1, &invalidPrivateKey2,
	} {
		*privateKeyP, err = btcec.NewPrivateKey()
		require.NoError(t, err)
	}

	// INFO: Build MultiSig 4 of 4.
	leafTapScript, err := utils.NewTaprootMultiSigLeafTapScript(tapScriptPrivateKey1, tapScriptPrivateKey2,
		tapScriptPrivateKey3, tapScriptPrivateKey4)
	require.NoErrorf(t, err, "leaf tapScript building")

	leafTapScriptUnspendable, err := utils.NewUnspendableScript([]byte("really_unspendable_!")...)
	require.NoErrorf(t, err, "leaf tapScript unspendable building")

	// INFO: Generate Taproot address.
	taprootAddress, err := utils.NewTaprootAddressFromScripts(chainParams, masterPrivateKey, leafTapScript, leafTapScriptUnspendable)
	require.NoErrorf(t, err, "taproot address generation")

	// INFO: Generate TapScript tree.
	tapScriptTree, err := utils.NewTapScriptTreeFromRawScripts(leafTapScript, leafTapScriptUnspendable)
	require.NoErrorf(t, err, "tapScript tree generation")

	invalidTapScriptTree, err := utils.NewTapScriptTreeFromRawScripts(leafTapScript)
	require.NoErrorf(t, err, "tapScript tree invalid generation")

	masterPublicKeyXOnly := masterPrivateKey.PubKey().SerializeCompressed()[1:]

	tests := []struct {
		name                 string
		masterPrivateKey     *btcec.PrivateKey
		tapScriptPrivateKeys []*btcec.PrivateKey
		tapScriptTree        *txscript.IndexedTapScriptTree
		err                  error
	}{
		{
			name:                 "valid 4 of 4 signature",
			tapScriptPrivateKeys: []*btcec.PrivateKey{tapScriptPrivateKey4, tapScriptPrivateKey3, tapScriptPrivateKey2, tapScriptPrivateKey1},
			tapScriptTree:        tapScriptTree,
		},
		{
			name:                 "not enough signatures (3 of 4 private keys for leaf signatures)",
			tapScriptPrivateKeys: []*btcec.PrivateKey{tapScriptPrivateKey3, tapScriptPrivateKey2, tapScriptPrivateKey1},
			tapScriptTree:        tapScriptTree,
			err:                  txscript.Error{ErrorCode: txscript.ErrInvalidStackOperation, Description: "index 0 is invalid for stack size 0"},
		},
		{
			name:                 "private keys invalid order",
			tapScriptPrivateKeys: []*btcec.PrivateKey{tapScriptPrivateKey1, tapScriptPrivateKey2, tapScriptPrivateKey3, tapScriptPrivateKey4},
			tapScriptTree:        tapScriptTree,
			err:                  txscript.Error{ErrorCode: txscript.ErrNullFail, Description: "signature not empty on failed checksig"},
		},
		{
			name:                 "invalid leaf keys",
			tapScriptPrivateKeys: []*btcec.PrivateKey{invalidPrivateKey1, tapScriptPrivateKey3, tapScriptPrivateKey2, invalidPrivateKey2},
			tapScriptTree:        tapScriptTree,
			err:                  txscript.Error{ErrorCode: txscript.ErrNullFail, Description: "signature not empty on failed checksig"},
		},
		{
			name:                 "unable to unlock by script spend path without correct script tree",
			tapScriptPrivateKeys: []*btcec.PrivateKey{tapScriptPrivateKey4, tapScriptPrivateKey3, tapScriptPrivateKey2, tapScriptPrivateKey1},
			err:                  txscript.Error{ErrorCode: txscript.ErrTaprootMerkleProofInvalid},
		},
		{
			name:                 "unable to unlock by script spend path with incorrect script tree",
			tapScriptPrivateKeys: []*btcec.PrivateKey{tapScriptPrivateKey4, tapScriptPrivateKey3, tapScriptPrivateKey2, tapScriptPrivateKey1},
			tapScriptTree:        invalidTapScriptTree,
			err:                  txscript.Error{ErrorCode: txscript.ErrTaprootMerkleProofInvalid},
		},
		{
			name:             "unlock with key spend path",
			masterPrivateKey: masterPrivateKey,
			tapScriptTree:    tapScriptTree,
		},
		{
			name:             "unlock with key spend path invalid private key",
			masterPrivateKey: invalidPrivateKey1,
			tapScriptTree:    tapScriptTree,
			err:              txscript.Error{ErrorCode: txscript.ErrTaprootSigInvalid},
		},
		{
			name:             "unable to unlock by key spend path without correct script tree",
			masterPrivateKey: invalidPrivateKey1,
			err:              txscript.Error{ErrorCode: txscript.ErrTaprootSigInvalid},
		},
		{
			name:             "unable to unlock by key spend path with incorrect script tree",
			masterPrivateKey: invalidPrivateKey1,
			tapScriptTree:    invalidTapScriptTree,
			err:              txscript.Error{ErrorCode: txscript.ErrTaprootSigInvalid},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			packetBytes := prepareTxPacketBytes(t, taprootAddress, masterPublicKeyXOnly, leafTapScript, test.tapScriptTree)

			var signedPSBTBytes []byte
			signedPSBTBytes, err = s.SignTaprootMulti(signer.SignTaprootMultiParams{
				SerializedPSBT:       packetBytes,
				Inputs:               []int{0},
				MasterPrivateKey:     test.masterPrivateKey,
				TapScriptPrivateKeys: test.tapScriptPrivateKeys,
			})
			require.NoError(t, err)

			err = prepareMultiSigEngine(t, signedPSBTBytes).Execute()
			require.ErrorIs(t, err, test.err)
		})
	}
}

func mustHex(s string) []byte {
	b, _ := hex.DecodeString(s)

	return b
}

func mustHash(s string) *chainhash.Hash {
	h, _ := chainhash.NewHashFromStr(s)

	return h
}

func copyBytes(b []byte) []byte {
	c := make([]byte, len(b))
	copy(c, b)

	return c
}

func prepareTxPacketBytes(t *testing.T, taprootAddress *btcutil.AddressTaproot, masterPubKeyXOlny,
	leafTapScript []byte, tapScriptTree *txscript.IndexedTapScriptTree) []byte {
	tx := wire.NewMsgTx(2)
	tx.AddTxIn(wire.NewTxIn(wire.NewOutPoint(mustHash("5aa4e4e957b467d07413aa75cdab5e4ce9ff2b714cd81b6af0e90bfee5ff070c"), 0), nil, nil))
	tx.AddTxOut(wire.NewTxOut(43000, mustHex("512015ae9a1bdfb273684b8c1107cc2dccf51f2235d8c79fe8b8e6555ad826415011")))

	taprootAddressScript, err := txscript.PayToAddrScript(taprootAddress)
	require.NoError(t, err)

	packet, err := psbt.NewFromUnsignedTx(tx)
	require.NoError(t, err)

	packet.Inputs[0].WitnessUtxo = wire.NewTxOut(43000, taprootAddressScript)
	packet.Inputs[0].SighashType = txscript.SigHashAll
	packet.Inputs[0].TaprootInternalKey = masterPubKeyXOlny
	packet.Inputs[0].WitnessScript = leafTapScript

	if tapScriptTree != nil {
		require.NoError(t, utils.UpdatePSBTInputWithTapScriptLeafData(&packet.Inputs[0], tapScriptTree))
	}

	packetBytes := bytes.NewBuffer(nil)
	err = packet.Serialize(packetBytes)
	require.NoError(t, err)

	return packetBytes.Bytes()
}

func prepareMultiSigEngine(t *testing.T, signedPSBTBytes []byte) *txscript.Engine {
	signedPSBT, err := psbt.NewFromRawBytes(bytes.NewReader(signedPSBTBytes), false)
	require.NoError(t, err)
	require.NoError(t, psbt.Finalize(signedPSBT, 0))

	signedTx, err := psbt.Extract(signedPSBT)
	require.NoError(t, err)

	prevFetcher := txscript.NewCannedPrevOutputFetcher(copyBytes(signedPSBT.Inputs[0].WitnessUtxo.PkScript), signedPSBT.Inputs[0].WitnessUtxo.Value)
	sigHashes := txscript.NewTxSigHashes(signedTx, prevFetcher)

	vm, err := txscript.NewEngine(
		signedPSBT.Inputs[0].WitnessUtxo.PkScript, signedTx, 0, txscript.StandardVerifyFlags,
		nil, sigHashes, 43000, prevFetcher,
	)
	require.NoError(t, err)

	return vm
}
