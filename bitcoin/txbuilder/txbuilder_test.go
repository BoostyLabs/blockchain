// Copyright (C) 2024 Creditor Corp. Group.
// See LICENSE for copying information.

package txbuilder_test

import (
	"bytes"
	"encoding/base64"
	"errors"
	"fmt"
	"math/big"
	"math/rand"
	"testing"

	"github.com/btcsuite/btcd/btcutil/psbt"
	"github.com/btcsuite/btcd/chaincfg"
	"github.com/stretchr/testify/require"

	"github.com/BoostyLabs/blockchain/bitcoin"
	"github.com/BoostyLabs/blockchain/bitcoin/ord/inscriptions"
	"github.com/BoostyLabs/blockchain/bitcoin/ord/runes"
	"github.com/BoostyLabs/blockchain/bitcoin/txbuilder"
	"github.com/BoostyLabs/blockchain/internal/numbers"
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
			{big.NewInt(255000), nil, 2, nil, txbuilder.NewInsufficientError(txbuilder.InsufficientErrorTypeBitcoin, big.NewInt(255000), big.NewInt(225000))},
			{big.NewInt(255000), big.NewInt(260000), 4, []*bitcoin.UTXO{&utxos[0], &utxos[1], &utxos[2], &utxos[3]}, nil},
			{big.NewInt(255000), big.NewInt(260546), 5, []*bitcoin.UTXO{&utxos[0], &utxos[1], &utxos[2], &utxos[3], &utxos[5]}, nil},
			{big.NewInt(200000), nil, 1, nil, txbuilder.NewInsufficientError(txbuilder.InsufficientErrorTypeBitcoin, big.NewInt(200000), big.NewInt(150000))},
			{big.NewInt(200000), nil, 8, nil, txbuilder.ErrInvalidUTXOAmount},
		}

		// by utxo test.
		utxoFn := func(utxo *bitcoin.UTXO) *big.Int { return utxo.Amount }
		for _, test := range tests {
			usedUTXOs, totalAmount, err := txbuilder.SelectUTXO(utxos, utxoFn, test.minAmount, test.requiredUTXOs, txbuilder.InsufficientNativeBalanceError)
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
			if ie := new(txbuilder.InsufficientError); errors.As(test.err, &ie) {
				test.err = txbuilder.NewInsufficientError(txbuilder.InsufficientErrorTypeRune, ie.Need, ie.Have)
			}

			usedUTXOs, totalAmount, err := txbuilder.SelectUTXO(utxos, runeFn, test.minAmount, test.requiredUTXOs, txbuilder.InsufficientRuneBalanceError)
			require.Equal(t, test.err, err, test.minAmount.String())
			require.Equal(t, test.utxos, usedUTXOs, test.minAmount.String())
			require.EqualValues(t, test.totalAmount, totalAmount, test.minAmount.String())
		}
	})

	t.Run("BuildRuneTransferTx", func(t *testing.T) {
		runeID := runes.RuneID{Block: 1122, TxID: 77}
		tests := []struct {
			name          string
			expectedTxB64 string
			outputs       int
			params        txbuilder.BaseRunesTransferParams
		}{
			{
				name:          "transfer runes with change",
				expectedTxB64: "cHNidP8BAPICAAAAAkZXKFP369ZOSUKg4F+781Lp64ePDidu1UPsQxzWUorXBAAAAAD/////RlcoU/fr1k5JQqDgX7vzUunrh48OJ27VQ+xDHNZSitcCAAAAAP////8EAAAAAAAAAAAMal0JFgIA4ghNnRoBIgIAAAAAAAAiUSAu6vu/kq8tH14IZsvr1he5lWJfN2J6Y4yQTd0mhUTDECICAAAAAAAAIlEgyTbXlQM2cHAjy50YCG0+l5N+McVx/87HcNiEC44gWmQb8AwAAAAAACJRIMk215UDNnBwI8udGAhtPpeTfjHFcf/Ox3DYhAuOIFpkAAAAAAEQAQABEQEBAAEBKiICAAAAAAAAIV9iaXRjb2luX3RyYW5zYWN0aW9uX3J1bmVfc2NyaXB0XwEDBAEAAAABFyAp+mEcNhNVsILuWT/rNoAJqpxr0e02yZg+3NET+42jPwABASVQ+AwAAAAAABxfYml0Y29pbl90cmFuc2FjdGlvbl9zY3JpcHRfAQMEAQAAAAEXINF2YbgU368/fW5w6NTI9eb9vngKLANz3QbKfXXcGfi+AAAAAAA=",
				outputs:       4,
				params: txbuilder.BaseRunesTransferParams{
					RuneID: runeID,
					RunesSender: &txbuilder.PaymentData{
						UTXOs: []bitcoin.UTXO{
							{
								TxHash:  "d78a52d61c43ec43d56e270e8f87ebe952f3bb5fe0a042494ed6ebf753285746",
								Index:   4,
								Amount:  big.NewInt(546),
								Script:  []byte("_bitcoin_transaction_rune_script_"),
								Address: "tb1peymd09grxec8qg7tn5vqsmf7j7fhuvw9w8lua3msmzzqhr3qtfjqlj50zg",
								Runes:   []bitcoin.RuneUTXO{{RuneID: runeID, Amount: big.NewInt(7726)}},
							},
						},
						Address: "tb1peymd09grxec8qg7tn5vqsmf7j7fhuvw9w8lua3msmzzqhr3qtfjqlj50zg",
						PubKey:  "29fa611c361355b082ee593feb368009aa9c6bd1ed36c9983edcd113fb8da33f",
					},
					FeePayer: &txbuilder.PaymentData{
						UTXOs: []bitcoin.UTXO{
							{
								TxHash:  "d78a52d61c43ec43d56e270e8f87ebe952f3bb5fe0a042494ed6ebf753285746",
								Index:   2,
								Amount:  big.NewInt(850000), // 0.0085 BTC.
								Script:  []byte("_bitcoin_transaction_script_"),
								Address: "tb1peymd09grxec8qg7tn5vqsmf7j7fhuvw9w8lua3msmzzqhr3qtfjqlj50zg",
							},
						},
						Address: "tb1peymd09grxec8qg7tn5vqsmf7j7fhuvw9w8lua3msmzzqhr3qtfjqlj50zg",
						PubKey:  "03d17661b814dfaf3f7d6e70e8d4c8f5e6fdbe780a2c0373dd06ca7d75dc19f8be",
					},
					TransferRuneAmount:    big.NewInt(3357),
					SatoshiPerKVByte:      big.NewInt(5000), // 5 sat/vB.
					RunesRecipientAddress: "tb1p9m40h0uj4uk37hsgvm97h4shhx2kyhehvfax8rysfhwjdp2ycvgqtxqsu0",
				},
			},
			{
				name:          "transfer runes without change",
				expectedTxB64: "cHNidP8BAMUCAAAAAkZXKFP369ZOSUKg4F+781Lp64ePDidu1UPsQxzWUorXBAAAAAD/////RlcoU/fr1k5JQqDgX7vzUunrh48OJ27VQ+xDHNZSitcCAAAAAP////8DAAAAAAAAAAAKal0HAOIITa48ASICAAAAAAAAIlEgLur7v5KvLR9eCGbL69YXuZViXzdiemOMkE3dJoVEwxDT8gwAAAAAACJRIMk215UDNnBwI8udGAhtPpeTfjHFcf/Ox3DYhAuOIFpkAAAAAAEQAQABEQEBAAEBKiICAAAAAAAAIV9iaXRjb2luX3RyYW5zYWN0aW9uX3J1bmVfc2NyaXB0XwEDBAEAAAABFyAp+mEcNhNVsILuWT/rNoAJqpxr0e02yZg+3NET+42jPwABASVQ+AwAAAAAABxfYml0Y29pbl90cmFuc2FjdGlvbl9zY3JpcHRfAQMEAQAAAAEXINF2YbgU368/fW5w6NTI9eb9vngKLANz3QbKfXXcGfi+AAAAAA==",
				outputs:       3,
				params: txbuilder.BaseRunesTransferParams{
					RuneID: runeID,
					RunesSender: &txbuilder.PaymentData{
						UTXOs: []bitcoin.UTXO{
							{
								TxHash:  "d78a52d61c43ec43d56e270e8f87ebe952f3bb5fe0a042494ed6ebf753285746",
								Index:   4,
								Amount:  big.NewInt(546),
								Script:  []byte("_bitcoin_transaction_rune_script_"),
								Address: "tb1peymd09grxec8qg7tn5vqsmf7j7fhuvw9w8lua3msmzzqhr3qtfjqlj50zg",
								Runes:   []bitcoin.RuneUTXO{{RuneID: runeID, Amount: big.NewInt(7726)}},
							},
						},
						Address: "tb1peymd09grxec8qg7tn5vqsmf7j7fhuvw9w8lua3msmzzqhr3qtfjqlj50zg",
						PubKey:  "29fa611c361355b082ee593feb368009aa9c6bd1ed36c9983edcd113fb8da33f",
					},
					FeePayer: &txbuilder.PaymentData{
						UTXOs: []bitcoin.UTXO{
							{
								TxHash:  "d78a52d61c43ec43d56e270e8f87ebe952f3bb5fe0a042494ed6ebf753285746",
								Index:   2,
								Amount:  big.NewInt(850000), // 0.0085 BTC.
								Script:  []byte("_bitcoin_transaction_script_"),
								Address: "tb1peymd09grxec8qg7tn5vqsmf7j7fhuvw9w8lua3msmzzqhr3qtfjqlj50zg",
							},
						},
						Address: "tb1peymd09grxec8qg7tn5vqsmf7j7fhuvw9w8lua3msmzzqhr3qtfjqlj50zg",
						PubKey:  "03d17661b814dfaf3f7d6e70e8d4c8f5e6fdbe780a2c0373dd06ca7d75dc19f8be",
					},
					TransferRuneAmount:    big.NewInt(7726),
					SatoshiPerKVByte:      big.NewInt(5000), // 5 sat/vB.
					RunesRecipientAddress: "tb1p9m40h0uj4uk37hsgvm97h4shhx2kyhehvfax8rysfhwjdp2ycvgqtxqsu0",
				},
			},
			{
				name:          "burn only with change",
				expectedTxB64: "cHNidP8BAMcCAAAAAkZXKFP369ZOSUKg4F+781Lp64ePDidu1UPsQxzWUorXBAAAAAD/////RlcoU/fr1k5JQqDgX7vzUunrh48OJ27VQ+xDHNZSitcCAAAAAP////8DAAAAAAAAAAAMal0JFgEA4ghNuBcAIgIAAAAAAAAiUSDJNteVAzZwcCPLnRgIbT6Xk34xxXH/zsdw2IQLjiBaZNPyDAAAAAAAIlEgyTbXlQM2cHAjy50YCG0+l5N+McVx/87HcNiEC44gWmQAAAAAARABAAERAQEAAQEqIgIAAAAAAAAhX2JpdGNvaW5fdHJhbnNhY3Rpb25fcnVuZV9zY3JpcHRfAQMEAQAAAAEXICn6YRw2E1Wwgu5ZP+s2gAmqnGvR7TbJmD7c0RP7jaM/AAEBJVD4DAAAAAAAHF9iaXRjb2luX3RyYW5zYWN0aW9uX3NjcmlwdF8BAwQBAAAAARcg0XZhuBTfrz99bnDo1Mj15v2+eAosA3PdBsp9ddwZ+L4AAAAA",
				outputs:       3,
				params: txbuilder.BaseRunesTransferParams{
					RuneID: runeID,
					RunesSender: &txbuilder.PaymentData{
						UTXOs: []bitcoin.UTXO{
							{
								TxHash:  "d78a52d61c43ec43d56e270e8f87ebe952f3bb5fe0a042494ed6ebf753285746",
								Index:   4,
								Amount:  big.NewInt(546),
								Script:  []byte("_bitcoin_transaction_rune_script_"),
								Address: "tb1peymd09grxec8qg7tn5vqsmf7j7fhuvw9w8lua3msmzzqhr3qtfjqlj50zg",
								Runes:   []bitcoin.RuneUTXO{{RuneID: runeID, Amount: big.NewInt(7726)}},
							},
						},
						Address: "tb1peymd09grxec8qg7tn5vqsmf7j7fhuvw9w8lua3msmzzqhr3qtfjqlj50zg",
						PubKey:  "29fa611c361355b082ee593feb368009aa9c6bd1ed36c9983edcd113fb8da33f",
					},
					FeePayer: &txbuilder.PaymentData{
						UTXOs: []bitcoin.UTXO{
							{
								TxHash:  "d78a52d61c43ec43d56e270e8f87ebe952f3bb5fe0a042494ed6ebf753285746",
								Index:   2,
								Amount:  big.NewInt(850000), // 0.0085 BTC.
								Script:  []byte("_bitcoin_transaction_script_"),
								Address: "tb1peymd09grxec8qg7tn5vqsmf7j7fhuvw9w8lua3msmzzqhr3qtfjqlj50zg",
							},
						},
						Address: "tb1peymd09grxec8qg7tn5vqsmf7j7fhuvw9w8lua3msmzzqhr3qtfjqlj50zg",
						PubKey:  "03d17661b814dfaf3f7d6e70e8d4c8f5e6fdbe780a2c0373dd06ca7d75dc19f8be",
					},
					BurnRuneAmount:        big.NewInt(3000),
					SatoshiPerKVByte:      big.NewInt(5000), // 5 sat/vB.
					RunesRecipientAddress: "tb1p9m40h0uj4uk37hsgvm97h4shhx2kyhehvfax8rysfhwjdp2ycvgqtxqsu0",
				},
			},
			{
				name:          "transfer runes with burn without change",
				expectedTxB64: "cHNidP8BAMoCAAAAAkZXKFP369ZOSUKg4F+781Lp64ePDidu1UPsQxzWUorXBAAAAAD/////RlcoU/fr1k5JQqDgX7vzUunrh48OJ27VQ+xDHNZSitcCAAAAAP////8DAAAAAAAAAAAPal0MAOIITfYkAQAAuBcAIgIAAAAAAAAiUSAu6vu/kq8tH14IZsvr1he5lWJfN2J6Y4yQTd0mhUTDENPyDAAAAAAAIlEgyTbXlQM2cHAjy50YCG0+l5N+McVx/87HcNiEC44gWmQAAAAAARABAAERAQEAAQEqIgIAAAAAAAAhX2JpdGNvaW5fdHJhbnNhY3Rpb25fcnVuZV9zY3JpcHRfAQMEAQAAAAEXICn6YRw2E1Wwgu5ZP+s2gAmqnGvR7TbJmD7c0RP7jaM/AAEBJVD4DAAAAAAAHF9iaXRjb2luX3RyYW5zYWN0aW9uX3NjcmlwdF8BAwQBAAAAARcg0XZhuBTfrz99bnDo1Mj15v2+eAosA3PdBsp9ddwZ+L4AAAAA",
				outputs:       3,
				params: txbuilder.BaseRunesTransferParams{
					RuneID: runeID,
					RunesSender: &txbuilder.PaymentData{
						UTXOs: []bitcoin.UTXO{
							{
								TxHash:  "d78a52d61c43ec43d56e270e8f87ebe952f3bb5fe0a042494ed6ebf753285746",
								Index:   4,
								Amount:  big.NewInt(546),
								Script:  []byte("_bitcoin_transaction_rune_script_"),
								Address: "tb1peymd09grxec8qg7tn5vqsmf7j7fhuvw9w8lua3msmzzqhr3qtfjqlj50zg",
								Runes:   []bitcoin.RuneUTXO{{RuneID: runeID, Amount: big.NewInt(7726)}},
							},
						},
						Address: "tb1peymd09grxec8qg7tn5vqsmf7j7fhuvw9w8lua3msmzzqhr3qtfjqlj50zg",
						PubKey:  "29fa611c361355b082ee593feb368009aa9c6bd1ed36c9983edcd113fb8da33f",
					},
					FeePayer: &txbuilder.PaymentData{
						UTXOs: []bitcoin.UTXO{
							{
								TxHash:  "d78a52d61c43ec43d56e270e8f87ebe952f3bb5fe0a042494ed6ebf753285746",
								Index:   2,
								Amount:  big.NewInt(850000), // 0.0085 BTC.
								Script:  []byte("_bitcoin_transaction_script_"),
								Address: "tb1peymd09grxec8qg7tn5vqsmf7j7fhuvw9w8lua3msmzzqhr3qtfjqlj50zg",
							},
						},
						Address: "tb1peymd09grxec8qg7tn5vqsmf7j7fhuvw9w8lua3msmzzqhr3qtfjqlj50zg",
						PubKey:  "03d17661b814dfaf3f7d6e70e8d4c8f5e6fdbe780a2c0373dd06ca7d75dc19f8be",
					},
					TransferRuneAmount:    big.NewInt(4726),
					BurnRuneAmount:        big.NewInt(3000),
					SatoshiPerKVByte:      big.NewInt(5000), // 5 sat/vB.
					RunesRecipientAddress: "tb1p9m40h0uj4uk37hsgvm97h4shhx2kyhehvfax8rysfhwjdp2ycvgqtxqsu0",
				},
			},
			{
				name:          "burn only without change",
				expectedTxB64: "cHNidP8BAJoCAAAAAkZXKFP369ZOSUKg4F+781Lp64ePDidu1UPsQxzWUorXBAAAAAD/////RlcoU/fr1k5JQqDgX7vzUunrh48OJ27VQ+xDHNZSitcCAAAAAP////8CAAAAAAAAAAAKal0HAOIITa48AIv1DAAAAAAAIlEgyTbXlQM2cHAjy50YCG0+l5N+McVx/87HcNiEC44gWmQAAAAAARABAAERAQEAAQEqIgIAAAAAAAAhX2JpdGNvaW5fdHJhbnNhY3Rpb25fcnVuZV9zY3JpcHRfAQMEAQAAAAEXICn6YRw2E1Wwgu5ZP+s2gAmqnGvR7TbJmD7c0RP7jaM/AAEBJVD4DAAAAAAAHF9iaXRjb2luX3RyYW5zYWN0aW9uX3NjcmlwdF8BAwQBAAAAARcg0XZhuBTfrz99bnDo1Mj15v2+eAosA3PdBsp9ddwZ+L4AAAA=",
				outputs:       2,
				params: txbuilder.BaseRunesTransferParams{
					RuneID: runeID,
					RunesSender: &txbuilder.PaymentData{
						UTXOs: []bitcoin.UTXO{
							{
								TxHash:  "d78a52d61c43ec43d56e270e8f87ebe952f3bb5fe0a042494ed6ebf753285746",
								Index:   4,
								Amount:  big.NewInt(546),
								Script:  []byte("_bitcoin_transaction_rune_script_"),
								Address: "tb1peymd09grxec8qg7tn5vqsmf7j7fhuvw9w8lua3msmzzqhr3qtfjqlj50zg",
								Runes:   []bitcoin.RuneUTXO{{RuneID: runeID, Amount: big.NewInt(7726)}},
							},
						},
						Address: "tb1peymd09grxec8qg7tn5vqsmf7j7fhuvw9w8lua3msmzzqhr3qtfjqlj50zg",
						PubKey:  "29fa611c361355b082ee593feb368009aa9c6bd1ed36c9983edcd113fb8da33f",
					},
					FeePayer: &txbuilder.PaymentData{
						UTXOs: []bitcoin.UTXO{
							{
								TxHash:  "d78a52d61c43ec43d56e270e8f87ebe952f3bb5fe0a042494ed6ebf753285746",
								Index:   2,
								Amount:  big.NewInt(850000), // 0.0085 BTC.
								Script:  []byte("_bitcoin_transaction_script_"),
								Address: "tb1peymd09grxec8qg7tn5vqsmf7j7fhuvw9w8lua3msmzzqhr3qtfjqlj50zg",
							},
						},
						Address: "tb1peymd09grxec8qg7tn5vqsmf7j7fhuvw9w8lua3msmzzqhr3qtfjqlj50zg",
						PubKey:  "03d17661b814dfaf3f7d6e70e8d4c8f5e6fdbe780a2c0373dd06ca7d75dc19f8be",
					},
					BurnRuneAmount:        big.NewInt(7726),
					SatoshiPerKVByte:      big.NewInt(5000), // 5 sat/vB.
					RunesRecipientAddress: "tb1p9m40h0uj4uk37hsgvm97h4shhx2kyhehvfax8rysfhwjdp2ycvgqtxqsu0",
				},
			},
		}
		for _, test := range tests {
			t.Run(test.name, func(t *testing.T) {
				result, err := txBuilder.BuildRunesTransferTx(test.params)
				require.NoError(t, err)
				require.EqualValues(t, test.expectedTxB64, base64.StdEncoding.EncodeToString(result.SerializedPSBT))

				p, err := psbt.NewFromRawBytes(bytes.NewBuffer(result.SerializedPSBT), false)
				require.NoError(t, err)
				require.Len(t, p.Outputs, test.outputs)
			})
		}
	})

	t.Run("BuildBTCTransferTx", func(t *testing.T) {
		tests := []struct {
			expectedTxB64 string
			params        txbuilder.BaseBTCTransferParams
		}{
			{
				"cHNidP8BAH4CAAAAAUZXKFP369ZOSUKg4F+781Lp64ePDidu1UPsQxzWUorXAgAAAAD/////AjxzAAAAAAAAIlEgLur7v5KvLR9eCGbL69YXuZViXzdiemOMkE3dJoVEwxDvgQwAAAAAABepFKpYjpRh5/yszRC1NNtHIt1yMSLBhwAAAAABIAEAAAEBJVD4DAAAAAAAHF9iaXRjb2luX3RyYW5zYWN0aW9uX3NjcmlwdF8BAwQBAAAAAQQWABTz6zxFOwEUHmAr6y0TNfa+UHuBOAAAAA==",
				txbuilder.BaseBTCTransferParams{
					TransferSatoshiAmount: big.NewInt(29500), // 0.000295 BTC.
					Sender: &txbuilder.PaymentData{
						UTXOs: []bitcoin.UTXO{
							{
								TxHash:  "d78a52d61c43ec43d56e270e8f87ebe952f3bb5fe0a042494ed6ebf753285746",
								Index:   2,
								Amount:  big.NewInt(850000), // 0.0085 BTC.
								Script:  []byte("_bitcoin_transaction_script_"),
								Address: "2N8mvwwUPfXt8FczXvE1UvM8ioVTW9LQLj1",
							},
							{
								TxHash:  "d78a52d61c43ec43d56e270e8f87ebe952f3bb5fe0a042494ed6ebf753285746",
								Index:   4,
								Amount:  big.NewInt(27000), // 0.00027 BTC.
								Script:  []byte("_bitcoin_transaction_script_"),
								Address: "2N8mvwwUPfXt8FczXvE1UvM8ioVTW9LQLj1",
							},
						},
						Address: "2N8mvwwUPfXt8FczXvE1UvM8ioVTW9LQLj1",
						PubKey:  "03d17661b814dfaf3f7d6e70e8d4c8f5e6fdbe780a2c0373dd06ca7d75dc19f8be",
					},
					SatoshiPerKVByte: big.NewInt(5000), // 5 sat/vB.
					RecipientAddress: "tb1p9m40h0uj4uk37hsgvm97h4shhx2kyhehvfax8rysfhwjdp2ycvgqtxqsu0",
				},
			},
			{
				"cHNidP8BAIkCAAAAAUZXKFP369ZOSUKg4F+781Lp64ePDidu1UPsQxzWUorXAgAAAAD/////AjxzAAAAAAAAIlEgLur7v5KvLR9eCGbL69YXuZViXzdiemOMkE3dJoVEwxDvgQwAAAAAACJRIMk215UDNnBwI8udGAhtPpeTfjHFcf/Ox3DYhAuOIFpkAAAAAAEQAQAAAQElUPgMAAAAAAAcX2JpdGNvaW5fdHJhbnNhY3Rpb25fc2NyaXB0XwEDBAEAAAABFyAp+mEcNhNVsILuWT/rNoAJqpxr0e02yZg+3NET+42jPwAAAA==",
				txbuilder.BaseBTCTransferParams{
					TransferSatoshiAmount: big.NewInt(29500), // 0.000295 BTC.
					Sender: &txbuilder.PaymentData{
						UTXOs: []bitcoin.UTXO{
							{
								TxHash:  "d78a52d61c43ec43d56e270e8f87ebe952f3bb5fe0a042494ed6ebf753285746",
								Index:   2,
								Amount:  big.NewInt(850000), // 0.0085 BTC.
								Script:  []byte("_bitcoin_transaction_script_"),
								Address: "tb1peymd09grxec8qg7tn5vqsmf7j7fhuvw9w8lua3msmzzqhr3qtfjqlj50zg",
							},
							{
								TxHash:  "d78a52d61c43ec43d56e270e8f87ebe952f3bb5fe0a042494ed6ebf753285746",
								Index:   4,
								Amount:  big.NewInt(27000), // 0.00027 BTC.
								Script:  []byte("_bitcoin_transaction_script_"),
								Address: "tb1peymd09grxec8qg7tn5vqsmf7j7fhuvw9w8lua3msmzzqhr3qtfjqlj50zg",
							},
						},
						Address: "tb1peymd09grxec8qg7tn5vqsmf7j7fhuvw9w8lua3msmzzqhr3qtfjqlj50zg",
						PubKey:  "29fa611c361355b082ee593feb368009aa9c6bd1ed36c9983edcd113fb8da33f",
					},
					SatoshiPerKVByte: big.NewInt(5000), // 5 sat/vB.
					RecipientAddress: "tb1p9m40h0uj4uk37hsgvm97h4shhx2kyhehvfax8rysfhwjdp2ycvgqtxqsu0",
				},
			},
			{
				"cHNidP8BAPsCAAAAA0ZXKFP369ZOSUKg4F+781Lp64ePDidu1UPsQxzWUorXAgAAAAD/////RlcoU/fr1k5JQqDgX7vzUunrh48OJ27VQ+xDHNZSitcEAAAAAP////9GVyhT9+vWTklCoOBfu/NS6euHjw4nbtVD7EMc1lKK1wQAAAAA/////wM8cwAAAAAAACJRIC7q+7+Sry0fXghmy+vWF7mVYl83YnpjjJBN3SaFRMMQ6AMAAAAAAAAXqRSqWI6UYef8rM0QtTTbRyLdcjEiwYfBLQwAAAAAACJRIMk215UDNnBwI8udGAhtPpeTfjHFcf/Ox3DYhAuOIFpkAAAAAAEgAgABAREBAgABASWsDQAAAAAAABxfYml0Y29pbl90cmFuc2FjdGlvbl9zY3JpcHRfAQMEAQAAAAEEFgAU8+s8RTsBFB5gK+stEzX2vlB7gTgAAQEleGkAAAAAAAAcX2JpdGNvaW5fdHJhbnNhY3Rpb25fc2NyaXB0XwEDBAEAAAABBBYAFPPrPEU7ARQeYCvrLRM19r5Qe4E4AAEBJQA1DAAAAAAAHF9iaXRjb2luX3RyYW5zYWN0aW9uX3NjcmlwdF8BAwQBAAAAARcgKfphHDYTVbCC7lk/6zaACaqca9HtNsmYPtzRE/uNoz8AAAAA",
				txbuilder.BaseBTCTransferParams{
					TransferSatoshiAmount: big.NewInt(29500), // 0.000295 BTC.
					Sender: &txbuilder.PaymentData{
						UTXOs: []bitcoin.UTXO{
							{
								TxHash:  "d78a52d61c43ec43d56e270e8f87ebe952f3bb5fe0a042494ed6ebf753285746",
								Index:   2,
								Amount:  big.NewInt(3500), // 0.000025 BTC.
								Script:  []byte("_bitcoin_transaction_script_"),
								Address: "2N8mvwwUPfXt8FczXvE1UvM8ioVTW9LQLj1",
							},
							{
								TxHash:  "d78a52d61c43ec43d56e270e8f87ebe952f3bb5fe0a042494ed6ebf753285746",
								Index:   4,
								Amount:  big.NewInt(27000), // 0.00027 BTC.
								Script:  []byte("_bitcoin_transaction_script_"),
								Address: "2N8mvwwUPfXt8FczXvE1UvM8ioVTW9LQLj1",
							},
						},
						Address: "2N8mvwwUPfXt8FczXvE1UvM8ioVTW9LQLj1",
						PubKey:  "03d17661b814dfaf3f7d6e70e8d4c8f5e6fdbe780a2c0373dd06ca7d75dc19f8be",
					},
					FeePayer: &txbuilder.PaymentData{
						UTXOs: []bitcoin.UTXO{
							{
								TxHash:  "d78a52d61c43ec43d56e270e8f87ebe952f3bb5fe0a042494ed6ebf753285746",
								Index:   2,
								Amount:  big.NewInt(850000), // 0.0085 BTC.
								Script:  []byte("_bitcoin_transaction_script_"),
								Address: "tb1peymd09grxec8qg7tn5vqsmf7j7fhuvw9w8lua3msmzzqhr3qtfjqlj50zg",
							},
							{
								TxHash:  "d78a52d61c43ec43d56e270e8f87ebe952f3bb5fe0a042494ed6ebf753285746",
								Index:   4,
								Amount:  big.NewInt(800000), // 0.008 BTC.
								Script:  []byte("_bitcoin_transaction_script_"),
								Address: "tb1peymd09grxec8qg7tn5vqsmf7j7fhuvw9w8lua3msmzzqhr3qtfjqlj50zg",
							},
						},
						Address: "tb1peymd09grxec8qg7tn5vqsmf7j7fhuvw9w8lua3msmzzqhr3qtfjqlj50zg",
						PubKey:  "29fa611c361355b082ee593feb368009aa9c6bd1ed36c9983edcd113fb8da33f",
					},
					SatoshiPerKVByte: big.NewInt(5000), // 5 sat/vB.
					RecipientAddress: "tb1p9m40h0uj4uk37hsgvm97h4shhx2kyhehvfax8rysfhwjdp2ycvgqtxqsu0",
				},
			},
		}
		for i, test := range tests {
			t.Run(fmt.Sprintf("test %d", i), func(t *testing.T) {
				result, err := txBuilder.BuildBTCTransferTx(test.params)
				require.NoError(t, err)
				require.EqualValues(t, test.expectedTxB64, base64.StdEncoding.EncodeToString(result.SerializedPSBT))
			})
		}
	})

	t.Run("BuildBaseInscriptionTx", func(t *testing.T) {
		rune_, err := runes.NewRuneFromString("HELLO")
		require.NoError(t, err)

		tests := []struct {
			expectedTxB64 string
			error         error
			params        txbuilder.BaseInscriptionTxParams
		}{
			{
				"cHNidP8BAH4CAAAAAUZXKFP369ZOSUKg4F+781Lp64ePDidu1UPsQxzWUorXBAAAAAD/////AsMGAAAAAAAAIlEgo5FkqP6gH/aAcA2jr3Pmcup6Y/YeKSLHDN3hMIcCiZWQXwAAAAAAABepFKpYjpRh5/yszRC1NNtHIt1yMSLBhwAAAAABIAEAAAEBJXhpAAAAAAAAHF9iaXRjb2luX3RyYW5zYWN0aW9uX3NjcmlwdF8BAwQBAAAAAQQWABTz6zxFOwEUHmAr6y0TNfa+UHuBOAAAAA==",
				nil,
				txbuilder.BaseInscriptionTxParams{
					Sender: &txbuilder.PaymentData{
						UTXOs: []bitcoin.UTXO{
							{
								TxHash:  "d78a52d61c43ec43d56e270e8f87ebe952f3bb5fe0a042494ed6ebf753285746",
								Index:   4,
								Amount:  big.NewInt(27000), // 0.00027 BTC.
								Script:  []byte("_bitcoin_transaction_script_"),
								Address: "2N8mvwwUPfXt8FczXvE1UvM8ioVTW9LQLj1",
							},
						},
						Address: "2N8mvwwUPfXt8FczXvE1UvM8ioVTW9LQLj1",
						PubKey:  "03d17661b814dfaf3f7d6e70e8d4c8f5e6fdbe780a2c0373dd06ca7d75dc19f8be",
					},
					SatoshiPerKVByte: big.NewInt(5000), // 5 sat/vB.
					Inscription: &inscriptions.Inscription{
						Rune: rune_,
						Body: []byte("test data"),
					},
					InscriptionBasePubKey: "02f58a2a986582ffd680e572f2413feea6ce05dad8bed004fe5a262198312867fa",
				},
			},
			{
				"", txbuilder.InsufficientNativeBalanceError,
				txbuilder.BaseInscriptionTxParams{
					Sender: &txbuilder.PaymentData{
						UTXOs: []bitcoin.UTXO{
							{
								TxHash:  "d78a52d61c43ec43d56e270e8f87ebe952f3bb5fe0a042494ed6ebf753285746",
								Index:   4,
								Amount:  big.NewInt(27000), // 0.00027 BTC.
								Script:  []byte("_bitcoin_transaction_script_"),
								Address: "2N8mvwwUPfXt8FczXvE1UvM8ioVTW9LQLj1",
							},
						},
						Address: "2N8mvwwUPfXt8FczXvE1UvM8ioVTW9LQLj1",
						PubKey:  "03d17661b814dfaf3f7d6e70e8d4c8f5e6fdbe780a2c0373dd06ca7d75dc19f8be",
					},
					SatoshiPerKVByte: big.NewInt(5000), // 5 sat/vB.
					Inscription: &inscriptions.Inscription{
						Rune: rune_,
						Body: []byte("test data"),
					},
					InscriptionBasePubKey:     "02f58a2a986582ffd680e572f2413feea6ce05dad8bed004fe5a262198312867fa",
					SatoshiCommissionAmount:   big.NewInt(100000),
					CommissionReceiverAddress: "2N8mvwwUPfXt8FczXvE1UvM8ioVTW9LQLj1",
				},
			},
			{
				"cHNidP8BAJ4CAAAAAUZXKFP369ZOSUKg4F+781Lp64ePDidu1UPsQxzWUorXAgAAAAD/////A8MGAAAAAAAAIlEgo5FkqP6gH/aAcA2jr3Pmcup6Y/YeKSLHDN3hMIcCiZWghgEAAAAAABepFKpYjpRh5/yszRC1NNtHIt1yMSLBhzJnCwAAAAAAF6kUqliOlGHn/KzNELU020ci3XIxIsGHAAAAAAEgAQAAAQElUPgMAAAAAAAcX2JpdGNvaW5fdHJhbnNhY3Rpb25fc2NyaXB0XwEDBAEAAAABBBYAFPPrPEU7ARQeYCvrLRM19r5Qe4E4AAAAAA==",
				nil,
				txbuilder.BaseInscriptionTxParams{
					Sender: &txbuilder.PaymentData{
						UTXOs: []bitcoin.UTXO{
							{
								TxHash:  "d78a52d61c43ec43d56e270e8f87ebe952f3bb5fe0a042494ed6ebf753285746",
								Index:   2,
								Amount:  big.NewInt(850000), // 0.0085 BTC.
								Script:  []byte("_bitcoin_transaction_script_"),
								Address: "2N8mvwwUPfXt8FczXvE1UvM8ioVTW9LQLj1",
							},
							{
								TxHash:  "d78a52d61c43ec43d56e270e8f87ebe952f3bb5fe0a042494ed6ebf753285746",
								Index:   4,
								Amount:  big.NewInt(27000), // 0.00027 BTC.
								Script:  []byte("_bitcoin_transaction_script_"),
								Address: "2N8mvwwUPfXt8FczXvE1UvM8ioVTW9LQLj1",
							},
						},
						Address: "2N8mvwwUPfXt8FczXvE1UvM8ioVTW9LQLj1",
						PubKey:  "03d17661b814dfaf3f7d6e70e8d4c8f5e6fdbe780a2c0373dd06ca7d75dc19f8be",
					},
					SatoshiPerKVByte: big.NewInt(5000), // 5 sat/vB.
					Inscription: &inscriptions.Inscription{
						Rune: rune_,
						Body: []byte("test data"),
					},
					InscriptionBasePubKey:     "02f58a2a986582ffd680e572f2413feea6ce05dad8bed004fe5a262198312867fa",
					SatoshiCommissionAmount:   big.NewInt(100000),
					CommissionReceiverAddress: "2N8mvwwUPfXt8FczXvE1UvM8ioVTW9LQLj1",
				},
			},
			{
				"cHNidP8BAH4CAAAAAUZXKFP369ZOSUKg4F+781Lp64ePDidu1UPsQxzWUorXBAAAAAD/////AjMMAAAAAAAAIlEgo5FkqP6gH/aAcA2jr3Pmcup6Y/YeKSLHDN3hMIcCiZUgWgAAAAAAABepFKpYjpRh5/yszRC1NNtHIt1yMSLBhwAAAAABIAEAAAEBJXhpAAAAAAAAHF9iaXRjb2luX3RyYW5zYWN0aW9uX3NjcmlwdF8BAwQBAAAAAQQWABTz6zxFOwEUHmAr6y0TNfa+UHuBOAAAAA==",
				nil,
				txbuilder.BaseInscriptionTxParams{
					Sender: &txbuilder.PaymentData{
						UTXOs: []bitcoin.UTXO{
							{
								TxHash:  "d78a52d61c43ec43d56e270e8f87ebe952f3bb5fe0a042494ed6ebf753285746",
								Index:   4,
								Amount:  big.NewInt(27000), // 0.00027 BTC.
								Script:  []byte("_bitcoin_transaction_script_"),
								Address: "2N8mvwwUPfXt8FczXvE1UvM8ioVTW9LQLj1",
							},
						},
						Address: "2N8mvwwUPfXt8FczXvE1UvM8ioVTW9LQLj1",
						PubKey:  "03d17661b814dfaf3f7d6e70e8d4c8f5e6fdbe780a2c0373dd06ca7d75dc19f8be",
					},
					SatoshiPerKVByte: big.NewInt(5000), // 5 sat/vB.
					Inscription: &inscriptions.Inscription{
						Rune: rune_,
						Body: []byte("test data"),
					},
					InscriptionBasePubKey:  "02f58a2a986582ffd680e572f2413feea6ce05dad8bed004fe5a262198312867fa",
					PremineSplittingFactor: 3,
				},
			},
		}
		for i, test := range tests {
			t.Run(fmt.Sprintf("test %d", i), func(t *testing.T) {
				result, err := txBuilder.BuildInscriptionTx(test.params)
				require.ErrorIs(t, err, test.error)
				require.EqualValues(t, test.expectedTxB64, base64.StdEncoding.EncodeToString(result.SerializedPSBT))
			})
		}
	})

	t.Run("BuildRuneEtchTx", func(t *testing.T) {
		rune_, err := runes.NewRuneFromString("HELLO")
		require.NoError(t, err)

		rune2, err := runes.NewRuneFromString("OKLETSGOGUYSS")
		require.NoError(t, err)

		premine, ok := new(big.Int).SetString("1000000000000000000000000000000000000000000000", 10)
		require.True(t, ok)

		symbol := rune(129297)
		spacers := uint32(8226)

		tests := []struct {
			expectedTxB64 string
			params        txbuilder.BaseRuneEtchTxParams
		}{
			{
				"cHNidP8BAJ8CAAAAAUZXKFP369ZOSUKg4F+781Lp64ePDidu1UPsQxzWUorXAgAAAAD/////AwAAAAAAAAAAGGpdFQEFAgEDJQS+geUBBV0GgJTr3AMWASICAAAAAAAAIlEgyTbXlQM2cHAjy50YCG0+l5N+McVx/87HcNiEC44gWmSN8QwAAAAAABepFKpYjpRh5/yszRC1NNtHIt1yMSLBhwAAAAAAAQElUPgMAAAAAAAcX2JpdGNvaW5fdHJhbnNhY3Rpb25fc2NyaXB0XwEDBAEAAAABBTog9YoqmGWC/9aA5XLyQT/ups4F2ti+0AT+WiYhmDEoZ/qsAGMDb3JkAQ0DvkA5AAl0ZXN0IGRhdGFoARcg9YoqmGWC/9aA5XLyQT/ups4F2ti+0AT+WiYhmDEoZ/oAAAAA",
				txbuilder.BaseRuneEtchTxParams{
					InscriptionReveal: &txbuilder.PaymentData{
						UTXOs: []bitcoin.UTXO{
							{
								TxHash:  "d78a52d61c43ec43d56e270e8f87ebe952f3bb5fe0a042494ed6ebf753285746",
								Index:   2,
								Amount:  big.NewInt(850000), // 0.0085 BTC.
								Script:  []byte("_bitcoin_transaction_script_"),
								Address: "tb1p5wgkf2875q0ldqrspk367ulxwt485clkrc5j93cvmhsnppcz3x2srcptmt",
							},
						},
						Address: "tb1p5wgkf2875q0ldqrspk367ulxwt485clkrc5j93cvmhsnppcz3x2srcptmt",
						PubKey:  "02f58a2a986582ffd680e572f2413feea6ce05dad8bed004fe5a262198312867fa",
					},
					Inscription: &inscriptions.Inscription{
						Rune: rune_,
						Body: []byte("test data"),
					},
					Rune: &runes.Etching{
						Divisibility: toPointer(byte(5)),
						Premine:      big.NewInt(1000000000),
						Rune:         rune_,
						Spacers:      toPointer(uint32(37)),
						Symbol:       toPointer(']'),
					},
					AdditionalPayments: &txbuilder.PaymentData{
						UTXOs: []bitcoin.UTXO{
							{
								TxHash:  "d78a52d61c43ec43d56e270e8f87ebe952f3bb5fe0a042494ed6ebf753285746",
								Index:   4,
								Amount:  big.NewInt(27000), // 0.00027 BTC.
								Script:  []byte("_bitcoin_transaction_script_"),
								Address: "2N8mvwwUPfXt8FczXvE1UvM8ioVTW9LQLj1",
							},
						},
						Address: "2N8mvwwUPfXt8FczXvE1UvM8ioVTW9LQLj1",
						PubKey:  "03d17661b814dfaf3f7d6e70e8d4c8f5e6fdbe780a2c0373dd06ca7d75dc19f8be",
					},
					SatoshiPerKVByte:      big.NewInt(5000), // 5 sat/vB.
					RunesRecipientAddress: "tb1peymd09grxec8qg7tn5vqsmf7j7fhuvw9w8lua3msmzzqhr3qtfjqlj50zg",
					SatoshiChangeAddress:  "2N8mvwwUPfXt8FczXvE1UvM8ioVTW9LQLj1",
				},
			},
			{
				"cHNidP8BAOwCAAAAAq6V20f0qai87sqrY5zA3ubZpjgPM5n+b7J3ozxfRL2EAAAAAAD/////XHgKXBsP1r/EbXOKQpHCSEKyk/5DMVZVn7lFZAEHeVUBAAAAAP////8DAAAAAAAAAAAxal0uASYCAQOiQATcqYXt3+DCuRQFkfIHBoCAgICAgKiRi8Ciu6+cz9yGwb+7zQUWASICAAAAAAAAIlEg5aLj+ttIbun6sth40Iz+ok3PsqGS4Be9+bwYk6BACxAYEAAAAAAAACJRIOWi4/rbSG7p+rLYeNCM/qJNz7KhkuAXvfm8GJOgQAsQAAAAAAERAQEAAQE5CBwAAAAAAAAwVVNBSHh3ZTlPdUsxdFRpcXR4SkxkVWd4eklPUUI5a2xOd0pObXA4NWlwVUtaZz09AQMEAQAAAAEF/UASIBVku0l57bXXTn7tOuomXXW3PJ5idYN12RjneMPtPrwPrABjA29yZAENCNxUof0FC3MUAE0IAmlWQk9SdzBLR2dvQUFBQU5TVWhFVWdBQUFBc0FBQUFLQ0FZQUFBQmk4S1NEQUFBS3NHbERRMUJKUTBNZ1VISnZabWxzWlFBQVNJbVZsd2RVazlrU2dPLy9wNGVFbG9CMFFtK0NkQUpJQ1QzVTBJdW9oQ1NRVUVJTUJCVXJ5T0lLcmdVVkVWUVdaRlZFd2JVQXN0aFF4TUlpWUFFVlhaQkZRVjBYQzZLaThuN2dFSGIzbmZmZWVaTXpaNzUvL3JsejU5NXo3MzhtQUpDcGJKRW9EWllISUYyWUpRN3o5YURGeE1iUmNDTUFDekFBQnFwQWs4M0pGREZZckVDQXlKejl1M3k0RDZCcGU4ZDhPdGUvdi8rdm9zRGxaWElBZ0ZnSUozSXpPZWtJbjBGMGpDTVNad0dBcWtiOGVpdXpSTk44SFdHcUdDa1E0ZjVwVHA3bHNXbE9uR0UwZWlZbUlzd1RZUlVBOENRMlc1d01BRWtmOGRPeU9jbElIcElYd3BaQ3JrQ0lNUElNWE5QVE03Z0lJL01DWXlSR2hQQjBmbnJpWC9Jay95MW5valFubTUwczVkbTF6QWplUzVBcFNtT3YvaiszNDM5TGVwcGtiZzVEUkVsOHNWOFlZcEc2b0w3VWpBQXBDeE9EUStaWXdKMkpuMkcreEM5eWpqbVpubkZ6ekdWN0JVakhNCAJwZ1VIem5HU3dJY3B6WlBGakpoalhxWjMrQnlMTThLa2N5V0pQUmx6ekJiUHp5dEpqWlQ2K1R5bU5IOE9QeUo2anJNRlVjRnpuSmthSGpBZjR5bjFpeVZoMHZwNVFsK1ArWGw5cEd0UHovekxlZ1ZNNmRnc2ZvU2ZkTzNzK2ZwNVFzWjh6c3dZYVcxY25wZjNmRXlrTkY2VTVTR2RTNVRHa3NiejBueWwvc3pzY09uWUxPUkF6bzlsU2Zjd2hlM1BtbVBnQmJ4QklQS2pBUmF3QnJhSVdnTS80SjNGV3pWOVJvRm5obWkxV0pETXo2SXhrRnZHb3pHRkhJdUZOR3RMYTFzQXB1L3M3SkY0MXpkekZ5RmwvTHhQdUJ3QXUrbTlYRC92NDB3QWNFNGRBTVVYOHo3OVhPUTZsZ0Z3c1kwakVXZlArcWF2RS9JbElBSTVRRVcrQmxwQUR4Z0RjNlF5ZStBTTNKR0svVUVJaUFDeFlCbmdBRDVJQjJLd0Vxd0Z1YUFBRklFZFlBOG9BeFhnRURnS1RvQlRvQkcwZ012Z0dyZ0Z1c0E5OEFnTWdHSHdFb3lCRDJBU2dpQWNSSVlva0Nxa0RSbEFacEExUklkY0lXOG9FQXFEWXFFRUtCa1NRaEpvTGJRSktvS0tvVEtvRXFxQmZvYk9RWmVoRzFBMzlBQWFoRWFoTQgCdDlCbkdBV1RZQ3FzQ1J2Q2kyQTZ6SUFENEFoNEtad01yNEJ6NEh4NEcxd0tWOEhINFFiNE1ud0x2Z2NQd0MvaGNSUkF5YUNVVVRvb2N4UWQ1WWtLUWNXaGtsQmkxSHBVSWFvRVZZV3FReldqMmxGM1VBT29WNmhQYUN5YWdxYWh6ZEhPYUQ5MEpKcURYb0Zlajk2S0xrTWZSVGVncjZMdm9BZlJZK2h2R0RKR0EyT0djY0l3TVRHWVpNeEtUQUdtQkhNWWN4YlRocm1IR2NaOHdHS3h5bGdqckFQV0R4dUxUY0d1d1c3RkhzRFdZeTlodTdGRDJIRWNEcWVLTThPNTRFSndiRndXcmdDM0QzY2NkeEhYZ3h2R2ZjVEw0TFh4MW5nZmZCeGVpTS9EbCtDUDRTL2dlL0RQOFpNRWVZSUJ3WWtRUXVBU1ZoTzJFNm9KellUYmhHSENKRkdCYUVSMElVWVFVNGk1eEZKaUhiR04yRTk4SnlNam95dmpLQk1xSTVEWktGTXFjMUxtdXN5Z3pDZVNJc21VNUVtS0owbEkyMGhIU0pkSUQwanZ5R1N5SWRtZEhFZk9JbThqMTVDdmtKK1FQOHBTWkMxa21iSmMyUTJ5NWJJTnNqMnlyK1VJY2daeURMbGxjamx5SlhLbjVXN0x2WklueUJ2S2U4cXo1ZGZMbDh1Zk0IAmsrK1ZIMWVnS0ZncGhDaWtLMnhWT0tad1EyRkVFYWRvcU9pdHlGWE1WenlrZUVWeGlJS2k2RkU4S1J6S0prbzFwWTB5VE1WU2phaE1hZ3ExaUhxQzJra2RVMUpVc2xXS1VscWxWSzUwWG1sQUdhVnNxTXhVVGxQZXJueEsrYjd5NXdXYUN4Z0xlQXUyTEtoYjBMTmdRa1ZkeFYyRnAxS29VcTl5VCtXektrM1ZXelZWZGFkcW8rcGpOYlNhcVZxbzJrcTFnMnB0YXEvVXFlck82aHoxUXZWVDZnODFZQTFUalRDTk5ScUhORG8weGpXMU5IMDFSWnI3Tks5b3Z0SlMxbkxYU3RIYXJYVkJhMVNib3UycUxkRGVyWDFSK3dWTmljYWdwZEZLYVZkcFl6b2FPbjQ2RXAxS25VNmRTVjBqM1VqZFBOMTYzY2Q2UkQyNlhwTGVicjFXdlRGOWJmMGcvYlg2dGZvUERRZ0dkQU8rd1Y2RGRvTUpReVBEYU1QTmhvMkdJMFlxUmt5akhLTmFvMzVqc3JHYjhRcmpLdU83SmxnVHVrbXF5UUdUTGxQWTFNNlViMXB1ZXRzTU5yTTNFNWdkTU90ZWlGbm91RkM0c0dwaHJ6bkpuR0dlYlY1clBtaWhiQkZva1dmUmFQRjZrZjZpdUVVN0Y3VXYrbVpwWjVsbVdXMzVNCAJ5RXJSeXQ4cXo2clo2cTIxcVRYSHV0ejZyZzNaeHNkbWcwMlR6UnRiTTF1ZTdVSGJQanVLWFpEZFpydFd1Ni8yRHZaaSt6cjdVUWQ5aHdTSC9RNjlkQ3FkUmQ5S3YrNkljZlJ3M09EWTR2akp5ZDRweSttVTA1L081czZwenNlY1J4WWJMZVl0cmw0ODVLTHJ3bmFwZEJsd3Bia211UDdvT3VDbTQ4WjJxM0o3NnE3bnpuVS83UDZjWWNKSVlSeG52UGF3OUJCN25QV1k4SFR5WE9kNXlRdmw1ZXRWNk5YcHJlZ2Q2VjNtL2NSSDF5ZlpwOVpuek5mT2Q0M3ZKVCtNWDREZlRyOWVwaWFUdzZ4aGp2azcrSy96dnhwQUNnZ1BLQXQ0R21nYUtBNXNEb0tEL0lOMkJmVUhHd1FMZ3h0RFFBZ3paRmZJWTVZUmF3WHJsMUJzS0N1MFBQUlptRlhZMnJEMmNFcjQ4dkJqNFI4aVBDSzJSenlLTkk2VVJMWkd5VVhGUjlWRVRVUjdSUmRIRDhRc2lsa1hjeXRXTFZZUTJ4U0hpNHVLT3h3M3ZzUjd5WjRsdy9GMjhRWHg5NWNhTFYyMTlNWXl0V1ZweTg0dmwxdk9YbjQ2QVpNUW5YQXM0UXM3aEYzRkhrOWtKdTVQSE9ONGN2WnlYbkxkdWJ1NW96d1hYakh2TQgCZVpKTFVuSFNTTEpMOHE3a1ViNGJ2NFQvU3VBcEtCTzhTZkZMcVVpWlNBMUpQWkk2bFJhZFZwK09UMDlJUHlkVUZLWUtyMlpvWmF6SzZCYVppUXBFQXl1Y1Z1eFpNU1lPRUIvT2hES1haalpsVVpIbXFFTmlMUGxPTXBqdG1sMmUvWEZsMU1yVHF4UldDVmQxckRaZHZXWDE4eHlmbkovV29OZHcxclN1MVZtYnUzWndIV05kNVhwb2ZlTDYxZzE2Ry9JM0RHLzAzWGcwbDVpYm12dHJubVZlY2Q3N1RkR2Jtdk0xOHpmbUQzM24rMTF0Z1d5QnVLQjNzL1BtaXUvUjN3dSs3OXhpczJYZmxtK0YzTUtiUlpaRkpVVmZ0bksyM3Z6QjZvZlNINmEySlczcjNHNi8vZUFPN0E3aGp2czczWFllTFZZb3ppa2UyaFcwcTJFM2JYZmg3dmQ3bHUrNVVXSmJVckdYdUZleWQ2QTBzTFJwbi82K0hmdStsUEhMN3BWN2xOZnYxOWkvWmYvRUFlNkJub1B1QitzcU5DdUtLajcvS1BpeHI5SzNzcUhLc0tya0VQWlE5cUZuMVZIVjdUL1JmNm81ckhhNDZQRFhJOElqQTBmRGpsNnRjYWlwT2FaeGJIc3RYQ3VwSFQwZWY3enJoTmVKcGpyenVzcDY1ZnFpaytDa00IAjVPU0xueE4rdm44cTRGVHJhZnJwdWpNR1ovYWZwWnd0YklBYVZqZU1OZkliQjVwaW03clArWjlyYlhadVB2dUx4UzlIV25SYXlzOHJuZDkrZ1hnaC84TFV4WnlMNDVkRWwxNWRUcjQ4MUxxODlkR1ZtQ3QzcjRaZTdXd0xhTHQremVmYWxYWkcrOFhyTHRkYmJqamRPSGVUZnJQeGx2MnRoZzY3anJPLzJ2MTZ0dE8rcytHMncrMm1Mc2V1NXU3RjNSZDYzSG91My9HNmMrMHU4KzZ0ZThIM3V1OUgzdS9yamU4ZDZPUDJqVHhJZS9EbVlmYkR5VWNiK3pIOWhZL2xINWM4MFhoUzladkpiL1VEOWdQbkI3MEdPNTZHUDMwMHhCbDYrWHZtNzErRzg1K1JuNVU4MTM1ZU0ySTkwakxxTTlyMVlzbUw0WmVpbDVPdkN2NVErR1AvYStQWFovNTAvN05qTEdacytJMzR6ZFRicmU5VTN4MTViL3UrZFp3MS91UkQrb2ZKaWNLUHFoK1BmcUovYXY4Yy9mbjU1TW92dUMrbFgwMitObjhMK05ZL2xUNDFKV0tMMlRPdEFBcFJPQ2tKZ0xkSEFDREhBa0RwQW9DNFpMYW5uaEZvOW4vQURJSC94TE45OTR6WUExRHJEa0E0b2lHSUh0Z0lnQUhpbGtjc0MzbU9NCAJjQWV3alkxVTUvcmZtVjU5V3VTUEExQjV6ZHJCeCtOeFN3VU4vRU5tKy9pLzFQMVBDNlJaLzJiL0JWcUxCakg1elRYQ0FBQUFWbVZZU1daTlRRQXFBQUFBQ0FBQmgya0FCQUFBQUFFQUFBQWFBQUFBQUFBRGtvWUFCd0FBQUJJQUFBQkVvQUlBQkFBQUFBRUFBQUFMb0FNQUJBQUFBQUVBQUFBS0FBQUFBRUZUUTBsSkFBQUFVMk55WldWdWMyaHZkTlU0blRVQUFBSFVhVlJZZEZoTlREcGpiMjB1WVdSdlltVXVlRzF3QUFBQUFBQThlRHA0YlhCdFpYUmhJSGh0Ykc1ek9uZzlJbUZrYjJKbE9tNXpPbTFsZEdFdklpQjRPbmh0Y0hSclBTSllUVkFnUTI5eVpTQTJMakF1TUNJK0NpQWdJRHh5WkdZNlVrUkdJSGh0Ykc1ek9uSmtaajBpYUhSMGNEb3ZMM2QzZHk1M015NXZjbWN2TVRrNU9TOHdNaTh5TWkxeVpHWXRjM2x1ZEdGNExXNXpJeUkrQ2lBZ0lDQWdJRHh5WkdZNlJHVnpZM0pwY0hScGIyNGdjbVJtT21GaWIzVjBQU0lpQ2lBZ0lDQWdJQ0FnSUNBZ0lIaHRiRzV6T21WNGFXWTlJbWgwZEhBNkx5OXVjeTVoWkc5aVpTNWpiMjB2TbABWlhocFppOHhMakF2SWo0S0lDQWdJQ0FnSUNBZ1BHVjRhV1k2VUdsNFpXeFpSR2x0Wlc1emFXOXVQakV3UEM5bGVHbG1PbEJwZUdWc1dVUnBiV1Z1YzJsdmJqNEtJQ0FnSUNBZ0lDQWdQR1Y0YVdZNlVHbDRaV3hZUkdsdFpXNXphVzl1UGpFeFBDOWxlR2xtT2xCcGVHVnNXRVJwYldWdWMybHZiajRLSUNBZ0lDQWdJQ0FnUEdWNGFXWTZWWE5sY2tOdmJXMWxiblErVTJOeVpXVnVjMmh2ZER3dlpYaHBaanBWYzJWeVEyOXRiV1Z1ZEQ0S0lDQWdJQ0FnUEM5eVpHWTZSR1Z6WTNKcGNIUnBiMjQrQ2lBZ0lEd3ZjbVJtT2xKRVJqNEtQQzk0T25odGNHMWxkR0UrQ2xUajBvY0FBQUE5U1VSQlZCZ1pZMlJpWmYzUFFDUmdJbElkV05sZ1ZBenpLVG9OOHhjTFRBSW1BT09qMHlCNUZrYVlLaUpvSkpOQlp1SFhpbVF5SXdPNmNuUStBS1FKRENLSGM4cmpBQUFBQUVsRlRrU3VRbUNDaAEXIBVku0l57bXXTn7tOuomXXW3PJ5idYN12RjneMPtPrwPAAEBOUAbAAAAAAAAMFVTRGxvdVA2MjBodTZmcXkySGpRalA2aVRjK3lvWkxnRjczNXZCaVRvRUFMRUE9PQEDBAEAAAABFyAVZLtJee21105+7TrqJl11tzyeYnWDddkY53jD7T68DwAAAAA=",
				txbuilder.BaseRuneEtchTxParams{
					InscriptionReveal: &txbuilder.PaymentData{
						UTXOs: []bitcoin.UTXO{
							{
								TxHash:  "84bd445f3ca377b26ffe99330f38a6d9e6dec09c63abcaeebca8a9f447db95ae",
								Index:   0,
								Amount:  big.NewInt(7176), // 0.00007176 BTC.
								Script:  []byte("USAHxwe9OuK1tTiqtxJLdUgxzIOQB9klNwJNmp85ipUKZg=="),
								Address: "tb1pqlrs00f6u26m2w92kufyka2gx8xg8yq8myjnwqjdn20nnz54pfnq6jx4ad",
							},
						},
						Address: "tb1pqlrs00f6u26m2w92kufyka2gx8xg8yq8myjnwqjdn20nnz54pfnq6jx4ad",
						PubKey:  "021564bb4979edb5d74e7eed3aea265d75b73c9e62758375d918e778c3ed3ebc0f",
					},
					Inscription: &inscriptions.Inscription{
						Rune: rune2,
						Body: []byte("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"),
					},
					Rune: &runes.Etching{
						Divisibility: toPointer(byte(38)),
						Premine:      premine,
						Rune:         rune2,
						Spacers:      &spacers,
						Symbol:       &symbol,
					},
					AdditionalPayments: &txbuilder.PaymentData{
						UTXOs: []bitcoin.UTXO{
							{
								TxHash:  "d498489a8ac7832c545ca983f3b62c98d0796e3a90931fe1c0a5775692d15182",
								Index:   0,
								Amount:  big.NewInt(500000000),
								Script:  []byte("USDlouP620hu6fqy2HjQjP6iTc+yoZLgF735vBiToEALEA=="),
								Address: "tb1puk3w87kmfphwn74jmpudpr875fxulv4pjtsp000ehsvf8gzqpvgqvmvx99",
							},
							{
								TxHash:  "6e890e44f9f79e556d0eb7c52eefa3f286f5cbfa981b80b2a7c18f8165b33ea1",
								Index:   1,
								Amount:  big.NewInt(99886770),
								Script:  []byte("USDlouP620hu6fqy2HjQjP6iTc+yoZLgF735vBiToEALEA=="),
								Address: "tb1puk3w87kmfphwn74jmpudpr875fxulv4pjtsp000ehsvf8gzqpvgqvmvx99",
							},
							{
								TxHash:  "5cd6a0a0ba4e14c2e97b8c48f1929f6c7ec902640ebfaa3e4586e8bd30675fa6",
								Index:   1,
								Amount:  big.NewInt(99333840),
								Script:  []byte("USDlouP620hu6fqy2HjQjP6iTc+yoZLgF735vBiToEALEA=="),
								Address: "tb1puk3w87kmfphwn74jmpudpr875fxulv4pjtsp000ehsvf8gzqpvgqvmvx99",
							},
							{
								TxHash:  "2c5ef2311c95a140442cfa29b8fdfd1221d61213b38758091a62d0cd8615a9d3",
								Index:   6,
								Amount:  big.NewInt(972446),
								Script:  []byte("USDlouP620hu6fqy2HjQjP6iTc+yoZLgF735vBiToEALEA=="),
								Address: "tb1puk3w87kmfphwn74jmpudpr875fxulv4pjtsp000ehsvf8gzqpvgqvmvx99",
							},
							{
								TxHash:  "9e1d53d91e13d6174cdaaa55edc0c2f83b6d74ad72c2846178f308893b4c21c5",
								Index:   7,
								Amount:  big.NewInt(183848),
								Script:  []byte("USDlouP620hu6fqy2HjQjP6iTc+yoZLgF735vBiToEALEA=="),
								Address: "tb1puk3w87kmfphwn74jmpudpr875fxulv4pjtsp000ehsvf8gzqpvgqvmvx99",
							},
							{
								TxHash:  "84bd445f3ca377b26ffe99330f38a6d9e6dec09c63abcaeebca8a9f447db95ae",
								Index:   1,
								Amount:  big.NewInt(7072),
								Script:  []byte("USDlouP620hu6fqy2HjQjP6iTc+yoZLgF735vBiToEALEA=="),
								Address: "tb1puk3w87kmfphwn74jmpudpr875fxulv4pjtsp000ehsvf8gzqpvgqvmvx99",
							},
							{
								TxHash:  "557907016445b99f55563143fe93b24248c291428a736dc4bfd60f1b5c0a785c",
								Index:   1,
								Amount:  big.NewInt(6976),
								Script:  []byte("USDlouP620hu6fqy2HjQjP6iTc+yoZLgF735vBiToEALEA=="),
								Address: "tb1puk3w87kmfphwn74jmpudpr875fxulv4pjtsp000ehsvf8gzqpvgqvmvx99",
							},
							{
								TxHash:  "89fc9847c6f3c466e28dbe40f7ce963b7201227f5359058a090ece1e85c5837e",
								Index:   4,
								Amount:  big.NewInt(2500),
								Script:  []byte("USDlouP620hu6fqy2HjQjP6iTc+yoZLgF735vBiToEALEA=="),
								Address: "tb1puk3w87kmfphwn74jmpudpr875fxulv4pjtsp000ehsvf8gzqpvgqvmvx99",
							},
							{
								TxHash:  "90718612bfcfdb06f1583f87e65079a1e3d4fdeced46aa847444b40560313d78",
								Index:   4,
								Amount:  big.NewInt(2500),
								Script:  []byte("USDlouP620hu6fqy2HjQjP6iTc+yoZLgF735vBiToEALEA=="),
								Address: "tb1puk3w87kmfphwn74jmpudpr875fxulv4pjtsp000ehsvf8gzqpvgqvmvx99",
							},
							{
								TxHash:  "6ff6aae5a2ed339810946eca6eeb30739ea5028d2250732e576cdb0addf82527",
								Index:   4,
								Amount:  big.NewInt(2500),
								Script:  []byte("USDlouP620hu6fqy2HjQjP6iTc+yoZLgF735vBiToEALEA=="),
								Address: "tb1puk3w87kmfphwn74jmpudpr875fxulv4pjtsp000ehsvf8gzqpvgqvmvx99",
							},
							{
								TxHash:  "2a52e2efb33a0377906545a030fe778c333d6922cf1085ea9a7ca4dc6bc5c03c",
								Index:   4,
								Amount:  big.NewInt(2500),
								Script:  []byte("USDlouP620hu6fqy2HjQjP6iTc+yoZLgF735vBiToEALEA=="),
								Address: "tb1puk3w87kmfphwn74jmpudpr875fxulv4pjtsp000ehsvf8gzqpvgqvmvx99",
							},
							{
								TxHash:  "82736c9620e828a3871ad92ea769bd43aeac7d62ac0a8aade7992a5c6adf677f",
								Index:   6,
								Amount:  big.NewInt(600),
								Script:  []byte("USDlouP620hu6fqy2HjQjP6iTc+yoZLgF735vBiToEALEA=="),
								Address: "tb1puk3w87kmfphwn74jmpudpr875fxulv4pjtsp000ehsvf8gzqpvgqvmvx99",
							},
							{
								TxHash:  "9e1d53d91e13d6174cdaaa55edc0c2f83b6d74ad72c2846178f308893b4c21c5",
								Index:   6,
								Amount:  big.NewInt(600),
								Script:  []byte("USDlouP620hu6fqy2HjQjP6iTc+yoZLgF735vBiToEALEA=="),
								Address: "tb1puk3w87kmfphwn74jmpudpr875fxulv4pjtsp000ehsvf8gzqpvgqvmvx99",
							},
						},
						Address: "tb1puk3w87kmfphwn74jmpudpr875fxulv4pjtsp000ehsvf8gzqpvgqvmvx99",
						PubKey:  "021564bb4979edb5d74e7eed3aea265d75b73c9e62758375d918e778c3ed3ebc0f",
					},
					SatoshiPerKVByte:      big.NewInt(6000), // 6 sat/vB.
					RunesRecipientAddress: "tb1puk3w87kmfphwn74jmpudpr875fxulv4pjtsp000ehsvf8gzqpvgqvmvx99",
					SatoshiChangeAddress:  "tb1puk3w87kmfphwn74jmpudpr875fxulv4pjtsp000ehsvf8gzqpvgqvmvx99",
				},
			},
		}
		for i, test := range tests {
			t.Run(fmt.Sprintf("test %d", i), func(t *testing.T) {
				result, err := txBuilder.BuildRuneEtchTx(test.params)
				require.NoError(t, err)
				require.EqualValues(t, test.expectedTxB64, base64.StdEncoding.EncodeToString(result.SerializedPSBT))
			})
		}
	})

	t.Run("BuildRuneEtchTx with primine splitting factor", func(t *testing.T) {
		rune_, err := runes.NewRuneFromString("HELLO")
		require.NoError(t, err)

		tests := []struct {
			name            string
			expectedTxB64   string
			expectedOutputs int
			edictsSize      int
			pointer         *uint32
			changePresent   bool
			params          txbuilder.BaseRuneEtchTxParams
		}{
			{
				name:            "psf - 0, no change",
				expectedTxB64:   "cHNidP8BAH8CAAAAAUZXKFP369ZOSUKg4F+781Lp64ePDidu1UPsQxzWUorXAgAAAAD/////AgAAAAAAAAAAGGpdFQEFAgEDJQS+geUBBV0GgJTr3AMWASICAAAAAAAAIlEgyTbXlQM2cHAjy50YCG0+l5N+McVx/87HcNiEC44gWmQAAAAAAAEBJcMGAAAAAAAAHF9iaXRjb2luX3RyYW5zYWN0aW9uX3NjcmlwdF8BAwQBAAAAAQU6IPWKKphlgv/WgOVy8kE/7qbOBdrYvtAE/lomIZgxKGf6rABjA29yZAENA75AOQAJdGVzdCBkYXRhaAEXIPWKKphlgv/WgOVy8kE/7qbOBdrYvtAE/lomIZgxKGf6AAAA",
				expectedOutputs: 2,
				edictsSize:      0,
				pointer:         toPointer[uint32](1),
				changePresent:   false,
				params: txbuilder.BaseRuneEtchTxParams{
					InscriptionReveal: &txbuilder.PaymentData{
						UTXOs: []bitcoin.UTXO{
							{
								TxHash:  "d78a52d61c43ec43d56e270e8f87ebe952f3bb5fe0a042494ed6ebf753285746",
								Index:   2,
								Amount:  big.NewInt(1731), // no change.
								Script:  []byte("_bitcoin_transaction_script_"),
								Address: "tb1p5wgkf2875q0ldqrspk367ulxwt485clkrc5j93cvmhsnppcz3x2srcptmt",
							},
						},
						Address: "tb1p5wgkf2875q0ldqrspk367ulxwt485clkrc5j93cvmhsnppcz3x2srcptmt",
						PubKey:  "02f58a2a986582ffd680e572f2413feea6ce05dad8bed004fe5a262198312867fa",
					},
					Inscription: &inscriptions.Inscription{
						Rune: rune_,
						Body: []byte("test data"),
					},
					Rune: &runes.Etching{
						Divisibility: toPointer(byte(5)),
						Premine:      big.NewInt(1000000000),
						Rune:         rune_,
						Spacers:      toPointer(uint32(37)),
						Symbol:       toPointer(']'),
					},
					SatoshiPerKVByte:       big.NewInt(5000), // 5 sat/vB.
					RunesRecipientAddress:  "tb1peymd09grxec8qg7tn5vqsmf7j7fhuvw9w8lua3msmzzqhr3qtfjqlj50zg",
					SatoshiChangeAddress:   "2N8mvwwUPfXt8FczXvE1UvM8ioVTW9LQLj1",
					PremineSplittingFactor: 0,
				},
			},
			{
				name:            "psf - 0 + change",
				expectedTxB64:   "cHNidP8BAJ8CAAAAAUZXKFP369ZOSUKg4F+781Lp64ePDidu1UPsQxzWUorXAgAAAAD/////AwAAAAAAAAAAGGpdFQEFAgEDJQS+geUBBV0GgJTr3AMWASICAAAAAAAAIlEgyTbXlQM2cHAjy50YCG0+l5N+McVx/87HcNiEC44gWmQjAgAAAAAAABepFKpYjpRh5/yszRC1NNtHIt1yMSLBhwAAAAAAAQEl5ggAAAAAAAAcX2JpdGNvaW5fdHJhbnNhY3Rpb25fc2NyaXB0XwEDBAEAAAABBTog9YoqmGWC/9aA5XLyQT/ups4F2ti+0AT+WiYhmDEoZ/qsAGMDb3JkAQ0DvkA5AAl0ZXN0IGRhdGFoARcg9YoqmGWC/9aA5XLyQT/ups4F2ti+0AT+WiYhmDEoZ/oAAAAA",
				expectedOutputs: 3,
				edictsSize:      0,
				pointer:         toPointer[uint32](1),
				changePresent:   true,
				params: txbuilder.BaseRuneEtchTxParams{
					InscriptionReveal: &txbuilder.PaymentData{
						UTXOs: []bitcoin.UTXO{
							{
								TxHash:  "d78a52d61c43ec43d56e270e8f87ebe952f3bb5fe0a042494ed6ebf753285746",
								Index:   2,
								Amount:  big.NewInt(2278), // 546 change.
								Script:  []byte("_bitcoin_transaction_script_"),
								Address: "tb1p5wgkf2875q0ldqrspk367ulxwt485clkrc5j93cvmhsnppcz3x2srcptmt",
							},
						},
						Address: "tb1p5wgkf2875q0ldqrspk367ulxwt485clkrc5j93cvmhsnppcz3x2srcptmt",
						PubKey:  "02f58a2a986582ffd680e572f2413feea6ce05dad8bed004fe5a262198312867fa",
					},
					Inscription: &inscriptions.Inscription{
						Rune: rune_,
						Body: []byte("test data"),
					},
					Rune: &runes.Etching{
						Divisibility: toPointer(byte(5)),
						Premine:      big.NewInt(1000000000),
						Rune:         rune_,
						Spacers:      toPointer(uint32(37)),
						Symbol:       toPointer(']'),
					},
					SatoshiPerKVByte:       big.NewInt(5000), // 5 sat/vB.
					RunesRecipientAddress:  "tb1peymd09grxec8qg7tn5vqsmf7j7fhuvw9w8lua3msmzzqhr3qtfjqlj50zg",
					SatoshiChangeAddress:   "2N8mvwwUPfXt8FczXvE1UvM8ioVTW9LQLj1",
					PremineSplittingFactor: 0,
				},
			},
			{
				name:            "psf - 1, no change",
				expectedTxB64:   "cHNidP8BAH8CAAAAAUZXKFP369ZOSUKg4F+781Lp64ePDidu1UPsQxzWUorXAgAAAAD/////AgAAAAAAAAAAGGpdFQEFAgEDJQS+geUBBV0GgJTr3AMWASICAAAAAAAAIlEgyTbXlQM2cHAjy50YCG0+l5N+McVx/87HcNiEC44gWmQAAAAAAAEBJcMGAAAAAAAAHF9iaXRjb2luX3RyYW5zYWN0aW9uX3NjcmlwdF8BAwQBAAAAAQU6IPWKKphlgv/WgOVy8kE/7qbOBdrYvtAE/lomIZgxKGf6rABjA29yZAENA75AOQAJdGVzdCBkYXRhaAEXIPWKKphlgv/WgOVy8kE/7qbOBdrYvtAE/lomIZgxKGf6AAAA",
				expectedOutputs: 2,
				edictsSize:      0,
				pointer:         toPointer[uint32](1),
				changePresent:   false,
				params: txbuilder.BaseRuneEtchTxParams{
					InscriptionReveal: &txbuilder.PaymentData{
						UTXOs: []bitcoin.UTXO{
							{
								TxHash:  "d78a52d61c43ec43d56e270e8f87ebe952f3bb5fe0a042494ed6ebf753285746",
								Index:   2,
								Amount:  big.NewInt(1731), // no change.
								Script:  []byte("_bitcoin_transaction_script_"),
								Address: "tb1p5wgkf2875q0ldqrspk367ulxwt485clkrc5j93cvmhsnppcz3x2srcptmt",
							},
						},
						Address: "tb1p5wgkf2875q0ldqrspk367ulxwt485clkrc5j93cvmhsnppcz3x2srcptmt",
						PubKey:  "02f58a2a986582ffd680e572f2413feea6ce05dad8bed004fe5a262198312867fa",
					},
					Inscription: &inscriptions.Inscription{
						Rune: rune_,
						Body: []byte("test data"),
					},
					Rune: &runes.Etching{
						Divisibility: toPointer(byte(5)),
						Premine:      big.NewInt(1000000000),
						Rune:         rune_,
						Spacers:      toPointer(uint32(37)),
						Symbol:       toPointer(']'),
					},
					SatoshiPerKVByte:       big.NewInt(5000), // 5 sat/vB.
					RunesRecipientAddress:  "tb1peymd09grxec8qg7tn5vqsmf7j7fhuvw9w8lua3msmzzqhr3qtfjqlj50zg",
					SatoshiChangeAddress:   "2N8mvwwUPfXt8FczXvE1UvM8ioVTW9LQLj1",
					PremineSplittingFactor: 1,
				},
			},
			{
				name:            "psf - 2, no change, divisible",
				expectedTxB64:   "cHNidP8BALECAAAAAUZXKFP369ZOSUKg4F+781Lp64ePDidu1UPsQxzWUorXAgAAAAD/////AwAAAAAAAAAAH2pdHAEFAgEDJQS+geUBBV0GgJTr3AMAAACAyrXuAQMiAgAAAAAAACJRIMk215UDNnBwI8udGAhtPpeTfjHFcf/Ox3DYhAuOIFpkIgIAAAAAAAAiUSDJNteVAzZwcCPLnRgIbT6Xk34xxXH/zsdw2IQLjiBaZAAAAAAAAQElewkAAAAAAAAcX2JpdGNvaW5fdHJhbnNhY3Rpb25fc2NyaXB0XwEDBAEAAAABBTog9YoqmGWC/9aA5XLyQT/ups4F2ti+0AT+WiYhmDEoZ/qsAGMDb3JkAQ0DvkA5AAl0ZXN0IGRhdGFoARcg9YoqmGWC/9aA5XLyQT/ups4F2ti+0AT+WiYhmDEoZ/oAAAAA",
				expectedOutputs: 3,
				edictsSize:      1,
				pointer:         nil,
				changePresent:   false,
				params: txbuilder.BaseRuneEtchTxParams{
					InscriptionReveal: &txbuilder.PaymentData{
						UTXOs: []bitcoin.UTXO{
							{
								TxHash:  "d78a52d61c43ec43d56e270e8f87ebe952f3bb5fe0a042494ed6ebf753285746",
								Index:   2,
								Amount:  big.NewInt(2427), // no change.
								Script:  []byte("_bitcoin_transaction_script_"),
								Address: "tb1p5wgkf2875q0ldqrspk367ulxwt485clkrc5j93cvmhsnppcz3x2srcptmt",
							},
						},
						Address: "tb1p5wgkf2875q0ldqrspk367ulxwt485clkrc5j93cvmhsnppcz3x2srcptmt",
						PubKey:  "02f58a2a986582ffd680e572f2413feea6ce05dad8bed004fe5a262198312867fa",
					},
					Inscription: &inscriptions.Inscription{
						Rune: rune_,
						Body: []byte("test data"),
					},
					Rune: &runes.Etching{
						Divisibility: toPointer(byte(5)),
						Premine:      big.NewInt(1000000000),
						Rune:         rune_,
						Spacers:      toPointer(uint32(37)),
						Symbol:       toPointer(']'),
					},
					SatoshiPerKVByte:       big.NewInt(5000), // 5 sat/vB.
					RunesRecipientAddress:  "tb1peymd09grxec8qg7tn5vqsmf7j7fhuvw9w8lua3msmzzqhr3qtfjqlj50zg",
					SatoshiChangeAddress:   "2N8mvwwUPfXt8FczXvE1UvM8ioVTW9LQLj1",
					PremineSplittingFactor: 2,
				},
			},
			{
				name:            "psf - 3, no change, not divisible",
				expectedTxB64:   "cHNidP8BAOACAAAAAUZXKFP369ZOSUKg4F+781Lp64ePDidu1UPsQxzWUorXAgAAAAD/////BAAAAAAAAAAAI2pdIAEFAgEDJQS+geUBBV0GgJTr3AMAAAABAQAA1Yb5ngEEIgIAAAAAAAAiUSDJNteVAzZwcCPLnRgIbT6Xk34xxXH/zsdw2IQLjiBaZCICAAAAAAAAIlEgyTbXlQM2cHAjy50YCG0+l5N+McVx/87HcNiEC44gWmQiAgAAAAAAACJRIMk215UDNnBwI8udGAhtPpeTfjHFcf/Ox3DYhAuOIFpkAAAAAAABASUzDAAAAAAAABxfYml0Y29pbl90cmFuc2FjdGlvbl9zY3JpcHRfAQMEAQAAAAEFOiD1iiqYZYL/1oDlcvJBP+6mzgXa2L7QBP5aJiGYMShn+qwAYwNvcmQBDQO+QDkACXRlc3QgZGF0YWgBFyD1iiqYZYL/1oDlcvJBP+6mzgXa2L7QBP5aJiGYMShn+gAAAAAA",
				expectedOutputs: 4,
				edictsSize:      2,
				pointer:         nil,
				changePresent:   false,
				params: txbuilder.BaseRuneEtchTxParams{
					InscriptionReveal: &txbuilder.PaymentData{
						UTXOs: []bitcoin.UTXO{
							{
								TxHash:  "d78a52d61c43ec43d56e270e8f87ebe952f3bb5fe0a042494ed6ebf753285746",
								Index:   2,
								Amount:  big.NewInt(3123), // no change.
								Script:  []byte("_bitcoin_transaction_script_"),
								Address: "tb1p5wgkf2875q0ldqrspk367ulxwt485clkrc5j93cvmhsnppcz3x2srcptmt",
							},
						},
						Address: "tb1p5wgkf2875q0ldqrspk367ulxwt485clkrc5j93cvmhsnppcz3x2srcptmt",
						PubKey:  "02f58a2a986582ffd680e572f2413feea6ce05dad8bed004fe5a262198312867fa",
					},
					Inscription: &inscriptions.Inscription{
						Rune: rune_,
						Body: []byte("test data"),
					},
					Rune: &runes.Etching{
						Divisibility: toPointer(byte(5)),
						Premine:      big.NewInt(1000000000),
						Rune:         rune_,
						Spacers:      toPointer(uint32(37)),
						Symbol:       toPointer(']'),
					},
					SatoshiPerKVByte:       big.NewInt(5000), // 5 sat/vB.
					RunesRecipientAddress:  "tb1peymd09grxec8qg7tn5vqsmf7j7fhuvw9w8lua3msmzzqhr3qtfjqlj50zg",
					SatoshiChangeAddress:   "2N8mvwwUPfXt8FczXvE1UvM8ioVTW9LQLj1",
					PremineSplittingFactor: 3,
				},
			},
			{
				name:            "psf - 3, change, not divisible",
				expectedTxB64:   "cHNidP8BAP0AAQIAAAABRlcoU/fr1k5JQqDgX7vzUunrh48OJ27VQ+xDHNZSitcCAAAAAP////8FAAAAAAAAAAAjal0gAQUCAQMlBL6B5QEFXQaAlOvcAwAAAAEBAADVhvmeAQUiAgAAAAAAACJRIMk215UDNnBwI8udGAhtPpeTfjHFcf/Ox3DYhAuOIFpkIgIAAAAAAAAiUSDJNteVAzZwcCPLnRgIbT6Xk34xxXH/zsdw2IQLjiBaZCICAAAAAAAAIlEgyTbXlQM2cHAjy50YCG0+l5N+McVx/87HcNiEC44gWmQjAgAAAAAAABepFKpYjpRh5/yszRC1NNtHIt1yMSLBhwAAAAAAAQElVg4AAAAAAAAcX2JpdGNvaW5fdHJhbnNhY3Rpb25fc2NyaXB0XwEDBAEAAAABBTog9YoqmGWC/9aA5XLyQT/ups4F2ti+0AT+WiYhmDEoZ/qsAGMDb3JkAQ0DvkA5AAl0ZXN0IGRhdGFoARcg9YoqmGWC/9aA5XLyQT/ups4F2ti+0AT+WiYhmDEoZ/oAAAAAAAA=",
				expectedOutputs: 5,
				edictsSize:      2,
				pointer:         nil,
				changePresent:   false,
				params: txbuilder.BaseRuneEtchTxParams{
					InscriptionReveal: &txbuilder.PaymentData{
						UTXOs: []bitcoin.UTXO{
							{
								TxHash:  "d78a52d61c43ec43d56e270e8f87ebe952f3bb5fe0a042494ed6ebf753285746",
								Index:   2,
								Amount:  big.NewInt(3670), // change 546.
								Script:  []byte("_bitcoin_transaction_script_"),
								Address: "tb1p5wgkf2875q0ldqrspk367ulxwt485clkrc5j93cvmhsnppcz3x2srcptmt",
							},
						},
						Address: "tb1p5wgkf2875q0ldqrspk367ulxwt485clkrc5j93cvmhsnppcz3x2srcptmt",
						PubKey:  "02f58a2a986582ffd680e572f2413feea6ce05dad8bed004fe5a262198312867fa",
					},
					Inscription: &inscriptions.Inscription{
						Rune: rune_,
						Body: []byte("test data"),
					},
					Rune: &runes.Etching{
						Divisibility: toPointer(byte(5)),
						Premine:      big.NewInt(1000000000),
						Rune:         rune_,
						Spacers:      toPointer(uint32(37)),
						Symbol:       toPointer(']'),
					},
					SatoshiPerKVByte:       big.NewInt(5000), // 5 sat/vB.
					RunesRecipientAddress:  "tb1peymd09grxec8qg7tn5vqsmf7j7fhuvw9w8lua3msmzzqhr3qtfjqlj50zg",
					SatoshiChangeAddress:   "2N8mvwwUPfXt8FczXvE1UvM8ioVTW9LQLj1",
					PremineSplittingFactor: 3,
				},
			},
		}
		for _, test := range tests {
			t.Run(test.name, func(t *testing.T) {
				result, err := txBuilder.BuildRuneEtchTx(test.params)
				require.NoError(t, err)
				require.EqualValues(t, test.expectedTxB64, base64.StdEncoding.EncodeToString(result.SerializedPSBT))

				p, err := psbt.NewFromRawBytes(bytes.NewReader(result.SerializedPSBT), false)
				require.NoError(t, err)

				tx := p.UnsignedTx
				require.Len(t, tx.TxOut, test.expectedOutputs)

				runestore, err := runes.ParseRunestone(tx.TxOut[0].PkScript)
				require.NoError(t, err)

				require.Len(t, runestore.Edicts, test.edictsSize)
				require.Equal(t, test.pointer, runestore.Pointer)

				if test.edictsSize == 1 {
					require.EqualValues(t, test.expectedOutputs, runestore.Edicts[0].Output)
					require.True(t, numbers.IsEqual(test.params.Rune.Premine,
						new(big.Int).Mul(runestore.Edicts[0].Amount, big.NewInt(int64(test.params.PremineSplittingFactor)))))
				}
				if test.edictsSize == 2 {
					require.EqualValues(t, 1, runestore.Edicts[0].Output)
					require.EqualValues(t, test.expectedOutputs, runestore.Edicts[1].Output)

					sum := new(big.Int).Mul(runestore.Edicts[1].Amount, big.NewInt(int64(test.params.PremineSplittingFactor)))
					sum.Add(sum, runestore.Edicts[0].Amount)
					require.True(t, numbers.IsEqual(test.params.Rune.Premine, sum))
				}
			})
		}
	})
}

func toPointer[T any](val T) *T {
	return &val
}
