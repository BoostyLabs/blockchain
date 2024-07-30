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
