// Copyright (C) 2024 Creditor Corp. Group.
// See LICENSE for copying information.

package txbuilder_test

import (
	"encoding/base64"
	"errors"
	"math/big"
	"math/rand"
	"testing"

	"github.com/btcsuite/btcd/chaincfg"
	"github.com/stretchr/testify/require"

	"github.com/BoostyLabs/blockchain/bitcoin"
	"github.com/BoostyLabs/blockchain/bitcoin/ord/inscriptions"
	"github.com/BoostyLabs/blockchain/bitcoin/ord/runes"
	"github.com/BoostyLabs/blockchain/bitcoin/txbuilder"
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
		expectedTxB64 := "cHNidP8BAPICAAAAAkZXKFP369ZOSUKg4F+781Lp64ePDidu1UPsQxzWUorXBAAAAAD/////RlcoU/fr1k5JQqDgX7vzUunrh48OJ27VQ+xDHNZSitcCAAAAAP////8EAAAAAAAAAAAMal0JFgIA4ghNnRoBIgIAAAAAAAAiUSAu6vu/kq8tH14IZsvr1he5lWJfN2J6Y4yQTd0mhUTDECICAAAAAAAAIlEgyTbXlQM2cHAjy50YCG0+l5N+McVx/87HcNiEC44gWmQb8AwAAAAAACJRIMk215UDNnBwI8udGAhtPpeTfjHFcf/Ox3DYhAuOIFpkAAAAAAEQAQABEQEBAAEBKiICAAAAAAAAIV9iaXRjb2luX3RyYW5zYWN0aW9uX3J1bmVfc2NyaXB0XwEDBAEAAAABFyAp+mEcNhNVsILuWT/rNoAJqpxr0e02yZg+3NET+42jPwABASVQ+AwAAAAAABxfYml0Y29pbl90cmFuc2FjdGlvbl9zY3JpcHRfAQMEAQAAAAEXINF2YbgU368/fW5w6NTI9eb9vngKLANz3QbKfXXcGfi+AAAAAAA="
		runeID := runes.RuneID{Block: 1122, TxID: 77}
		result, err := txBuilder.BuildRunesTransferTx(txbuilder.BaseRunesTransferParams{
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
		})
		require.NoError(t, err)
		require.EqualValues(t, expectedTxB64, base64.StdEncoding.EncodeToString(result.SerializedPSBT))
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
		for _, test := range tests {
			result, err := txBuilder.BuildBTCTransferTx(test.params)
			require.NoError(t, err)
			require.EqualValues(t, test.expectedTxB64, base64.StdEncoding.EncodeToString(result.SerializedPSBT))
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
		}
		for _, test := range tests {
			result, err := txBuilder.BuildInscriptionTx(test.params)
			require.ErrorIs(t, err, test.error)
			require.EqualValues(t, test.expectedTxB64, base64.StdEncoding.EncodeToString(result.SerializedPSBT))
		}
	})

	t.Run("BuildRuneEtchTx", func(t *testing.T) {
		rune_, err := runes.NewRuneFromString("HELLO")
		require.NoError(t, err)

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
		}
		for _, test := range tests {
			result, err := txBuilder.BuildRuneEtchTx(test.params)
			require.NoError(t, err)
			require.EqualValues(t, test.expectedTxB64, base64.StdEncoding.EncodeToString(result.SerializedPSBT))
		}
	})
}

func toPointer[T any](val T) *T {
	return &val
}
