// Copyright (C) 2024 Creditor Corp. Group.
// See LICENSE for copying information.

package txbuilder

import (
	"bytes"
	"errors"
	"fmt"
	"math/big"

	"github.com/btcsuite/btcd/btcutil"
	"github.com/btcsuite/btcd/btcutil/psbt"
	"github.com/btcsuite/btcd/chaincfg"
	"github.com/btcsuite/btcd/chaincfg/chainhash"
	"github.com/btcsuite/btcd/txscript"
	"github.com/btcsuite/btcd/wire"

	"github.com/BoostyLabs/blockchain/bitcoin"
	"github.com/BoostyLabs/blockchain/bitcoin/ord/inscriptions"
	"github.com/BoostyLabs/blockchain/bitcoin/ord/runes"
	"github.com/BoostyLabs/blockchain/internal/numbers"
)

var (
	// InsufficientNativeBalanceError describes class of errors when there is not enough native balance to cover the payment.
	InsufficientNativeBalanceError = &InsufficientError{Type: InsufficientErrorTypeBitcoin}
	// InsufficientRuneBalanceError describes class of errors when there is not enough rune balance to cover the payment.
	InsufficientRuneBalanceError = &InsufficientError{Type: InsufficientErrorTypeRune}
	// ErrInvalidUTXOAmount describes that there was invalid UTXO amount transmitted.
	ErrInvalidUTXOAmount = errors.New("invalid UTXO amount")
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

	// inscriptionInputSizeVBytes defined rough tx input size in vBytes
	// with signature, but without witness script data size.
	inscriptionInputSizeVBytes = big.NewInt(61)

	// nonDustBitcoinAmount defined the smallest needed amount in satoshi to link to rune output.
	nonDustBitcoinAmount = big.NewInt(546)

	// recipientOutput defines runes output for recipient (transferring) by base rune tx.
	recipientOutput uint32 = 1
	// returnOutput defines runes output for sender (change) by base rune tx.
	returnOutput uint32 = 2
)

// BaseRunesTransferParams describes basic data needed to build rune transfer transaction.
// NOTE: fee payer's utxos should contain btc only, any joined runes will transferred to RunesRecipientAddress.
type BaseRunesTransferParams struct {
	RuneID             runes.RuneID
	TransferRuneAmount *big.Int // runes amount to transfer.
	// BurnRuneAmount is a runes amount to burn. all burning processes are applied after transferring only.
	// If burn amount is greater than total transfer amount, then only the absolute difference be burned or 0 (what is greater).
	BurnRuneAmount             *big.Int
	RunesSender                *PaymentData // mandatory. must be sorted by rune amount desc.
	FeePayer                   *PaymentData // mandatory. must be sorted by btc amount desc.
	SatoshiPerKVByte           *big.Int     // fee rate in satoshi per kilo virtual byte.
	RunesRecipientAddress      string       // recipient runes address.
	SatoshiCommissionAmount    *big.Int     // additional commission in satoshi to be charged from user.
	CommissionRecipientAddress string       // recipient commission address.
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
	RunesSenderPubKey  string
	RunesSenderAddress string
	FeePayerPubKey     string
	FeePayerAddress    string
}

// PaymentData defined data needed to construct inputs.
type PaymentData struct {
	UTXOs   []bitcoin.UTXO // must be sorted by target token amount desc.
	Address string         // payer address.
	PubKey  string         // payer public key.
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

// BaseBTCTransferResult describes result of buildBaseTransferBTCTx method.
type BaseBTCTransferResult struct {
	UnsignedRawTx         *wire.MsgTx     // unsigned btc transfer transaction.
	UsedSenderBaseUTXOs   []*bitcoin.UTXO // used sender's bitcoin utxos in transaction.
	UsedFeePayerBaseUTXOs []*bitcoin.UTXO // used fee payer's bitcoin utxos in transaction.
	EstimatedFee          *big.Int        // estimated transaction fee in Satoshi.
}

// BuildBTCTransferTxResult describes result of BuildBTCTransferTx method.
type BuildBTCTransferTxResult struct {
	SerializedPSBT        []byte          // serialised unsigned btc transfer transaction in PSBT format.
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

// BaseInscriptionTxParams describes basic data needed to build inscription commitment transaction.
// NOTE: utxos should contain btc only, any joined runes will be lost.
type BaseInscriptionTxParams struct {
	Sender                    *PaymentData              // sender payment data. mandatory.
	SatoshiPerKVByte          *big.Int                  // fee rate in satoshi per kilo virtual byte.
	SatoshiCommissionAmount   *big.Int                  // additional commission in satoshi to be charged from user, optional.
	CommissionReceiverAddress string                    // recipient commission address, optional.
	Inscription               *inscriptions.Inscription // inscription data to commit.
	InscriptionBasePubKey     string                    // public key needed to create inscription address.
	PremineSplittingFactor    uint                      // for more details see [BaseRuneEtchTxParams.PremineSplittingFactor].
}

// BaseInscriptionTxResult describes result of buildBaseInscriptionTx method.
type BaseInscriptionTxResult struct {
	UnsignedRawTx *wire.MsgTx     // unsigned inscription commitment transaction.
	UsedBaseUTXOs []*bitcoin.UTXO // used sender's bitcoin utxos in transaction.
	EstimatedFee  *big.Int        // estimated transaction fee in Satoshi.
}

// BuildInscriptionTxPSBTParams describes data needed to convert unsigned
// inscription commitment transaction to partly signed bitcoin transaction (PSBT).
type BuildInscriptionTxPSBTParams struct {
	BaseInscriptionTxResult
	SenderAddress string
	SenderPubKey  string
}

// BuildInscriptionTxPSBTResult describes result of buildInscriptionTxPSBT method.
type BuildInscriptionTxPSBTResult struct {
	SerializedPSBT []byte          // serialised unsigned inscription commitment transaction in PSBT format.
	UsedBaseUTXOs  []*bitcoin.UTXO // used sender's bitcoin utxos in transaction.
	EstimatedFee   *big.Int        // estimated transaction fee in Satoshi.
}

// BaseRuneEtchTxParams describes basic data needed to build inscription reveal - etch transaction.
// NOTE: utxos should contain btc only, any joined runes will be transferred to RunesRecipientAddress.
type BaseRuneEtchTxParams struct {
	InscriptionReveal     *PaymentData              // inscription commitment data. mandatory. must contain one utxo only. address can be omitted.
	Inscription           *inscriptions.Inscription // used inscription data.
	Rune                  *runes.Etching            // rune etching data.
	AdditionalPayments    *PaymentData              // sender payment data. mandatory.
	SatoshiPerKVByte      *big.Int                  // fee rate in satoshi per kilo virtual byte.
	RunesRecipientAddress string                    // recipient address to receive etched runes.
	SatoshiChangeAddress  string                    // address to receive btc change if any left.
	// PremineSplittingFactor defines between how many utxos premine value will be split.
	// NOTE: Valid if [Rune.Premine] is positive. Must be less then premine value.
	// Premine value splitting rules
	//  - Rune.Premine % PremineSplittingFactor == 0.
	//  The edict with amount equals quotient and index equals outputs number is applied to the Runestone.
	//  Example: premine = 12000, PremineSplittingFactor = 6:
	//  []Edicts{ Edict{RuneID: (0,0),Amount: Rune.Premine / PremineSplittingFactor, Output: len(ouputs)} }.
	//  As a result there will be: 0 output - Runestone, 1-7 outputs - each containing 2000 runes, 8 - optional change output.
	//  - Rune.Premine % PremineSplittingFactor == 0.
	//  The edict from the upper case is applied, but before the remainder edict is applied to the first non-OR_RETURN output.
	//  Ex. premine = 12005, PremineSplittingFactor = 6:
	//  []Edicts{
	//  Edict{RuneID: (0,0),Amount: Rune.Premine % PremineSplittingFactor, Output: 1},
	//  Edict{RuneID: (0,0),Amount: Rune.Premine / PremineSplittingFactor, Output: len(ouputs)},
	//  }.
	//  As a result there will be: 0 output - Runestone, 1 output - 2000 + 5 runes, 2-7 outputs, each containing 2000 runes,
	//  8 - optional change output.
	PremineSplittingFactor uint
}

// BaseRuneEtchTxResult describes result of buildBaseRuneEtchTx method.
type BaseRuneEtchTxResult struct {
	UnsignedRawTx           *wire.MsgTx               // unsigned inscription reveal - etch transaction.
	InscriptionUTXO         bitcoin.UTXO              // reveal inscription utxo data.
	InscriptionReveal       *inscriptions.Inscription // used inscription data.
	UsedAdditionalBaseUTXOs []*bitcoin.UTXO           // used additional payment bitcoin utxos in transaction.
	EstimatedFee            *big.Int                  // estimated transaction fee in Satoshi.
}

// BuildRuneEtchTxPSBTParams describes data needed to convert unsigned inscription
// reveal - etch transaction to partly signed bitcoin transaction (PSBT).
type BuildRuneEtchTxPSBTParams struct {
	BaseRuneEtchTxResult
	InscriptionBasePubKey     string // public key needed to create inscription address.
	InscriptionBaseAddress    string // inscription generated address.
	AdditionalPaymentsAddress string
	AdditionalPaymentsPubKey  string
}

// BuildRuneEtchTxPSBTResult describes result of BuildRuneEtchTx method.
type BuildRuneEtchTxPSBTResult struct {
	SerializedPSBT          []byte          // serialised unsigned inscription reveal - etch transaction in PSBT format.
	UsedAdditionalBaseUTXOs []*bitcoin.UTXO // used additional payment bitcoin utxos in transaction.
	EstimatedFee            *big.Int        // estimated transaction fee in Satoshi.
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

	result.SerializedPSBT, err = b.buildRunesTransferPSBT(BuildRunesTransferPSBTParams{
		BaseRunesTransferResult: buildBaseTransferRuneTxResult,
		RunesSenderPubKey:       params.RunesSender.PubKey,
		RunesSenderAddress:      params.RunesSender.Address,
		FeePayerPubKey:          params.FeePayer.PubKey,
		FeePayerAddress:         params.FeePayer.Address,
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
//	│       1 │ rune output  │ optional, output to link runes         │
//	│         │              │ to recipient, present if rune transfer │
//	│         │              │ is positive.                           │
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
	if params.RunesSender == nil {
		return result, errors.New("runes sender data required")
	}
	if params.FeePayer == nil {
		return result, errors.New("fee payer data required")
	}
	if params.TransferRuneAmount == nil || numbers.IsNegative(params.TransferRuneAmount) {
		params.TransferRuneAmount = big.NewInt(0)
	}
	if params.BurnRuneAmount == nil || numbers.IsNegative(params.BurnRuneAmount) {
		params.BurnRuneAmount = big.NewInt(0)
	}

	totalAllocatingRuneAmount := new(big.Int).Add(params.TransferRuneAmount, params.BurnRuneAmount)
	runeUTXOs, totalRuneAmount, err := PrepareRuneUTXOs(params.RunesSender.UTXOs, totalAllocatingRuneAmount, params.RuneID)
	if err != nil {
		if errIns := new(InsufficientError); errors.As(err, &errIns) {
			return result, errIns.setCauser(CauserSender)
		}

		return result, err
	}

	outputs := 2
	satTransferAmount := big.NewInt(0)
	runestone := &runes.Runestone{}
	isRunesTransferred := false

	// runes transfer output + edict.
	if numbers.IsPositive(params.TransferRuneAmount) {
		isRunesTransferred = true
		outputs++
		satTransferAmount.Add(satTransferAmount, nonDustBitcoinAmount)

		runestone.Edicts = append(runestone.Edicts, runes.Edict{
			RuneID: params.RuneID,
			Amount: params.TransferRuneAmount,
			Output: recipientOutput,
		})
	}
	if numbers.IsPositive(params.BurnRuneAmount) {
		runestone.Edicts = append(runestone.Edicts, runes.Edict{
			RuneID: params.RuneID,
			Amount: params.BurnRuneAmount,
			Output: 0,
		})
	}

	// runes return output.
	if numbers.IsGreater(totalRuneAmount, totalAllocatingRuneAmount) {
		outputs++
		satTransferAmount.Add(satTransferAmount, nonDustBitcoinAmount)
		returnOutputCopy := returnOutput
		runestone.Pointer = &returnOutputCopy
		if !isRunesTransferred {
			*runestone.Pointer--
		}
	}

	// commission output.
	if params.SatoshiCommissionAmount != nil && numbers.IsPositive(params.SatoshiCommissionAmount) {
		outputs++
		satTransferAmount.Add(satTransferAmount, params.SatoshiCommissionAmount)
	}

	prepareUTXOsResult, err := PrepareUTXOs(PrepareUTXOsParams{
		Utxos:            params.FeePayer.UTXOs,
		Inputs:           len(runeUTXOs),
		Outputs:          outputs,
		TransferAmount:   satTransferAmount,
		SatoshiPerKVByte: params.SatoshiPerKVByte,
	})
	if err != nil {
		if errIns := new(InsufficientError); errors.As(err, &errIns) {
			return result, errIns.setCauser(CauserFeePayer)
		}

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
	if isRunesTransferred {
		err = b.addOutput(tx, nonDustBitcoinAmount, prepareUTXOsResult.TotalAmount, params.RunesRecipientAddress)
		if err != nil {
			return result, err
		}
	}

	// change runes output (#2).
	if runestone.Pointer != nil {
		err = b.addOutput(tx, nonDustBitcoinAmount, prepareUTXOsResult.TotalAmount, params.RunesSender.Address)
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
		err = b.addOutput(tx, prepareUTXOsResult.TotalAmount, prepareUTXOsResult.TotalAmount, params.FeePayer.Address)
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

// buildRunesTransferPSBT returns serialised PSBT from unsigned rune transferring transaction
// with indexes provided in Unknowns field defining indexes of inputs with different types.
func (b *TxBuilder) buildRunesTransferPSBT(params BuildRunesTransferPSBTParams) ([]byte, error) {
	p, err := psbt.NewFromUnsignedTx(params.UnsignedRawTx)
	if err != nil {
		return nil, err
	}

	runesSenderInputBuilder, err := NewPSBTInputBuilder(params.RunesSenderPubKey, params.RunesSenderAddress, b.networkParams)
	if err != nil {
		return nil, err
	}

	feePayerAddressInputBuilder, err := NewPSBTInputBuilder(params.FeePayerPubKey, params.FeePayerAddress, b.networkParams)
	if err != nil {
		return nil, err
	}

	senderIndexes := make([]byte, len(params.UsedRuneUTXOs))
	for i, utxo := range params.UsedRuneUTXOs {
		runesSenderInputBuilder.PrepareInput(&(p.Inputs[i]))
		p.Inputs[i].WitnessUtxo = wire.NewTxOut(utxo.Amount.Int64(), utxo.Script)
		p.Inputs[i].SighashType = signHashType
		senderIndexes[i] = byte(i)
	}

	p.Unknowns = append(p.Unknowns, &psbt.Unknown{Key: runesSenderInputBuilder.InputsHelpingKey(false).Bytes(), Value: senderIndexes})

	shift := len(params.UsedRuneUTXOs) // sender runes utxos inputs shift.
	feePayerIndexes := make([]byte, len(params.UsedBaseUTXOs))
	for i, utxo := range params.UsedBaseUTXOs {
		feePayerAddressInputBuilder.PrepareInput(&(p.Inputs[shift+i]))
		p.Inputs[shift+i].WitnessUtxo = wire.NewTxOut(utxo.Amount.Int64(), utxo.Script)
		p.Inputs[shift+i].SighashType = signHashType
		feePayerIndexes[i] = byte(shift + i)
	}

	p.Unknowns = append(p.Unknowns, &psbt.Unknown{Key: feePayerAddressInputBuilder.InputsHelpingKey(true).Bytes(), Value: feePayerIndexes})

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
	result.SerializedPSBT, err = b.buildBTCTransferPSBT(psbtParams)
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
	if len(params.Sender.UTXOs) == 0 {
		return result, errors.New("sender utxos len: 0")
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
			if errIns := new(InsufficientError); errors.As(err, &errIns) {
				return result, errIns.setCauser(CauserSender)
			}

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
			if errIns := new(InsufficientError); errors.As(err, &errIns) {
				return result, errIns.setCauser(CauserFeePayer)
			}

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
			if errIns := new(InsufficientError); errors.As(err, &errIns) {
				return result, errIns.setCauser(CauserSender)
			}

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

// buildBTCTransferPSBT returns serialised PSBT from unsigned btc transferring transaction
// with indexes provided in Unknowns field defining indexes of inputs with different types.
func (b *TxBuilder) buildBTCTransferPSBT(params BuildBTCTransferPSBTParams) ([]byte, error) {
	p, err := psbt.NewFromUnsignedTx(params.UnsignedRawTx)
	if err != nil {
		return nil, err
	}

	var (
		senderInputBuilder   *PSBTInputBuilder
		feePayerInputBuilder *PSBTInputBuilder
	)
	senderInputBuilder, err = NewPSBTInputBuilder(params.SenderPubKey, params.SenderAddress, b.networkParams)
	if err != nil {
		return nil, err
	}

	if len(params.UsedFeePayerBaseUTXOs) != 0 {
		feePayerInputBuilder, err = NewPSBTInputBuilder(params.FeePayerPubKey, params.FeePayerAddress, b.networkParams)
		if err != nil {
			return nil, err
		}
	}

	senderIndexes := make([]byte, len(params.UsedSenderBaseUTXOs))
	for i, utxo := range params.UsedSenderBaseUTXOs {
		senderInputBuilder.PrepareInput(&(p.Inputs[i]))
		p.Inputs[i].WitnessUtxo = wire.NewTxOut(utxo.Amount.Int64(), utxo.Script)
		p.Inputs[i].SighashType = signHashType
		senderIndexes[i] = byte(i)
	}

	p.Unknowns = append(p.Unknowns, &psbt.Unknown{Key: senderInputBuilder.InputsHelpingKey(false).Bytes(), Value: senderIndexes})

	if len(params.UsedFeePayerBaseUTXOs) != 0 {
		shift := len(params.UsedSenderBaseUTXOs) // sender utxos inputs shift.
		feePayerIndexes := make([]byte, len(params.UsedFeePayerBaseUTXOs))
		for i, utxo := range params.UsedFeePayerBaseUTXOs {
			feePayerInputBuilder.PrepareInput(&(p.Inputs[shift+i]))
			p.Inputs[shift+i].WitnessUtxo = wire.NewTxOut(utxo.Amount.Int64(), utxo.Script)
			p.Inputs[shift+i].SighashType = signHashType
			feePayerIndexes[i] = byte(shift + i)
		}

		p.Unknowns = append(p.Unknowns, &psbt.Unknown{Key: feePayerInputBuilder.InputsHelpingKey(true).Bytes(), Value: feePayerIndexes})
	}

	w := bytes.NewBuffer(nil)
	err = p.Serialize(w)
	if err != nil {
		return nil, err
	}

	return w.Bytes(), nil
}

// BuildInscriptionTx constructs inscription commitment transaction in PSBT
// format with inputs indexes assigned in unknown fields. Includes estimated
// transaction fee for inscription reveal - etching transaction. Returns serialized
// PSBT transaction with used base outputs, estimated fee in satoshi, and error if any.
func (b *TxBuilder) BuildInscriptionTx(params BaseInscriptionTxParams) (result BuildInscriptionTxPSBTResult, _ error) {
	buildBaseInscriptionTxResult, err := b.buildBaseInscriptionTx(params)
	if err != nil {
		return result, err
	}

	result.UsedBaseUTXOs = buildBaseInscriptionTxResult.UsedBaseUTXOs
	result.EstimatedFee = buildBaseInscriptionTxResult.EstimatedFee

	result.SerializedPSBT, err = b.buildInscriptionTxPSBT(BuildInscriptionTxPSBTParams{
		BaseInscriptionTxResult: buildBaseInscriptionTxResult,
		SenderAddress:           params.Sender.Address,
		SenderPubKey:            params.Sender.PubKey,
	})
	if err != nil {
		return result, err
	}

	return result, nil
}

// buildBaseInscriptionTx constructs base inscription commitment transaction.
// Returns transaction, list of used base utxos pointers, estimated fee,
// and error if any.
//
//	Tx struct
//	inputs:
//	┌─────────┬──────────────┬────────────────────────────────────────┐
//	│  index  │     type     │             description                │
//	├=========┼==============┼========================================┤
//	│   0 - n │ base inputs  │ sender's utxos with bitcoin only,      │
//	│         │              │ to transfer required amount of         │
//	│         │              │ satoshi, to cover commit transaction   │
//	│         │              │ fee and additional transaction fee for │
//	│         │              │ reveal-etch transaction.               │
//	└─────────┴──────────────┴────────────────────────────────────────┘
//
//	outputs:
//	┌─────────┬──────────────┬────────────────────────────────────────┐
//	│  index  │     type     │             description                │
//	├=========┼==============┼========================================┤
//	│       0 │ base output  │ mandatory, output to transfer bitcoin  │
//	│         │              │ to created by inscription address.     │
//	├─────────┼──────────────┼────────────────────────────────────────┤
//	│       1 │ base output  │ service native commission. optional,   │
//	│         │              │ charge commission from sender if       │
//	│         │              │ satoshi commission amount is not 0.    │
//	├─────────┼──────────────┼────────────────────────────────────────┤
//	│       2 │ base output  │ outputs to change sender's bitcoins    │
//	│         │              │ amount. 99% mandatory, in case         │
//	│         │              │ any non-dust btc left.                 │
//	└─────────┴──────────────┴────────────────────────────────────────┘
func (b *TxBuilder) buildBaseInscriptionTx(params BaseInscriptionTxParams) (result BaseInscriptionTxResult, err error) {
	if params.Sender == nil {
		return result, errors.New("sender data is required")
	}
	if len(params.Sender.UTXOs) == 0 {
		return result, errors.New("sender utxos len: 0")
	}
	if params.PremineSplittingFactor == 0 {
		params.PremineSplittingFactor = 1 // INFO: set to default.
	}

	var (
		outputs                = 2 // inscription commitment + sender btc change.
		satTransferAmount      = big.NewInt(0)
		inscriptionAddress     string
		inscriptionWitnessSize int
		depositAmount          = big.NewInt(0)
	)
	if params.SatoshiCommissionAmount != nil && numbers.IsPositive(params.SatoshiCommissionAmount) {
		outputs++ // internal commission.
		satTransferAmount.Add(satTransferAmount, params.SatoshiCommissionAmount)
	}

	inscriptionAddress, err = params.Inscription.IntoAddress(params.InscriptionBasePubKey, b.networkParams)
	if err != nil {
		return result, err
	}

	inscriptionWitnessSize, err = params.Inscription.VBytesSize()
	if err != nil {
		return result, err
	}

	etchTransactionFee := RoughEtchFeeEstimate(big.NewInt(int64(inscriptionWitnessSize)),
		params.SatoshiPerKVByte, int(params.PremineSplittingFactor))
	depositAmount.Add(depositAmount, etchTransactionFee)
	depositAmount.Add(depositAmount, new(big.Int).Mul(nonDustBitcoinAmount,
		big.NewInt(int64(params.PremineSplittingFactor)))) // INFO: add runes recipient output.

	satTransferAmount.Add(satTransferAmount, depositAmount)
	senderUTXOsResult, err := PrepareUTXOs(PrepareUTXOsParams{
		Utxos:            params.Sender.UTXOs,
		Inputs:           0,
		Outputs:          outputs,
		TransferAmount:   satTransferAmount,
		SatoshiPerKVByte: params.SatoshiPerKVByte,
	})
	if err != nil {
		if errIns := new(InsufficientError); errors.As(err, &errIns) {
			return result, errIns.setCauser(CauserSender)
		}

		return result, err
	}

	bitcoinAmount := new(big.Int).Set(senderUTXOsResult.TotalAmount)

	tx := wire.NewMsgTx(txVersion)
	for _, i := range senderUTXOsResult.UsedUTXOs {
		utxoHash, err := chainhash.NewHashFromStr(i.TxHash)
		if err != nil {
			return result, err
		}

		tx.AddTxIn(wire.NewTxIn(wire.NewOutPoint(utxoHash, i.Index), nil, nil))
	}

	// subtract fee.
	bitcoinAmount.Sub(bitcoinAmount, senderUTXOsResult.RoughEstimate)

	// inscription commitment output (#0).
	err = b.addOutput(tx, depositAmount, bitcoinAmount, inscriptionAddress)
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
	if numbers.IsGreater(bitcoinAmount, nonDustBitcoinAmount) {
		err = b.addOutput(tx, bitcoinAmount, bitcoinAmount, params.Sender.Address)
		if err != nil {
			return result, err
		}
	}

	result.UnsignedRawTx = tx
	result.UsedBaseUTXOs = senderUTXOsResult.UsedUTXOs
	result.EstimatedFee = senderUTXOsResult.RoughEstimate

	return result, nil
}

// buildInscriptionTxPSBT returns serialised PSBT from unsigned inscription commitment transaction
// with indexes provided in Unknowns field defining indexes of inputs with different types.
func (b *TxBuilder) buildInscriptionTxPSBT(params BuildInscriptionTxPSBTParams) ([]byte, error) {
	p, err := psbt.NewFromUnsignedTx(params.UnsignedRawTx)
	if err != nil {
		return nil, err
	}

	senderInputBuilder, err := NewPSBTInputBuilder(params.SenderPubKey, params.SenderAddress, b.networkParams)
	if err != nil {
		return nil, err
	}

	senderIndexes := make([]byte, len(params.UsedBaseUTXOs))
	for i, utxo := range params.UsedBaseUTXOs {
		senderInputBuilder.PrepareInput(&(p.Inputs[i]))
		p.Inputs[i].WitnessUtxo = wire.NewTxOut(utxo.Amount.Int64(), utxo.Script)
		p.Inputs[i].SighashType = signHashType
		senderIndexes[i] = byte(i)
	}

	p.Unknowns = append(p.Unknowns, &psbt.Unknown{Key: senderInputBuilder.InputsHelpingKey(false).Bytes(), Value: senderIndexes})

	w := bytes.NewBuffer(nil)
	err = p.Serialize(w)
	if err != nil {
		return nil, err
	}

	return w.Bytes(), nil
}

// BuildRuneEtchTx constructs inscription reveal - etch transaction in PSBT
// format with inputs indexes assigned in unknown fields. Transaction fee will be
// charged from inscription commitment utxo, if there won't be enough, the additional
// payment data will be used to cover transaction fee. Returns serialized
// PSBT transaction with used base outputs, estimated fee in satoshi, and error if any.
func (b *TxBuilder) BuildRuneEtchTx(params BaseRuneEtchTxParams) (result BuildRuneEtchTxPSBTResult, _ error) {
	buildBaseTransferRuneTxResult, err := b.buildRuneEtchTx(params)
	if err != nil {
		return result, err
	}

	result.UsedAdditionalBaseUTXOs = buildBaseTransferRuneTxResult.UsedAdditionalBaseUTXOs
	result.EstimatedFee = buildBaseTransferRuneTxResult.EstimatedFee

	inscriptionAddress, err := params.Inscription.IntoAddress(params.InscriptionReveal.PubKey, b.networkParams)
	if err != nil {
		return result, err
	}

	buildRuneEtchTxPSBTParams := BuildRuneEtchTxPSBTParams{
		BaseRuneEtchTxResult:   buildBaseTransferRuneTxResult,
		InscriptionBasePubKey:  params.InscriptionReveal.PubKey,
		InscriptionBaseAddress: inscriptionAddress,
	}
	if params.AdditionalPayments != nil {
		buildRuneEtchTxPSBTParams.AdditionalPaymentsAddress = params.AdditionalPayments.Address
		buildRuneEtchTxPSBTParams.AdditionalPaymentsPubKey = params.AdditionalPayments.PubKey
	}

	result.SerializedPSBT, err = b.buildRuneEtchTxPSBT(buildRuneEtchTxPSBTParams)
	if err != nil {
		return result, err
	}

	return result, nil
}

// buildRuneEtchTx constructs base rune etching transaction.
// Returns transaction, list of used base utxos pointers, estimated fee,
// and error if any.
//
//	Tx struct
//	inputs:
//	┌─────────┬──────────────┬────────────────────────────────────────┐
//	│  index  │     type     │             description                │
//	├=========┼==============┼========================================┤
//	│       0 │ base output  │ mandatory, inscription commitment utxo │
//	│         │              │ to reveal.                             │
//	├─────────┼──────────────┼────────────────────────────────────────┤
//	│   1 - n │ base inputs  │ additional payment utxos to cover      │
//	│         │              │ transaction fee in case deposited to   │
//	│         │              │ inscription address btc amount not     │
//	│         │              │ enough to do that.                     │
//	└─────────┴──────────────┴────────────────────────────────────────┘
//
//	outputs:
//	┌─────────┬──────────────┬────────────────────────────────────────┐
//	│  index  │     type     │             description                │
//	├=========┼==============┼========================================┤
//	│       0 │ runestone    │ rune protocol main output              │
//	├─────────┼──────────────┼────────────────────────────────────────┤
//	│ 1 - psf │ rune output  │ mandatory, output to link runes        │
//	│         │              │ to recipient.                          │
//	│         │              │       (psf - premine slpitting factor) │
//	├─────────┼──────────────┼────────────────────────────────────────┤
//	│ psf + 1 │ base output  │ outputs to change bitcoin amount.      │
//	│         │              │ 99% mandatory, if any non-dust left.   │
//	└─────────┴──────────────┴────────────────────────────────────────┘
func (b *TxBuilder) buildRuneEtchTx(params BaseRuneEtchTxParams) (result BaseRuneEtchTxResult, err error) {
	if params.InscriptionReveal == nil {
		return result, errors.New("inscription reveal data is required")
	}
	if params.Inscription == nil {
		return result, errors.New("inscription data is required")
	}
	if params.Rune != nil && params.Rune.Premine != nil && numbers.IsPositive(params.Rune.Premine) &&
		params.PremineSplittingFactor > 1 && numbers.IsGreater(big.NewInt(int64(params.PremineSplittingFactor)), params.Rune.Premine) {
		return result, errors.New("premine splitting factor is grater then premine")
	}
	if len(params.InscriptionReveal.UTXOs) != 1 {
		return result, fmt.Errorf("invalid inscription utxo data len: %d, must be: 1", len(params.InscriptionReveal.UTXOs))
	}

	var (
		pointerValue           uint32 = 1
		inscriptionWitnessSize int
		prepareUTXOsResult     PrepareUTXOsResult
		runeOutputs            = 1
		totalOutputs           = 1
	)

	if params.Rune.Premine != nil && numbers.IsPositive(params.Rune.Premine) && params.PremineSplittingFactor > 1 {
		runeOutputs = int(params.PremineSplittingFactor)
	}

	totalOutputs += runeOutputs

	bitcoinAmount := new(big.Int).Set(params.InscriptionReveal.UTXOs[0].Amount)

	inscriptionWitnessSize, err = params.Inscription.VBytesSize()
	if err != nil {
		return result, err
	}

	etchTransactionFee := RoughEtchFeeEstimate(big.NewInt(int64(inscriptionWitnessSize)), params.SatoshiPerKVByte, runeOutputs)
	transferAmount := new(big.Int).Add(etchTransactionFee, new(big.Int).Mul(nonDustBitcoinAmount, big.NewInt(int64(runeOutputs))))
	if numbers.IsGreater(transferAmount, params.InscriptionReveal.UTXOs[0].Amount) {
		if params.AdditionalPayments == nil {
			return result, InsufficientNativeBalanceError.
				clarify(transferAmount, params.InscriptionReveal.UTXOs[0].Amount).
				setCauser(CauserSender)
		}

		prepareUTXOsResult, err = PrepareUTXOs(PrepareUTXOsParams{
			Utxos:            params.AdditionalPayments.UTXOs,
			Inputs:           1,
			Outputs:          0,
			TransferAmount:   new(big.Int).Sub(transferAmount, params.InscriptionReveal.UTXOs[0].Amount),
			SatoshiPerKVByte: params.SatoshiPerKVByte,
		})
		if err != nil {
			if errIns := new(InsufficientError); errors.As(err, &errIns) {
				return result, errIns.setCauser(CauserFeePayer)
			}

			return result, err
		}

		bitcoinAmount.Add(bitcoinAmount, prepareUTXOsResult.TotalAmount)
		etchTransactionFee.Add(etchTransactionFee, prepareUTXOsResult.RoughEstimate)
	}

	tx := wire.NewMsgTx(txVersion)
	for _, i := range append([]*bitcoin.UTXO{&params.InscriptionReveal.UTXOs[0]}, prepareUTXOsResult.UsedUTXOs...) {
		utxoHash, err := chainhash.NewHashFromStr(i.TxHash)
		if err != nil {
			return result, err
		}

		tx.AddTxIn(wire.NewTxIn(wire.NewOutPoint(utxoHash, i.Index), nil, nil))
	}

	// subtract fee.
	bitcoinAmount.Sub(bitcoinAmount, etchTransactionFee)

	// recipient runes output (#1 - psf).
	for i := 0; i < runeOutputs; i++ {
		err = b.addOutput(tx, nonDustBitcoinAmount, bitcoinAmount, params.RunesRecipientAddress)
		if err != nil {
			return result, err
		}
	}

	// change btc output (#psf+1).
	if numbers.IsPositive(bitcoinAmount) && numbers.IsGreater(bitcoinAmount, nonDustBitcoinAmount) {
		err = b.addOutput(tx, bitcoinAmount, bitcoinAmount, params.SatoshiChangeAddress)
		if err != nil {
			return result, err
		}

		totalOutputs++
	}

	runestone := &runes.Runestone{
		Etching: params.Rune,
		Pointer: &pointerValue,
	}
	if runeOutputs > 1 {
		runestone.Pointer = nil
		quo, rem := new(big.Int).QuoRem(params.Rune.Premine, big.NewInt(int64(runeOutputs)), new(big.Int))
		if !numbers.IsZero(rem) {
			runestone.Edicts = append(runestone.Edicts, runes.Edict{
				RuneID: runes.RuneID{},
				Amount: rem,
				Output: 1,
			})
		}
		runestone.Edicts = append(runestone.Edicts, runes.Edict{
			RuneID: runes.RuneID{},
			Amount: quo,
			Output: uint32(totalOutputs),
		})
	}

	runestoneData, err := runestone.IntoScript()
	if err != nil {
		return result, err
	}

	// runestone output (#0).
	tx.TxOut = append([]*wire.TxOut{wire.NewTxOut(0, runestoneData)}, tx.TxOut...)

	result.UnsignedRawTx = tx
	result.InscriptionReveal = params.Inscription
	result.InscriptionUTXO = params.InscriptionReveal.UTXOs[0]
	result.UsedAdditionalBaseUTXOs = prepareUTXOsResult.UsedUTXOs
	result.EstimatedFee = etchTransactionFee

	return result, nil
}

// buildRuneEtchTxPSBT returns serialised PSBT from unsigned inscription reveal - etch transaction
// with indexes provided in Unknowns field defining indexes of inputs with different types.
func (b *TxBuilder) buildRuneEtchTxPSBT(params BuildRuneEtchTxPSBTParams) ([]byte, error) {
	p, err := psbt.NewFromUnsignedTx(params.UnsignedRawTx)
	if err != nil {
		return nil, err
	}

	inscriptionInputBuilder, err := NewPSBTInputBuilder(params.InscriptionBasePubKey, params.InscriptionBaseAddress, b.networkParams)
	if err != nil {
		return nil, err
	}

	inscriptionScript, err := params.InscriptionReveal.IntoScriptForWitness(inscriptionInputBuilder.xOnlyPubKey)
	if err != nil {
		return nil, err
	}

	// TODO: Figure out.
	// tapLeaf := txscript.NewBaseTapLeaf(inscriptionScript)
	// tapScriptTree := txscript.AssembleTaprootScriptTree(tapLeaf)
	//
	// ctrlBlock := tapScriptTree.LeafMerkleProofs[0].ToControlBlock(inscriptionInputBuilder.publicKeyBtcec)
	// ctrlBlockBytes, err := ctrlBlock.ToBytes()
	// if err != nil {
	//	return nil, err
	// }.

	p.Inputs[0].SighashType = signHashType
	inscriptionInputBuilder.PrepareInput(&(p.Inputs[0]))
	p.Inputs[0].WitnessUtxo = wire.NewTxOut(params.InscriptionUTXO.Amount.Int64(), params.InscriptionUTXO.Script)
	p.Inputs[0].WitnessScript = inscriptionScript
	// TODO: Figure out.
	// p.Inputs[0].TaprootLeafScript = []*psbt.TaprootTapLeafScript{{
	//	ControlBlock: ctrlBlockBytes,
	//	Script:       tapLeaf.Script,
	//	LeafVersion:  tapLeaf.LeafVersion,
	// }}.

	if len(params.UsedAdditionalBaseUTXOs) != 0 {
		additionalPaymentInputBuilder, err := NewPSBTInputBuilder(params.AdditionalPaymentsPubKey, params.AdditionalPaymentsAddress, b.networkParams)
		if err != nil {
			return nil, err
		}

		indexes := make([]byte, len(params.UsedAdditionalBaseUTXOs))
		for i, utxo := range params.UsedAdditionalBaseUTXOs {
			additionalPaymentInputBuilder.PrepareInput(&(p.Inputs[i+1]))
			p.Inputs[i+1].WitnessUtxo = wire.NewTxOut(utxo.Amount.Int64(), utxo.Script)
			p.Inputs[i+1].SighashType = signHashType
			indexes[i] = byte(i + 1)
		}

		p.Unknowns = append(p.Unknowns, &psbt.Unknown{Key: additionalPaymentInputBuilder.InputsHelpingKey(true).Bytes(), Value: indexes})
	}

	w := bytes.NewBuffer(nil)
	err = p.Serialize(w)
	if err != nil {
		return nil, err
	}

	return w.Bytes(), nil
}

// PrepareUTXOs selects utxos to cover rough estimated fee.
// Returns used utxos, total satoshi amount of utxos, rough estimation in satoshi and error if any.
func PrepareUTXOs(params PrepareUTXOsParams) (result PrepareUTXOsResult, err error) {
	satFn := func(u *bitcoin.UTXO) *big.Int { return u.Amount }

	var fullParams = !(params.SatoshiPerKVByte == nil && params.Inputs == 0 && params.Outputs == 0)
	for i := 1; i <= len(params.Utxos); i++ {
		if fullParams {
			// INFO: vB * ( sat / kvB ) = 1000 sat.
			result.RoughEstimate = new(big.Int).Mul(RoughTxSizeEstimate(i+params.Inputs, params.Outputs),
				params.SatoshiPerKVByte)
			result.RoughEstimate.Div(result.RoughEstimate, big.NewInt(1000)) // sat.

			result.UsedUTXOs, result.TotalAmount, err = SelectUTXO(params.Utxos, satFn,
				new(big.Int).Add(result.RoughEstimate, params.TransferAmount), i, InsufficientNativeBalanceError)
		} else {
			result.UsedUTXOs, result.TotalAmount, err = SelectUTXO(params.Utxos, satFn,
				new(big.Int).Set(params.TransferAmount), i, InsufficientNativeBalanceError)
		}
		if err != nil {
			if errors.As(err, new(*InsufficientError)) && i != len(params.Utxos) {
				continue
			}

			return result, err
		}

		return result, nil
	}

	// INFO: vB * ( sat / kvB ) = 1000 sat.
	result.RoughEstimate = new(big.Int).Mul(RoughTxSizeEstimate(1+params.Inputs, params.Outputs),
		params.SatoshiPerKVByte)
	result.RoughEstimate.Div(result.RoughEstimate, big.NewInt(1000)) // sat.
	need := new(big.Int).Add(result.RoughEstimate, params.TransferAmount)

	return result, InsufficientNativeBalanceError.clarify(need, big.NewInt(0))
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
		usedUTXOs, totalAmount, err = SelectUTXO(utxos, runeFn, transferAmount, i, InsufficientRuneBalanceError)
		if err != nil {
			if errors.As(err, new(*InsufficientError)) && i != len(utxos) {
				continue
			}

			return nil, nil, err
		}

		return usedUTXOs, totalAmount, nil
	}

	return nil, nil, InsufficientRuneBalanceError.clarify(transferAmount, big.NewInt(0))
}

// RoughTxSizeEstimate returns Tx rough estimated size in vBytes.
// TODO: increase precision.
func RoughTxSizeEstimate(inputs, outputs int) *big.Int {
	size := new(big.Int).Set(headerSizeVBytes)
	size.Add(size, new(big.Int).Mul(inputSizeVBytes, big.NewInt(int64(inputs))))
	size.Add(size, new(big.Int).Mul(outputSizeVBytes, big.NewInt(int64(outputs))))

	return size
}

// RoughEtchFeeEstimate returns etch transaction rough estimate in satoshi.
// TODO: increase precision.
func RoughEtchFeeEstimate(inscriptionWitnessSize, satoshiPerKVByte *big.Int, premineSplittingFactor int) (etchTransactionFee *big.Int) {
	// INFO:
	// header: static value [vB]
	// inputs: inscription witness data + raw inscription input size [vB]
	// outputs: runes protocol, runes recipient * premine splitting factor, btc change [vB]
	// (header + inputs + outputs) * fee rate / 1000 = tx fee in satoshi
	// [vB] * 1000 [sat/vB] / 1000 = sat.
	//
	// estimate runes protocol as maximum possible (3 * simple output ~ 80-90 vB).
	etchTransactionFee = new(big.Int).Add(inscriptionInputSizeVBytes, inscriptionWitnessSize)      // inputs [vB].
	etchTransactionFee.Add(etchTransactionFee, RoughTxSizeEstimate(0, 2+2+premineSplittingFactor)) // outputs + header [vB].
	etchTransactionFee.Mul(etchTransactionFee, satoshiPerKVByte)                                   // multiply by fee rate [vB * 1000(sat/vB)].
	etchTransactionFee.Div(etchTransactionFee, big.NewInt(1000))                                   // reduce kilo value [sat].

	return etchTransactionFee
}

// SelectUTXO is a partly greedy selection algorithm for UTXOs with 'requiredUTXOs' parameter.
// Returns list of selected by algorithm UTXOs with total amount, counted by passed amount function.
func SelectUTXO(utxos []bitcoin.UTXO, amountFn func(*bitcoin.UTXO) *big.Int, minAmount *big.Int, requiredUTXOs int,
	insufficientBalanceError *InsufficientError) (usedUTXOs []*bitcoin.UTXO, totalAmount *big.Int, _ error) {
	if len(utxos) == 0 || len(utxos) < requiredUTXOs {
		return nil, nil, ErrInvalidUTXOAmount
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
			return nil, nil, ErrInvalidUTXOAmount
		}

		usedIdxs = append(usedIdxs, idx)
		totalAmount.Add(totalAmount, amountFn(&utxos[idx]))
		usedUTXOs = append(usedUTXOs, &utxos[idx])
	}

	if numbers.IsGreater(minAmount, totalAmount) {
		return nil, nil, insufficientBalanceError.clarify(minAmount, totalAmount)
	}

	return usedUTXOs, totalAmount, nil
}

// addOutput adds output to transaction, subtracts amount from unallocated amount.
func (b *TxBuilder) addOutput(tx *wire.MsgTx, amount, unallocatedAmount *big.Int, address string) error {
	if numbers.IsLess(unallocatedAmount, amount) {
		return fmt.Errorf("the rest of the unallocated btc amount (%s) is less than the output allocating amount (%s)",
			unallocatedAmount.String(), amount.String())
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
