// Copyright (C) 2024 Creditor Corp. Group.
// See LICENSE for copying information.

package txbuilder

import (
	"bytes"
	"encoding/hex"
	"errors"
	"math/big"

	"github.com/btcsuite/btcd/btcec/v2"
	"github.com/btcsuite/btcd/btcutil"
	"github.com/btcsuite/btcd/btcutil/psbt"
	"github.com/btcsuite/btcd/chaincfg"
	"github.com/btcsuite/btcd/chaincfg/chainhash"
	"github.com/btcsuite/btcd/txscript"
	"github.com/btcsuite/btcd/wire"

	"github.com/BoostyLabs/blockchain/bitcoin"
	"github.com/BoostyLabs/blockchain/bitcoin/ord/runes"
	"github.com/BoostyLabs/blockchain/internal/numbers"
)

const (
	// txVersion defines transaction version for this builder.
	txVersion int32 = 2
	// signHashType define signature hash type for input signing.
	signHashType = txscript.SigHashAll
)

var (
	// headerSizeVBytes defined rough tx header size in vBytes.
	headerSizeVBytes = big.NewInt(11)
	// inputSizeVBytes defined rough tx input size in vBytes.
	inputSizeVBytes = big.NewInt(90)
	// outputSizeVBytes defined rough tx output size in vBytes.
	outputSizeVBytes = big.NewInt(30)

	// nonDustBitcoinAmount defined the smallest needed amount in satoshi to link to rune output.
	nonDustBitcoinAmount = big.NewInt(546)

	// recipientOutput defines runes output for recipient (transferring) by base rune tx.
	recipientOutput uint32 = 1
	// returnOutput defines runes output for sender (change) by base rune tx.
	returnOutput uint32 = 2
)

// BaseRunesTransferParams describes basic data needed to build rune transfer transaction.
type BaseRunesTransferParams struct {
	RuneID                     runes.RuneID
	RuneUTXOs                  []bitcoin.UTXO // must be sorted by rune amount desc.
	BaseUTXOs                  []bitcoin.UTXO // must be sorted by btc amount desc.
	TransferRuneAmount         *big.Int       // runes amount to transfer.
	SatoshiPerKVByte           *big.Int       // fee rate in satoshi per kilo virtual byte.
	SatoshiCommissionAmount    *big.Int       // additional commission in satoshi to be charged from user.
	RecipientTaprootAddress    string         // recipient runes address.
	CommissionRecipientAddress string         // recipient commission address.
	SenderTaprootAddress       string         // sender runes address.
	SenderPaymentAddress       string         // sender commission/fee payment address.
	SenderTaprootPubKey        string         // sender taproot public key.
	SenderPaymentPubKey        string         // sender payment public key.
}

// BaseRunesTransferResult describes result of buildBaseTransferRuneTx method.
type BaseRunesTransferResult struct {
	UnsignedRawTx *wire.MsgTx     // unsigned rune transfer transaction.
	UsedRuneUTXOs []*bitcoin.UTXO // used rune utxos in transaction.
	UsedBaseUTXOs []*bitcoin.UTXO // used bitcoin utxos in transaction.
	EstimatedFee  *big.Int        // estimated transaction fee in Satoshi.
}

// BuildRunesTransferTxResult describes result of BuildRunesTransferTx method.
type BuildRunesTransferTxResult struct {
	SerializedPSBT []byte          // serialised unsigned rune transfer transaction in PSBT format.
	UsedRuneUTXOs  []*bitcoin.UTXO // used rune utxos in transaction.
	UsedBaseUTXOs  []*bitcoin.UTXO // used bitcoin utxos in transaction.
	EstimatedFee   *big.Int        // estimated transaction fee in Satoshi.
}

// BuildRunesTransferPSBTParams describes data needed to convert unsigned rune transfer transaction
// to partly signed bitcoin transaction (PSBT).
type BuildRunesTransferPSBTParams struct {
	BaseRunesTransferResult
	SenderTaprootPubKey string
	SenderPaymentPubKey string
}

// BaseBTCTransferParams describes basic data needed to build btc transfer transaction.
// NOTE: utxos should contain btc only, any joined runes will be lost.
type BaseBTCTransferParams struct {
	Sender                    *PaymentData // sender payment data. mandatory. if FeePayer is not provided, sender is a FeePayer.
	FeePayer                  *PaymentData // specified fee payer data, optional.
	TransferSatoshiAmount     *big.Int     // amount to transfer in satoshi.
	SatoshiPerKVByte          *big.Int     // fee rate in satoshi per kilo virtual byte.
	RecipientAddress          string       // recipient btc address.
	SatoshiCommissionAmount   *big.Int     // additional commission in satoshi to be charged from user, optional.
	CommissionReceiverAddress string       // recipient commission address, optional.
}

// PaymentData defined data needed to construct inputs.
type PaymentData struct {
	UTXOs   []bitcoin.UTXO // must be sorted by target token amount desc.
	Address string         // payer address.
	PubKey  string         // payer public key.
}

// BaseBTCTransferResult describes result of buildBaseTransferBTCTx method.
type BaseBTCTransferResult struct {
	UnsignedRawTx         *wire.MsgTx     // unsigned btc transfer transaction.
	UsedSenderBaseUTXOs   []*bitcoin.UTXO // used sender's bitcoin utxos in transaction.
	UsedFeePayerBaseUTXOs []*bitcoin.UTXO // used fee payer's bitcoin utxos in transaction.
	EstimatedFee          *big.Int        // estimated transaction fee in Satoshi.
}

// BuildBTCTransferTxResult describes result of BuildBTCTransferTx method.
type BuildBTCTransferTxResult struct {
	SerializedPSBT        []byte          // serialised unsigned rune transfer transaction in PSBT format.
	UsedSenderBaseUTXOs   []*bitcoin.UTXO // used sender's bitcoin utxos in transaction.
	UsedFeePayerBaseUTXOs []*bitcoin.UTXO // used fee payer's bitcoin utxos in transaction.
	EstimatedFee          *big.Int        // estimated transaction fee in Satoshi.
}

// BuildBTCTransferPSBTParams describes data needed to convert unsigned btc transfer transaction
// to partly signed bitcoin transaction (PSBT).
type BuildBTCTransferPSBTParams struct {
	BaseBTCTransferResult
	SenderAddress   string
	SenderPubKey    string
	FeePayerAddress string
	FeePayerPubKey  string
}

// TxBuilder provides transaction building related logic.
type TxBuilder struct {
	networkParams *chaincfg.Params
}

// NewTxBuilder is a constructor for TxBuilder.
func NewTxBuilder(networkParams *chaincfg.Params) *TxBuilder {
	return &TxBuilder{
		networkParams: networkParams,
	}
}

// BuildRunesTransferTx constructs rune transferring transaction in PSBT
// format with inputs indexes assigned in unknown fields. Returns serialized
// PSBT transaction with used rune and base outputs, estimated fee in satoshi,
// and error if any.
func (b *TxBuilder) BuildRunesTransferTx(params BaseRunesTransferParams) (result BuildRunesTransferTxResult, _ error) {
	buildBaseTransferRuneTxResult, err := b.buildBaseTransferRuneTx(params)
	if err != nil {
		return result, err
	}

	result.UsedRuneUTXOs = buildBaseTransferRuneTxResult.UsedRuneUTXOs
	result.UsedBaseUTXOs = buildBaseTransferRuneTxResult.UsedBaseUTXOs
	result.EstimatedFee = buildBaseTransferRuneTxResult.EstimatedFee

	result.SerializedPSBT, err = b.BuildRunesTransferPSBT(BuildRunesTransferPSBTParams{
		BaseRunesTransferResult: buildBaseTransferRuneTxResult,
		SenderTaprootPubKey:     params.SenderTaprootPubKey,
		SenderPaymentPubKey:     params.SenderPaymentPubKey,
	})
	if err != nil {
		return result, err
	}

	return result, nil
}

// buildBaseTransferRuneTx constructs base rune transferring transaction.
// Returns transaction, list of used rune's utxos pointers,
// list of used base utxos pointers, estimated fee, and error if any.
//
//	Tx struct
//	inputs:
//	┌─────────┬──────────────┬────────────────────────────────────────┐
//	│  index  │     type     │             description                │
//	├=========┼==============┼========================================┤
//	│   0 - k │ rune inputs  │ utxos with linked runes, possibly many │
//	├─────────┼──────────────┼────────────────────────────────────────┤
//	│ k+1 - n │ base inputs  │ utxos with bitcoin only, possibly many │
//	└─────────┴──────────────┴────────────────────────────────────────┘
//
//	outputs:
//	┌─────────┬──────────────┬────────────────────────────────────────┐
//	│  index  │     type     │             description                │
//	├=========┼==============┼========================================┤
//	│       0 │ runestone    │ rune protocol main output              │
//	├─────────┼──────────────┼────────────────────────────────────────┤
//	│       1 │ rune output  │ mandatory, output to link runes        │
//	│         │              │ to recipient.                          │
//	├─────────┼──────────────┼────────────────────────────────────────┤
//	│       2 │ rune output  │ optional, output to return runes       │
//	│         │              │ change to sender.                      │
//	├─────────┼──────────────┼────────────────────────────────────────┤
//	│       3 │ base output  │ service native commission. optional,   │
//	│         │              │ charge commission from sender if       │
//	│         │              │ satoshi commission amount is not 0.    │
//	├─────────┼──────────────┼────────────────────────────────────────┤
//	│       4 │ base output  │ outputs to change bitcoin amount.      │
//	│         │              │ 99% mandatory, if any left.            │
//	└─────────┴──────────────┴────────────────────────────────────────┘
func (b *TxBuilder) buildBaseTransferRuneTx(params BaseRunesTransferParams) (result BaseRunesTransferResult, _ error) {
	runeUTXOs, totalRuneAmount, err := PrepareRuneUTXOs(params.RuneUTXOs, params.TransferRuneAmount, params.RuneID)
	if err != nil {
		return result, err
	}

	runestone := &runes.Runestone{
		Edicts: []runes.Edict{
			{
				RuneID: params.RuneID,
				Amount: params.TransferRuneAmount,
				Output: recipientOutput,
			},
		},
	}

	outputs := 3
	satTransferAmount := big.NewInt(0)
	if numbers.IsGreater(totalRuneAmount, params.TransferRuneAmount) {
		outputs++
		satTransferAmount.Add(satTransferAmount, nonDustBitcoinAmount)
		runestone.Pointer = &returnOutput
	}
	if params.SatoshiCommissionAmount != nil && numbers.IsPositive(params.SatoshiCommissionAmount) {
		outputs++
		satTransferAmount.Add(satTransferAmount, params.SatoshiCommissionAmount)
	}

	prepareUTXOsResult, err := PrepareUTXOs(PrepareUTXOsParams{
		Utxos:            params.BaseUTXOs,
		Inputs:           len(runeUTXOs),
		Outputs:          outputs,
		TransferAmount:   satTransferAmount,
		SatoshiPerKVByte: params.SatoshiPerKVByte,
	})
	if err != nil {
		return result, err
	}

	runestoneData, err := runestone.IntoScript()
	if err != nil {
		return result, err
	}

	tx := wire.NewMsgTx(txVersion)
	for _, i := range runeUTXOs {
		utxoHash, err := chainhash.NewHashFromStr(i.TxHash)
		if err != nil {
			return result, err
		}

		tx.AddTxIn(wire.NewTxIn(wire.NewOutPoint(utxoHash, i.Index), nil, nil))
		prepareUTXOsResult.TotalAmount.Add(prepareUTXOsResult.TotalAmount, i.Amount)
	}
	for _, i := range prepareUTXOsResult.UsedUTXOs {
		utxoHash, err := chainhash.NewHashFromStr(i.TxHash)
		if err != nil {
			return result, err
		}

		tx.AddTxIn(wire.NewTxIn(wire.NewOutPoint(utxoHash, i.Index), nil, nil))
	}

	// subtract fee.
	prepareUTXOsResult.TotalAmount.Sub(prepareUTXOsResult.TotalAmount, prepareUTXOsResult.RoughEstimate)

	// runestone output (#0).
	tx.AddTxOut(wire.NewTxOut(0, runestoneData))

	// recipient runes output (#1).
	err = b.addOutput(tx, nonDustBitcoinAmount, prepareUTXOsResult.TotalAmount, params.RecipientTaprootAddress)
	if err != nil {
		return result, err
	}

	// change runes output (#2).
	if runestone.Pointer != nil {
		err = b.addOutput(tx, nonDustBitcoinAmount, prepareUTXOsResult.TotalAmount, params.SenderTaprootAddress)
		if err != nil {
			return result, err
		}
	}

	// service commission output (#3).
	if params.SatoshiCommissionAmount != nil && numbers.IsPositive(params.SatoshiCommissionAmount) {
		err = b.addOutput(tx, params.SatoshiCommissionAmount, prepareUTXOsResult.TotalAmount, params.CommissionRecipientAddress)
		if err != nil {
			return result, err
		}
	}

	// change btc output (#4).
	if numbers.IsPositive(prepareUTXOsResult.TotalAmount) && numbers.IsGreater(prepareUTXOsResult.TotalAmount, nonDustBitcoinAmount) {
		err = b.addOutput(tx, prepareUTXOsResult.TotalAmount, prepareUTXOsResult.TotalAmount, params.SenderPaymentAddress)
		if err != nil {
			return result, err
		}
	}

	result.UnsignedRawTx = tx
	result.UsedRuneUTXOs = runeUTXOs
	result.UsedBaseUTXOs = prepareUTXOsResult.UsedUTXOs
	result.EstimatedFee = prepareUTXOsResult.RoughEstimate

	return result, nil
}

// BuildRunesTransferPSBT returns serialised PSBT from unsigned rune transferring transaction
// with indexes provided in Unknowns field defining indexes of inputs with different types.
// TODO: Fix public key parsing.
func (b *TxBuilder) BuildRunesTransferPSBT(params BuildRunesTransferPSBTParams) ([]byte, error) {
	p, err := psbt.NewFromUnsignedTx(params.UnsignedRawTx)
	if err != nil {
		return nil, err
	}

	publicKeyTP, err := hex.DecodeString(params.SenderTaprootPubKey)
	if err != nil {
		return nil, err
	}

	runeUTXOs := len(params.UsedRuneUTXOs)
	runeUTXOIndexes := make([]byte, len(params.UsedRuneUTXOs))
	for i := 0; i < runeUTXOs; i++ {
		p.Inputs[i].WitnessUtxo = wire.NewTxOut(params.UsedRuneUTXOs[i].Amount.Int64(), params.UsedRuneUTXOs[i].Script)
		p.Inputs[i].TaprootInternalKey = publicKeyTP
		p.Inputs[i].SighashType = signHashType
		runeUTXOIndexes[i] = byte(i)
	}

	publicKeyPayment, _ := hex.DecodeString(params.SenderPaymentPubKey)
	pubKey, err := btcec.ParsePubKey(publicKeyPayment)
	if err != nil {
		return nil, err
	}

	witness, err := btcutil.NewAddressWitnessPubKeyHash(btcutil.Hash160(pubKey.SerializeCompressed()), b.networkParams)
	if err != nil {
		return nil, err
	}

	witnessProg, err := txscript.PayToAddrScript(witness)
	if err != nil {
		return nil, err
	}

	baseUTXOIndexes := make([]byte, len(params.UsedBaseUTXOs))
	for i := 0; i < len(params.UsedBaseUTXOs); i++ {
		p.Inputs[i+runeUTXOs].WitnessUtxo = wire.NewTxOut(params.UsedBaseUTXOs[i].Amount.Int64(), params.UsedBaseUTXOs[i].Script)
		p.Inputs[i+runeUTXOs].RedeemScript = witnessProg
		p.Inputs[i+runeUTXOs].SighashType = signHashType
		baseUTXOIndexes[i] = byte(i + runeUTXOs)
	}

	p.Unknowns = append(p.Unknowns, &psbt.Unknown{Key: TaprootInputsHelpingKey.Bytes(), Value: runeUTXOIndexes})
	p.Unknowns = append(p.Unknowns, &psbt.Unknown{Key: PaymentInputsHelpingKey.Bytes(), Value: baseUTXOIndexes})

	w := bytes.NewBuffer(nil)
	err = p.Serialize(w)
	if err != nil {
		return nil, err
	}

	return w.Bytes(), nil
}

// BuildBTCTransferTx constructs btc transferring transaction in PSBT
// format with inputs indexes assigned in unknown fields. Returns serialized
// PSBT transaction with used base outputs, estimated fee in satoshi, and error if any.
func (b *TxBuilder) BuildBTCTransferTx(params BaseBTCTransferParams) (result BuildBTCTransferTxResult, _ error) {
	buildBaseTransferRuneTxResult, err := b.buildBaseTransferBTCTx(params)
	if err != nil {
		return result, err
	}

	result.UsedSenderBaseUTXOs = buildBaseTransferRuneTxResult.UsedSenderBaseUTXOs
	result.EstimatedFee = buildBaseTransferRuneTxResult.EstimatedFee

	psbtParams := BuildBTCTransferPSBTParams{
		BaseBTCTransferResult: buildBaseTransferRuneTxResult,
		SenderAddress:         params.Sender.Address,
		SenderPubKey:          params.Sender.PubKey,
	}

	if params.FeePayer != nil {
		psbtParams.FeePayerAddress = params.FeePayer.Address
		psbtParams.FeePayerPubKey = params.FeePayer.PubKey
	}
	result.SerializedPSBT, err = b.BuildBTCTransferPSBT(psbtParams)
	if err != nil {
		return result, err
	}

	return result, nil
}

// buildBaseTransferBTCTx constructs base btc transferring transaction.
// Returns transaction, list of used base utxos pointers, estimated fee,
// and error if any.
//
//	Tx struct
//	inputs:
//	┌─────────┬──────────────┬────────────────────────────────────────┐
//	│  index  │     type     │             description                │
//	├=========┼==============┼========================================┤
//	│   0 - k │ base inputs  │ sender's utxos with bitcoin only,      │
//	│         │              │ to transfer required amount of         │
//	│         │              │ satoshi. if the fee payer is not       │
//	│         │              │ provided, these utxos will be used to  │
//	│         │              │ pay transaction fee.                   │
//	├─────────┼──────────────┼────────────────────────────────────────┤
//	│ k+1 - n │ base inputs  │ fee payer's utxos with bitcoin only,   │
//	│         │              │ to pay transaction fee, if fee payer   │
//	│         │              │ data was provided. in this case sender │
//	│         │              │ utxos will be used to cover transfer   │
//	│         │              │ satoshi amount only.                   │
//	└─────────┴──────────────┴────────────────────────────────────────┘
//
//	outputs:
//	┌─────────┬──────────────┬────────────────────────────────────────┐
//	│  index  │     type     │             description                │
//	├=========┼==============┼========================================┤
//	│       0 │ base output  │ mandatory, output to transfer bitcoin. │
//	├─────────┼──────────────┼────────────────────────────────────────┤
//	│       1 │ base output  │ service native commission. optional,   │
//	│         │              │ charge commission from sender if       │
//	│         │              │ satoshi commission amount is not 0.    │
//	├─────────┼──────────────┼────────────────────────────────────────┤
//	│       2 │ base output  │ outputs to change sender's bitcoins    │
//	│         │              │ amount. 99% mandatory, in case         │
//	│         │              │ any non-dust btc left.                 │
//	├─────────┼──────────────┼────────────────────────────────────────┤
//	│       3 │ base output  │ outputs to change fee payer's bitcoins │
//	│         │              │ amount. optional, in case any non-dust │
//	│         │              │ btc left and the fee payer data was    │
//	│         │              │ provided.                              │
//	└─────────┴──────────────┴────────────────────────────────────────┘
func (b *TxBuilder) buildBaseTransferBTCTx(params BaseBTCTransferParams) (result BaseBTCTransferResult, _ error) {
	if params.Sender == nil {
		return result, errors.New("sender data is required")
	}

	var (
		outputs           = 2 // btc transfer + sender btc change.
		satTransferAmount = new(big.Int).Set(params.TransferSatoshiAmount)
		differentFeePayer = params.FeePayer != nil
		senderUsedUTXOs   []*bitcoin.UTXO
		feePayerUsedUTXOs []*bitcoin.UTXO
		fee               *big.Int
		bitcoinAmount     *big.Int
		senderChange      *big.Int
		feePayerChange    *big.Int
	)
	if params.SatoshiCommissionAmount != nil && numbers.IsPositive(params.SatoshiCommissionAmount) {
		outputs++ // internal commission.
		satTransferAmount.Add(satTransferAmount, params.SatoshiCommissionAmount)
	}

	if differentFeePayer {
		outputs++ // fee payer btc change.
		senderUTXOsResult, err := PrepareUTXOs(PrepareUTXOsParams{
			Utxos:          params.Sender.UTXOs,
			TransferAmount: satTransferAmount,
		})
		if err != nil {
			return result, err
		}

		feePayerUTXOsResult, err := PrepareUTXOs(PrepareUTXOsParams{
			Utxos:            params.FeePayer.UTXOs,
			Inputs:           len(senderUTXOsResult.UsedUTXOs),
			Outputs:          outputs,
			TransferAmount:   big.NewInt(0), // calculate tx fee only.
			SatoshiPerKVByte: params.SatoshiPerKVByte,
		})
		if err != nil {
			return result, err
		}

		senderUsedUTXOs = senderUTXOsResult.UsedUTXOs
		feePayerUsedUTXOs = feePayerUTXOsResult.UsedUTXOs
		bitcoinAmount = new(big.Int).Add(senderUTXOsResult.TotalAmount, feePayerUTXOsResult.TotalAmount)
		fee = feePayerUTXOsResult.RoughEstimate
		senderChange = new(big.Int).Sub(senderUTXOsResult.TotalAmount, satTransferAmount)
		feePayerChange = new(big.Int).Sub(feePayerUTXOsResult.TotalAmount, fee)
	} else {
		senderUTXOsResult, err := PrepareUTXOs(PrepareUTXOsParams{
			Utxos:            params.Sender.UTXOs,
			Inputs:           0,
			Outputs:          outputs,
			TransferAmount:   satTransferAmount,
			SatoshiPerKVByte: params.SatoshiPerKVByte,
		})
		if err != nil {
			return result, err
		}

		senderUsedUTXOs = senderUTXOsResult.UsedUTXOs
		bitcoinAmount = senderUTXOsResult.TotalAmount
		fee = senderUTXOsResult.RoughEstimate
		senderChange = new(big.Int).Sub(senderUTXOsResult.TotalAmount, satTransferAmount)
		senderChange.Sub(senderChange, fee)
	}

	tx := wire.NewMsgTx(txVersion)
	for _, i := range senderUsedUTXOs {
		utxoHash, err := chainhash.NewHashFromStr(i.TxHash)
		if err != nil {
			return result, err
		}

		tx.AddTxIn(wire.NewTxIn(wire.NewOutPoint(utxoHash, i.Index), nil, nil))
	}
	for _, i := range feePayerUsedUTXOs {
		utxoHash, err := chainhash.NewHashFromStr(i.TxHash)
		if err != nil {
			return result, err
		}

		tx.AddTxIn(wire.NewTxIn(wire.NewOutPoint(utxoHash, i.Index), nil, nil))
	}

	// subtract fee.
	bitcoinAmount.Sub(bitcoinAmount, fee)

	// recipient btc output (#0).
	err := b.addOutput(tx, params.TransferSatoshiAmount, bitcoinAmount, params.RecipientAddress)
	if err != nil {
		return result, err
	}

	// service commission output (#1).
	if params.SatoshiCommissionAmount != nil && numbers.IsPositive(params.SatoshiCommissionAmount) {
		err = b.addOutput(tx, params.SatoshiCommissionAmount, bitcoinAmount, params.CommissionReceiverAddress)
		if err != nil {
			return result, err
		}
	}

	// sender's change btc output (#2).
	if numbers.IsGreater(senderChange, nonDustBitcoinAmount) {
		err = b.addOutput(tx, senderChange, bitcoinAmount, params.Sender.Address)
		if err != nil {
			return result, err
		}
	}

	// fee payer's change btc output (#3).
	if differentFeePayer && numbers.IsGreater(feePayerChange, nonDustBitcoinAmount) {
		err = b.addOutput(tx, feePayerChange, bitcoinAmount, params.FeePayer.Address)
		if err != nil {
			return result, err
		}
	}

	result.UnsignedRawTx = tx
	result.UsedSenderBaseUTXOs = senderUsedUTXOs
	result.UsedFeePayerBaseUTXOs = feePayerUsedUTXOs
	result.EstimatedFee = fee

	return result, nil
}

// BuildBTCTransferPSBT returns serialised PSBT from unsigned btc transferring transaction
// with indexes provided in Unknowns field defining indexes of inputs with different types.
func (b *TxBuilder) BuildBTCTransferPSBT(params BuildBTCTransferPSBTParams) ([]byte, error) {
	p, err := psbt.NewFromUnsignedTx(params.UnsignedRawTx)
	if err != nil {
		return nil, err
	}

	var (
		senderAddressData   addressData
		feePayerAddressData addressData
	)
	senderAddressData, err = b.prepareAddressData(params.SenderPubKey, params.SenderAddress)
	if err != nil {
		return nil, err
	}

	if len(params.UsedFeePayerBaseUTXOs) != 0 {
		feePayerAddressData, err = b.prepareAddressData(params.FeePayerPubKey, params.FeePayerAddress)
		if err != nil {
			return nil, err
		}
	}

	senderIndexes := make([]byte, len(params.UsedSenderBaseUTXOs))
	for i, utxo := range params.UsedSenderBaseUTXOs {
		switch senderAddressData.addrType {
		case TaprootInputsHelpingKey:
			p.Inputs[i].TaprootInternalKey = senderAddressData.publicKeyBytes
		case PaymentInputsHelpingKey:
			p.Inputs[i].RedeemScript = senderAddressData.witnessProg
		}
		p.Inputs[i].WitnessUtxo = wire.NewTxOut(utxo.Amount.Int64(), utxo.Script)
		p.Inputs[i].SighashType = signHashType
		senderIndexes[i] = byte(i)
	}

	p.Unknowns = append(p.Unknowns, &psbt.Unknown{Key: senderAddressData.addrType.Bytes(), Value: senderIndexes})

	if len(params.UsedFeePayerBaseUTXOs) != 0 {
		switch feePayerAddressData.addrType {
		case TaprootInputsHelpingKey:
			feePayerAddressData.addrType = FeePayerTaprootInputsHelpingKey
		case PaymentInputsHelpingKey:
			feePayerAddressData.addrType = FeePayerPaymentInputsHelpingKey
		}

		feePayerIndexes := make([]byte, len(params.UsedFeePayerBaseUTXOs))
		for i, utxo := range params.UsedFeePayerBaseUTXOs {
			switch feePayerAddressData.addrType {
			case FeePayerTaprootInputsHelpingKey:
				p.Inputs[i].TaprootInternalKey = feePayerAddressData.publicKeyBytes
			case FeePayerPaymentInputsHelpingKey:
				p.Inputs[i].RedeemScript = feePayerAddressData.witnessProg
			}
			p.Inputs[i].WitnessUtxo = wire.NewTxOut(utxo.Amount.Int64(), utxo.Script)
			p.Inputs[i].SighashType = signHashType
			feePayerIndexes[i] = byte(i)
		}

		p.Unknowns = append(p.Unknowns, &psbt.Unknown{Key: feePayerAddressData.addrType.Bytes(), Value: feePayerIndexes})
	}

	w := bytes.NewBuffer(nil)
	err = p.Serialize(w)
	if err != nil {
		return nil, err
	}

	return w.Bytes(), nil
}

// addressData defines helping address data to build psbt.
type addressData struct {
	addrType       InputsHelpingKey
	publicKeyBytes []byte
	publicKeyBtcec *btcec.PublicKey
	witness        *btcutil.AddressWitnessPubKeyHash
	witnessProg    []byte
}

// prepareAddressData returns addressData from public key and address.
func (b *TxBuilder) prepareAddressData(pk, address string) (result addressData, err error) {
	result.publicKeyBytes, err = hex.DecodeString(pk)
	if err != nil {
		return result, err
	}

	addressType, err := btcutil.DecodeAddress(address, b.networkParams)
	if err != nil {
		return result, err
	}

	switch addressType.(type) {
	case *btcutil.AddressTaproot:
		result.addrType = TaprootInputsHelpingKey
		if len(result.publicKeyBytes) == 33 {
			result.publicKeyBytes = result.publicKeyBytes[1:]
		}
	case *btcutil.AddressPubKeyHash, *btcutil.AddressPubKey, *btcutil.AddressScriptHash:
		result.addrType = PaymentInputsHelpingKey
		result.publicKeyBtcec, err = btcec.ParsePubKey(result.publicKeyBytes)
		if err != nil {
			return result, err
		}

		result.witness, err = btcutil.NewAddressWitnessPubKeyHash(btcutil.Hash160(result.publicKeyBtcec.SerializeCompressed()), b.networkParams)
		if err != nil {
			return result, err
		}

		result.witnessProg, err = txscript.PayToAddrScript(result.witness)
		if err != nil {
			return result, err
		}
	default:
		return result, btcutil.ErrUnknownAddressType
	}

	return result, nil
}

// PrepareUTXOs selects utxos to cover rough estimated fee.
// Returns used utxos, total satoshi amount of utxos, rough estimation in satoshi and error if any.
func PrepareUTXOs(params PrepareUTXOsParams) (result PrepareUTXOsResult, err error) {
	satFn := func(u *bitcoin.UTXO) *big.Int { return u.Amount }

	var fullParams = !(params.SatoshiPerKVByte == nil && params.Inputs == 0 && params.Outputs == 0)
	for i := 1; i <= len(params.Utxos); i++ {
		if fullParams {
			// vB * ( sat / kvB ) = 1000 sat.
			result.RoughEstimate = new(big.Int).Mul(RoughTxSizeEstimate(i+params.Inputs, params.Outputs),
				params.SatoshiPerKVByte)
			result.RoughEstimate.Div(result.RoughEstimate, big.NewInt(1000)) // sat.

			result.UsedUTXOs, result.TotalAmount, err = SelectUTXO(params.Utxos, satFn,
				new(big.Int).Add(result.RoughEstimate, params.TransferAmount), i, bitcoin.ErrInsufficientNativeBalance)
		} else {
			result.UsedUTXOs, result.TotalAmount, err = SelectUTXO(params.Utxos, satFn,
				new(big.Int).Set(params.TransferAmount), i, bitcoin.ErrInsufficientNativeBalance)
		}
		if err != nil {
			if errors.Is(err, bitcoin.ErrInsufficientNativeBalance) {
				continue
			}

			return result, err
		}

		return result, nil
	}

	return result, bitcoin.ErrInsufficientNativeBalance
}

// PrepareUTXOsParams defines parameters for PrepareUTXOs function.
//
//	Parameter groups:
//	- Utxos, TransferAmount - to select utxos for transfer only.
//	- Utxos, Inputs, Outputs, TransferAmount, SatoshiPerKVByte - to select utxos for transfer including fee estimation.
type PrepareUTXOsParams struct {
	Utxos            []bitcoin.UTXO
	Inputs           int
	Outputs          int
	TransferAmount   *big.Int
	SatoshiPerKVByte *big.Int
}

// PrepareUTXOsResult describes result of the PrepareUTXOs function.
// In case all values in the PrepareUTXOsParams were transmitted,
// all values of the PrepareUTXOsResult will be created. Otherwise,
// RoughEstimate will be zero on nil.
type PrepareUTXOsResult struct {
	UsedUTXOs     []*bitcoin.UTXO
	TotalAmount   *big.Int
	RoughEstimate *big.Int
}

// PrepareRuneUTXOs selects utxos to cover rune transfer amount.
// Returns used utxos, total rune amount of utxos and error if any.
func PrepareRuneUTXOs(utxos []bitcoin.UTXO, transferAmount *big.Int, runeID runes.RuneID) (usedUTXOs []*bitcoin.UTXO, totalAmount *big.Int, err error) {
	runeFn := func(u *bitcoin.UTXO) *big.Int {
		for _, rune_ := range u.Runes {
			if rune_.RuneID == runeID {
				return rune_.Amount
			}
		}

		return big.NewInt(0)
	}

	for i := 1; i <= len(utxos); i++ {
		usedUTXOs, totalAmount, err = SelectUTXO(utxos, runeFn, transferAmount, i, bitcoin.ErrInsufficientRuneBalance)
		if err != nil {
			if errors.Is(err, bitcoin.ErrInsufficientRuneBalance) {
				continue
			}

			return nil, nil, err
		}

		return usedUTXOs, totalAmount, nil
	}

	return nil, nil, bitcoin.ErrInsufficientRuneBalance
}

// RoughTxSizeEstimate returns Tx rough estimated size in vBytes.
// TODO: increase precision.
func RoughTxSizeEstimate(inputs, outputs int) *big.Int {
	size := new(big.Int).Set(headerSizeVBytes)
	size.Add(size, new(big.Int).Mul(inputSizeVBytes, big.NewInt(int64(inputs))))
	size.Add(size, new(big.Int).Mul(outputSizeVBytes, big.NewInt(int64(outputs))))

	return size
}

// SelectUTXO is a partly greedy selection algorithm for UTXOs with 'requiredUTXOs' parameter.
// Returns list of selected by algorithm UTXOs with total amount, counted by passed amount function.
func SelectUTXO(utxos []bitcoin.UTXO, amountFn func(*bitcoin.UTXO) *big.Int, minAmount *big.Int, requiredUTXOs int,
	insufficientBalanceError error) (usedUTXOs []*bitcoin.UTXO, totalAmount *big.Int, _ error) {
	if len(utxos) < requiredUTXOs {
		return nil, nil, bitcoin.ErrInvalidUTXOAmount
	}

	usedUTXOs = make([]*bitcoin.UTXO, 0, requiredUTXOs)
	totalAmount = big.NewInt(0)
	var startIdx = 0
	var usedIdxs = make([]int, 0)

	// find the closest by amount UTXO that is grater then minAmount or take the biggest possible.
	for idx, utxo := range utxos {
		if numbers.IsGreater(minAmount, amountFn(&utxo)) {
			break
		}

		startIdx = idx
	}

	usedIdxs = append(usedIdxs, startIdx)
	totalAmount.Add(totalAmount, amountFn(&utxos[startIdx]))
	usedUTXOs = append(usedUTXOs, &utxos[startIdx])
	requiredUTXOs--

	// pick bigger amount if total amount do not cover minAmount, otherwise - the smallest to pass requiredUTXOs.
	for ; requiredUTXOs > 0; requiredUTXOs-- {
		idx := selectUnused(startIdx, len(utxos), usedIdxs, !numbers.IsGreater(minAmount, totalAmount))
		if idx == -1 {
			return nil, nil, bitcoin.ErrInvalidUTXOAmount
		}

		usedIdxs = append(usedIdxs, idx)
		totalAmount.Add(totalAmount, amountFn(&utxos[idx]))
		usedUTXOs = append(usedUTXOs, &utxos[idx])
	}

	if numbers.IsGreater(minAmount, totalAmount) {
		return nil, nil, insufficientBalanceError
	}

	return usedUTXOs, totalAmount, nil
}

// addOutput adds output to transaction, subtracts amount from unallocated amount.
func (b *TxBuilder) addOutput(tx *wire.MsgTx, amount, unallocatedAmount *big.Int, address string) error {
	if numbers.IsLess(unallocatedAmount, amount) {
		return errors.New("unallocated amount is less than the amount in provided inputs")
	}

	recipientAddress, err := btcutil.DecodeAddress(address, b.networkParams)
	if err != nil {
		return err
	}

	destinationAddrByte, err := txscript.PayToAddrScript(recipientAddress)
	if err != nil {
		return err
	}

	tx.AddTxOut(wire.NewTxOut(amount.Int64(), destinationAddrByte))
	unallocatedAmount.Sub(unallocatedAmount, amount)

	return nil
}

// selectUnused returns first unused idx depending on search direction.
func selectUnused(start, end int, usedIdxs []int, reversed bool) int {
	if reversed {
		for idx := end - 1; idx >= start; idx-- {
			if !isUsed(idx, usedIdxs) {
				return idx
			}
		}
	} else {
		for idx := start; idx < end; idx++ {
			if !isUsed(idx, usedIdxs) {
				return idx
			}
		}
	}

	return -1
}

// isUsed returns true id idx is in usedIdxs.
func isUsed(idx int, usedIdxs []int) bool {
	for _, used := range usedIdxs {
		if used == idx {
			return true
		}
	}

	return false
}
