// Copyright (C) 2024 Creditor Corp. Group.
// See LICENSE for copying information.

package txbuilder_test

import (
	"encoding/base64"
	"encoding/hex"
	"errors"
	"math/big"
	"math/rand"
	"testing"

	"github.com/btcsuite/btcd/chaincfg"
	"github.com/stretchr/testify/require"

	"blockchain/bitcoin"
	"blockchain/bitcoin/ord/runes"
	"blockchain/bitcoin/txbuilder"
)

func TestTxBuilder(t *testing.T) {
	txBuilder := txbuilder.NewTxBuilder(&chaincfg.TestNet3Params)

	t.Run("SelectUTXO", func(t *testing.T) {
		utxos := []bitcoin.UTXO{ // sorted by btc utxos.
			{Amount: big.NewInt(150000)},
			{Amount: big.NewInt(75000)},
			{Amount: big.NewInt(25000)},
			{Amount: big.NewInt(10000)},
			{Amount: big.NewInt(5000)},
			{Amount: big.NewInt(546)},
		}

		tests := []struct {
			minAmount     *big.Int
			totalAmount   *big.Int
			requiredUTXOs int
			utxos         []*bitcoin.UTXO
			err           error
		}{
			{big.NewInt(150000), big.NewInt(150000), 1, []*bitcoin.UTXO{&utxos[0]}, nil},
			{big.NewInt(149000), big.NewInt(150000), 1, []*bitcoin.UTXO{&utxos[0]}, nil},
			{big.NewInt(75000), big.NewInt(75000), 1, []*bitcoin.UTXO{&utxos[1]}, nil},
			{big.NewInt(74000), big.NewInt(75000), 1, []*bitcoin.UTXO{&utxos[1]}, nil},
			{big.NewInt(150000), big.NewInt(150546), 2, []*bitcoin.UTXO{&utxos[0], &utxos[5]}, nil},
			{big.NewInt(10020), big.NewInt(25546), 2, []*bitcoin.UTXO{&utxos[2], &utxos[5]}, nil},
			{big.NewInt(11000), big.NewInt(30546), 3, []*bitcoin.UTXO{&utxos[2], &utxos[5], &utxos[4]}, nil},
			{big.NewInt(255000), nil, 2, nil, bitcoin.ErrInsufficientNativeBalance},
			{big.NewInt(255000), big.NewInt(260000), 4, []*bitcoin.UTXO{&utxos[0], &utxos[1], &utxos[2], &utxos[3]}, nil},
			{big.NewInt(255000), big.NewInt(260546), 5, []*bitcoin.UTXO{&utxos[0], &utxos[1], &utxos[2], &utxos[3], &utxos[5]}, nil},
			{big.NewInt(200000), nil, 1, nil, bitcoin.ErrInsufficientNativeBalance},
			{big.NewInt(200000), nil, 8, nil, bitcoin.ErrInvalidUTXOAmount},
		}

		// by utxo test.
		utxoFn := func(utxo *bitcoin.UTXO) *big.Int { return utxo.Amount }
		for _, test := range tests {
			usedUTXOs, totalAmount, err := txbuilder.SelectUTXO(utxos, utxoFn, test.minAmount, test.requiredUTXOs, bitcoin.ErrInsufficientNativeBalance)
			require.Equal(t, test.err, err, test.minAmount.String())
			require.Equal(t, test.utxos, usedUTXOs, test.minAmount.String())
			require.EqualValues(t, test.totalAmount, totalAmount, test.minAmount.String())
		}

		testRuneID := runes.RuneID{Block: 20, TxID: 15}
		for idx := 0; idx < len(utxos); idx++ {
			k := rand.Uint32()
			if k%2 == 0 { // add random extra rune.
				utxos[idx].Runes = append(utxos[idx].Runes, bitcoin.RuneUTXO{
					RuneID: runes.RuneID{Block: uint64(k), TxID: k},
					Amount: big.NewInt(int64(k)),
				})
			}
			utxos[idx].Runes = append(utxos[idx].Runes, bitcoin.RuneUTXO{RuneID: testRuneID, Amount: utxos[idx].Amount})
		}

		// by rune test.
		runeFn := func(utxo *bitcoin.UTXO) *big.Int {
			for _, rune_ := range utxo.Runes {
				if rune_.RuneID == testRuneID {
					return rune_.Amount
				}
			}

			return big.NewInt(0)
		}
		for _, test := range tests {
			if errors.Is(test.err, bitcoin.ErrInsufficientNativeBalance) {
				test.err = bitcoin.ErrInsufficientRuneBalance
			}

			usedUTXOs, totalAmount, err := txbuilder.SelectUTXO(utxos, runeFn, test.minAmount, test.requiredUTXOs, bitcoin.ErrInsufficientRuneBalance)
			require.Equal(t, test.err, err, test.minAmount.String())
			require.Equal(t, test.utxos, usedUTXOs, test.minAmount.String())
			require.EqualValues(t, test.totalAmount, totalAmount, test.minAmount.String())
		}
	})

	t.Run("BuildRuneTransferTx", func(t *testing.T) {
		expectedTxB64 := "AgAAAAJGVyhT9+vWTklCoOBfu/NS6euHjw4nbtVD7EMc1lKK1wQAAAAA/////0ZXKFP369ZOSUKg4F+781Lp64ePDidu1UPsQxzWUorXAgAAAAD/////BiICAAAAAAAAIV9iaXRjb2luX3RyYW5zYWN0aW9uX3J1bmVfc2NyaXB0X1D4DAAAAAAAHF9iaXRjb2luX3RyYW5zYWN0aW9uX3NjcmlwdF8AAAAAAAAAAAxqXQkWAgDiCE2dGgEiAgAAAAAAACJRIC7q+7+Sry0fXghmy+vWF7mVYl83YnpjjJBN3SaFRMMQIgIAAAAAAAAiUSDJNteVAzZwcCPLnRgIbT6Xk34xxXH/zsdw2IQLjiBaZBvwDAAAAAAAIlEgyTbXlQM2cHAjy50YCG0+l5N+McVx/87HcNiEC44gWmQAAAAA"
		runeID := runes.RuneID{Block: 1122, TxID: 77}
		txBytes, _, _, _, err := txBuilder.BuildRunesTransferTx(txbuilder.BaseRunesTransferParams{
			RuneID: runeID,
			RuneUTXOs: []bitcoin.UTXO{
				{
					TxHash:  "d78a52d61c43ec43d56e270e8f87ebe952f3bb5fe0a042494ed6ebf753285746",
					Index:   4,
					Amount:  big.NewInt(546),
					Script:  []byte("_bitcoin_transaction_rune_script_"),
					Address: "tb1peymd09grxec8qg7tn5vqsmf7j7fhuvw9w8lua3msmzzqhr3qtfjqlj50zg",
					Runes:   []bitcoin.RuneUTXO{{RuneID: runeID, Amount: big.NewInt(7726)}},
				},
			},
			BaseUTXOs: []bitcoin.UTXO{
				{
					TxHash:  "d78a52d61c43ec43d56e270e8f87ebe952f3bb5fe0a042494ed6ebf753285746",
					Index:   2,
					Amount:  big.NewInt(850000), // 0.0085 BTC.
					Script:  []byte("_bitcoin_transaction_script_"),
					Address: "tb1peymd09grxec8qg7tn5vqsmf7j7fhuvw9w8lua3msmzzqhr3qtfjqlj50zg",
				},
			},
			TransferRuneAmount:      big.NewInt(3357),
			SatoshiPerKVByte:        big.NewInt(5000), // 5 sat/vB.
			RecipientTaprootAddress: "tb1p9m40h0uj4uk37hsgvm97h4shhx2kyhehvfax8rysfhwjdp2ycvgqtxqsu0",
			SenderTaprootAddress:    "tb1peymd09grxec8qg7tn5vqsmf7j7fhuvw9w8lua3msmzzqhr3qtfjqlj50zg",
			SenderPaymentAddress:    "tb1peymd09grxec8qg7tn5vqsmf7j7fhuvw9w8lua3msmzzqhr3qtfjqlj50zg",
		})
		require.NoError(t, err)
		require.EqualValues(t, expectedTxB64, base64.StdEncoding.EncodeToString(txBytes))
	})

	t.Run("BuildUserRuneTransferTx", func(t *testing.T) {
		expectedTxB64 := "AAAAAQAAAANwc2J0/wEA/VsBAgAAAAQQM59mtJnCrkg/Sj4O1EP2x+6JllKBu9ROcpHXH0m66gEAAAAA/////9Dz9Nv4btLdZfguL/SdH2s3EkM4VdQkCoUmSqDb7Kt4AAAAAAD/////sxo5ARA/WJnCcZ3a+q6JIzr5V8KGzJ1NWY1IyBwtKHACAAAAAP/////BtLuHj9VBtjFHSykxbn0QsjwAh8BROcT5tgpSqB+ibQAAAAAA/////wUAAAAAAAAAAA5qXQsWAgCN3p0BJpBOASICAAAAAAAAIlEgLur7v5KvLR9eCGbL69YXuZViXzdiemOMkE3dJoVEwxAiAgAAAAAAACJRIPkbPbLDMWl/+IzsLX9g4PFYSVOsLpfBuvfGFrjCjTtreB4AAAAAAAAXqRSqWI6UYef8rM0QtTTbRyLdcjEiwYfuAAAAAAAAABepFCUQTc/Trxe35YYAv98z2iZj4M3RhwAAAAAAAQErIgIAAAAAAAAiUSD5Gz2ywzFpf/iM7C1/YODxWElTrC6Xwbr3xha4wo07awEDBAEAAAABFyAp+mEcNhNVsILuWT/rNoAJqpxr0e02yZg+3NET+42jPwABASC8LwAAAAAAABepFCUQTc/Trxe35YYAv98z2iZj4M3RhwEDBAEAAAABBBYAFPPrPEU7ARQeYCvrLRM19r5Qe4E4AAEBIPAXAAAAAAAAF6kUJRBNz9OvF7flhgC/3zPaJmPgzdGHAQMEAQAAAAEEFgAU8+s8RTsBFB5gK+stEzX2vlB7gTgAAQEguAsAAAAAAAAXqRQlEE3P068Xt+WGAL/fM9omY+DN0YcBAwQBAAAAAQQWABTz6zxFOwEUHmAr6y0TNfa+UHuBOAAAAAAAAA=="
		runeID := runes.RuneID{Block: 2584333, TxID: 38}
		btx, _ := hex.DecodeString("5120f91b3db2c331697ff88cec2d7f60e0f1584953ac2e97c1baf7c616b8c28d3b6b")
		btx1, _ := hex.DecodeString("a91425104dcfd3af17b7e58600bfdf33da2663e0cdd187")
		txBytes, err := txBuilder.BuildUserTransferRuneTx(txbuilder.UserRunesTransferParams{
			BaseRunesTransferParams: txbuilder.BaseRunesTransferParams{
				RuneID: runeID,
				RuneUTXOs: []bitcoin.UTXO{
					{
						TxHash:  "eaba491fd791724ed4bb81529689eec7f643d40e3e4a3f48aec299b4669f3310",
						Index:   1,
						Amount:  big.NewInt(546),
						Script:  btx,
						Address: "tb1plydnmvkrx95hl7yvaskh7c8q79vyj5av96turwhhcctt3s5d8d4spjttqx",
						Runes:   []bitcoin.RuneUTXO{{RuneID: runeID, Amount: big.NewInt(15000)}},
					},
				},
				BaseUTXOs: []bitcoin.UTXO{
					{
						TxHash:  "78abecdba04a26850a24d455384312376b1f9df42f2ef865ddd26ef8dbf4f3d0",
						Index:   0,
						Amount:  big.NewInt(12220),
						Script:  btx1,
						Address: "2MvdCXCZZsJc3g9gsXhWdAoTwzoTX2vq3yv",
					},
					{
						TxHash:  "70282d1cc8488d594d9dcc86c257f93a2389aefada9d71c299583f1001391ab3",
						Index:   2,
						Amount:  big.NewInt(6128),
						Script:  btx1,
						Address: "2MvdCXCZZsJc3g9gsXhWdAoTwzoTX2vq3yv",
					},
					{
						TxHash:  "6da21fa8520ab6f9c43951c087003cb2107d6e31294b4731b641d58f87bbb4c1",
						Index:   0,
						Amount:  big.NewInt(3000),
						Script:  btx1,
						Address: "2MvdCXCZZsJc3g9gsXhWdAoTwzoTX2vq3yv",
					},
				},
				TransferRuneAmount:      big.NewInt(10000),
				RecipientTaprootAddress: "tb1p9m40h0uj4uk37hsgvm97h4shhx2kyhehvfax8rysfhwjdp2ycvgqtxqsu0",
				SenderTaprootAddress:    "tb1plydnmvkrx95hl7yvaskh7c8q79vyj5av96turwhhcctt3s5d8d4spjttqx",
				SenderPaymentAddress:    "2MvdCXCZZsJc3g9gsXhWdAoTwzoTX2vq3yv",
				SatoshiPerKVByte:        big.NewInt(24500),
				SatoshiCommissionAmount: big.NewInt(7800), // ~ 5$.
				RecipientPaymentAddress: "2N8mvwwUPfXt8FczXvE1UvM8ioVTW9LQLj1",
			},
			SenderTaprootPubKey: "29fa611c361355b082ee593feb368009aa9c6bd1ed36c9983edcd113fb8da33f",
			SenderPaymentPubKey: "03d17661b814dfaf3f7d6e70e8d4c8f5e6fdbe780a2c0373dd06ca7d75dc19f8be",
		})
		require.NoError(t, err)
		require.EqualValues(t, expectedTxB64, base64.StdEncoding.EncodeToString(txBytes))
	})
}
