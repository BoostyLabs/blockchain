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
type BaseBTCTransferParams struct {
	BaseUTXOs                 []bitcoin.UTXO // must be sorted by btc amount desc.
	TransferSatoshiAmount     *big.Int       // amount to transfer in satoshi.
	SatoshiPerKVByte          *big.Int       // fee rate in satoshi per kilo virtual byte.
	SatoshiCommissionAmount   *big.Int       // additional commission in satoshi to be charged from user.
	RecipientAddress          string         // recipient btc address.
	CommissionReceiverAddress string         // recipient commission address.
	SenderAddress             string         // sender address.
	SenderPubKey              string         // sender public key.
}

// BaseBTCTransferResult describes result of buildBaseTransferBTCTx method.
type BaseBTCTransferResult struct {
	UnsignedRawTx *wire.MsgTx     // unsigned btc transfer transaction.
	UsedBaseUTXOs []*bitcoin.UTXO // used bitcoin utxos in transaction.
	EstimatedFee  *big.Int        // estimated transaction fee in Satoshi.
}

// BuildBTCTransferTxResult describes result of BuildBTCTransferTx method.
type BuildBTCTransferTxResult struct {
	SerializedPSBT []byte          // serialised unsigned rune transfer transaction in PSBT format.
	UsedBaseUTXOs  []*bitcoin.UTXO // used bitcoin utxos in transaction.
	EstimatedFee   *big.Int        // estimated transaction fee in Satoshi.
}

// BuildBTCTransferPSBTParams describes data needed to convert unsigned btc transfer transaction
// to partly signed bitcoin transaction (PSBT).
type BuildBTCTransferPSBTParams struct {
	BaseBTCTransferResult
	SenderAddress string
	SenderPubKey  string
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

	baseUTXOs, bitcoinAmount, fee, err := PrepareUTXOs(params.BaseUTXOs, len(runeUTXOs), outputs,
		satTransferAmount, params.SatoshiPerKVByte)
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
		bitcoinAmount.Add(bitcoinAmount, i.Amount)
	}
	for _, i := range baseUTXOs {
		utxoHash, err := chainhash.NewHashFromStr(i.TxHash)
		if err != nil {
			return result, err
		}

		tx.AddTxIn(wire.NewTxIn(wire.NewOutPoint(utxoHash, i.Index), nil, nil))
	}

	// subtract fee.
	bitcoinAmount.Sub(bitcoinAmount, fee)

	// runestone output (#0).
	tx.AddTxOut(wire.NewTxOut(0, runestoneData))

	// recipient runes output (#1).
	err = b.addOutput(tx, nonDustBitcoinAmount, bitcoinAmount, params.RecipientTaprootAddress)
	if err != nil {
		return result, err
	}

	// change runes output (#2).
	if runestone.Pointer != nil {
		if params.SatoshiCommissionAmount == nil {
			// cover return runes output value for estimation fee for user.
			fee.Add(fee, nonDustBitcoinAmount)
		}

		err = b.addOutput(tx, nonDustBitcoinAmount, bitcoinAmount, params.SenderTaprootAddress)
		if err != nil {
			return result, err
		}
	}

	// service commission output (#3).
	if params.SatoshiCommissionAmount != nil && numbers.IsPositive(params.SatoshiCommissionAmount) {
		err = b.addOutput(tx, params.SatoshiCommissionAmount, bitcoinAmount, params.CommissionRecipientAddress)
		if err != nil {
			return result, err
		}
	}

	// change btc output (#4).
	if numbers.IsPositive(bitcoinAmount) {
		err = b.addOutput(tx, bitcoinAmount, bitcoinAmount, params.SenderPaymentAddress)
		if err != nil {
			return result, err
		}
	}

	result.UnsignedRawTx = tx
	result.UsedRuneUTXOs = runeUTXOs
	result.UsedBaseUTXOs = baseUTXOs
	result.EstimatedFee = fee

	return result, nil
}

// BuildRunesTransferPSBT returns serialised PSBT from unsigned rune transferring transaction
// with indexes provided in Unknowns field defining indexes of inputs with different types.
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
		baseUTXOIndexes[i] = byte(i)
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

	result.UsedBaseUTXOs = buildBaseTransferRuneTxResult.UsedBaseUTXOs
	result.EstimatedFee = buildBaseTransferRuneTxResult.EstimatedFee

	result.SerializedPSBT, err = b.BuildBTCTransferPSBT(BuildBTCTransferPSBTParams{
		BaseBTCTransferResult: buildBaseTransferRuneTxResult,
		SenderAddress:         params.SenderAddress,
		SenderPubKey:          params.SenderPubKey,
	})
	if err != nil {
		return result, err
	}

	return result, nil
}

// buildBaseTransferBTCTx constructs base btc transferring transaction.
// Returns transaction, list of used rune's utxos pointers,
// list of used base utxos pointers, estimated fee, and error if any.
//
//	Tx struct
//	inputs:
//	┌─────────┬──────────────┬────────────────────────────────────────┐
//	│  index  │     type     │             description                │
//	├=========┼==============┼========================================┤
//	│   0 - n │ rune inputs  │ utxos with bitcoin only, possibly many │
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
//	│       2 │ base output  │ outputs to change bitcoin amount.      │
//	│         │              │ 99% mandatory, if any btc left.        │
//	└─────────┴──────────────┴────────────────────────────────────────┘
func (b *TxBuilder) buildBaseTransferBTCTx(params BaseBTCTransferParams) (result BaseBTCTransferResult, _ error) {
	outputs := 2
	satTransferAmount := new(big.Int).Set(params.TransferSatoshiAmount)
	if params.SatoshiCommissionAmount != nil && numbers.IsPositive(params.SatoshiCommissionAmount) {
		outputs++
		satTransferAmount.Add(satTransferAmount, params.SatoshiCommissionAmount)
	}

	baseUTXOs, bitcoinAmount, fee, err := PrepareUTXOs(params.BaseUTXOs, 0, outputs, satTransferAmount, params.SatoshiPerKVByte)
	if err != nil {
		return result, err
	}

	tx := wire.NewMsgTx(txVersion)
	for _, i := range baseUTXOs {
		utxoHash, err := chainhash.NewHashFromStr(i.TxHash)
		if err != nil {
			return result, err
		}

		tx.AddTxIn(wire.NewTxIn(wire.NewOutPoint(utxoHash, i.Index), nil, nil))
	}

	// subtract fee.
	bitcoinAmount.Sub(bitcoinAmount, fee)

	// recipient btc output (#0).
	err = b.addOutput(tx, satTransferAmount, bitcoinAmount, params.RecipientAddress)
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

	// change btc output (#2).
	if numbers.IsPositive(bitcoinAmount) {
		err = b.addOutput(tx, bitcoinAmount, bitcoinAmount, params.SenderAddress)
		if err != nil {
			return result, err
		}
	}

	result.UnsignedRawTx = tx
	result.UsedBaseUTXOs = baseUTXOs
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

	publicKey, err := hex.DecodeString(params.SenderPubKey)
	if err != nil {
		return nil, err
	}

	addressType, err := btcutil.DecodeAddress(params.SenderAddress, b.networkParams)
	if err != nil {
		return nil, err
	}

	var (
		addrType    InputsHelpingKey
		pubKey      *btcec.PublicKey
		witness     *btcutil.AddressWitnessPubKeyHash
		witnessProg []byte
	)
	switch addressType.(type) {
	case *btcutil.AddressTaproot:
		addrType = TaprootInputsHelpingKey
	case *btcutil.AddressPubKeyHash, *btcutil.AddressPubKey, *btcutil.AddressScriptHash:
		addrType = PaymentInputsHelpingKey
		pubKey, err = btcec.ParsePubKey(publicKey)
		if err != nil {
			return nil, err
		}

		witness, err = btcutil.NewAddressWitnessPubKeyHash(btcutil.Hash160(pubKey.SerializeCompressed()), b.networkParams)
		if err != nil {
			return nil, err
		}

		witnessProg, err = txscript.PayToAddrScript(witness)
		if err != nil {
			return nil, err
		}
	default:
		return nil, btcutil.ErrUnknownAddressType
	}

	indexes := make([]byte, len(params.UsedBaseUTXOs))
	for i, utxo := range params.UsedBaseUTXOs {
		switch addrType {
		case TaprootInputsHelpingKey:
			p.Inputs[i].TaprootInternalKey = publicKey
		case PaymentInputsHelpingKey:
			p.Inputs[i].RedeemScript = witnessProg
		}
		p.Inputs[i].WitnessUtxo = wire.NewTxOut(utxo.Amount.Int64(), utxo.Script)
		p.Inputs[i].SighashType = signHashType
		indexes[i] = byte(i)
	}

	p.Unknowns = append(p.Unknowns, &psbt.Unknown{Key: addrType.Bytes(), Value: indexes})

	w := bytes.NewBuffer(nil)
	err = p.Serialize(w)
	if err != nil {
		return nil, err
	}

	return w.Bytes(), nil
}

// PrepareUTXOs selects utxos to cover rough estimated fee.
// Returns used utxos, total satoshi amount of utxos, rough estimation in satoshi and error if any.
func PrepareUTXOs(utxos []bitcoin.UTXO, inputs, outputs int, transferAmount, satoshiPerKVByte *big.Int) (usedUTXOs []*bitcoin.UTXO, totalAmount, roughEstimate *big.Int, err error) {
	satFn := func(u *bitcoin.UTXO) *big.Int { return u.Amount }

	for i := 1; i <= len(utxos); i++ {
		// vB * ( sat / kvB ) = 1000 sat.
		roughEstimate = new(big.Int).Mul(RoughTxSizeEstimate(i+inputs, outputs), satoshiPerKVByte)
		roughEstimate.Div(roughEstimate, big.NewInt(1000)) // sat.

		usedUTXOs, totalAmount, err = SelectUTXO(utxos, satFn, new(big.Int).Add(roughEstimate, transferAmount), i, bitcoin.ErrInsufficientNativeBalance)
		if err != nil {
			if errors.Is(err, bitcoin.ErrInsufficientNativeBalance) {
				continue
			}

			return nil, nil, nil, err
		}

		return usedUTXOs, totalAmount, roughEstimate, nil
	}

	return nil, nil, nil, bitcoin.ErrInsufficientNativeBalance
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
