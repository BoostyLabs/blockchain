// Copyright (C) 2025 Creditor Corp. Group.
// See LICENSE for copying information.

package txbuilder_test

import (
	"math/big"
	"testing"

	"github.com/btcsuite/btcd/btcutil"
	"github.com/btcsuite/btcd/chaincfg"
	"github.com/stretchr/testify/require"

	"github.com/BoostyLabs/blockchain/bitcoin/txbuilder"
)

func TestNewDefaultPaymentDataFees(t *testing.T) {
	type unknownAddressType struct {
		btcutil.Address
	}
	chainParams := &chaincfg.MainNetParams
	tests := []struct {
		name    string
		address btcutil.Address
		sum     int64
		errStr  string
	}{
		{
			name:    "P2PK",
			address: btcutilAddress(t, "034cf21794aef30e95631c1e43b91412cd6e7d4734a44eec91eb45518aa81f6390", chainParams),
			errStr:  "unsupported address type: too old (P2PK)",
		},
		{
			name:    "P2PKH",
			address: btcutilAddress(t, "13yQfEYjcDZt72pyoyhAZgr3qe8vvwrwgH", chainParams),
			sum:     182,
		},
		{
			name:    "P2SH",
			address: btcutilAddress(t, "33qK7XqdJ3v5ySvnN9PugSRZsK2ahL5U1B", chainParams),
			sum:     329,
		},
		{
			name:    "P2WPKH",
			address: btcutilAddress(t, "bc1qhl00zlummcd9zc5ppu3klmnp4m7slkuvuq4lz0", chainParams),
			sum:     108,
		},
		{
			name:    "P2WSH",
			address: btcutilAddress(t, "bc1q6qj9wcc5lqshlyk427437e9rkevlcsuuypsukaqhmckjf8exdqmsa0pqxh", chainParams),
			sum:     148,
		},
		{
			name:    "P2TR",
			address: btcutilAddress(t, "bc1pvep2j3hym7tp69zyd3pwhe7a7exztrtg6wwhdd66fwx2ewfwag7qgawmug", chainParams),
			sum:     101,
		},
		{
			name:    "UNKNOWN",
			address: unknownAddressType{btcutilAddress(t, "1A1zP1eP5QGefi2DMPTfTL5SLmv7DivfNa", chainParams)},
			errStr:  "unsupported address type",
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got, err := txbuilder.NewDefaultPaymentDataFees(test.address)
			if test.errStr != "" {
				require.Error(t, err)
				require.Contains(t, err.Error(), test.errStr)
				return
			}
			require.NoError(t, err)
			require.NotNil(t, got)

			sum := big.NewInt(0)
			sum.Add(sum, got.InputSizeVBytes)
			sum.Add(sum, got.WitnessSizeVBytes)
			sum.Add(sum, got.OutputSizeVBytes)
			require.EqualValues(t, test.sum, sum.Int64())
		})
	}
}

func TestTxSizeEstimation(t *testing.T) {
	params := &chaincfg.MainNetParams
	bridge, err := txbuilder.NewDefaultPaymentDataFees(btcutilAddress(t, "bc1pvep2j3hym7tp69zyd3pwhe7a7exztrtg6wwhdd66fwx2ewfwag7qgawmug", params))
	require.NoError(t, err)

	bridge.WitnessSizeVBytes.SetInt64(92) // INFO: Not standard script.

	sender, err := txbuilder.NewDefaultPaymentDataFees(btcutilAddress(t, "bc1p9spaecu0aszezcs2f48uk0fxcqgxry0menjn8jwjnv9pjmsew09s3j36k8", params))
	require.NoError(t, err)

	feePayer, err := txbuilder.NewDefaultPaymentDataFees(btcutilAddress(t, "bc1qhl00zlummcd9zc5ppu3klmnp4m7slkuvuq4lz0", params))
	require.NoError(t, err)

	commission, err := txbuilder.NewDefaultPaymentDataFees(btcutilAddress(t, "33qK7XqdJ3v5ySvnN9PugSRZsK2ahL5U1B", params))
	require.NoError(t, err)

	tests := []struct {
		RoughTxSize *txbuilder.RoughTxSize
		Estimation  int
	}{
		{
			RoughTxSize: &txbuilder.RoughTxSize{
				IncludeHeader: true,
				Elements: []*txbuilder.EstimationElement{
					{PaymentDataFees: bridge, InputsNumber: 6, OutputsNumber: 2},
					{PaymentDataFees: sender, InputsNumber: 1, OutputsNumber: 0},
				},
				ExtraExpenses: big.NewInt(17),
			},
			Estimation: 970,
		},
		{
			RoughTxSize: &txbuilder.RoughTxSize{
				IncludeHeader: true,
				Elements: []*txbuilder.EstimationElement{
					{PaymentDataFees: bridge, InputsNumber: 0, OutputsNumber: 1},
					{PaymentDataFees: sender, InputsNumber: 1, OutputsNumber: 1},
					{PaymentDataFees: feePayer, InputsNumber: 1, OutputsNumber: 1},
					{PaymentDataFees: commission, InputsNumber: 0, OutputsNumber: 1},
				},
				ExtraExpenses: big.NewInt(17),
			},
			Estimation: 312,
		},
	}
	for _, test := range tests {
		require.EqualValues(t, test.Estimation, test.RoughTxSize.Estimate().Int64())
	}
}

func btcutilAddress(t *testing.T, address string, params *chaincfg.Params) btcutil.Address {
	addr, err := btcutil.DecodeAddress(address, params)
	require.NoError(t, err)

	return addr
}
