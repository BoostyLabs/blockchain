// Copyright (C) 2025 Creditor Corp. Group.
// See LICENSE for copying information.

package txbuilder

import (
	"errors"
	"math/big"

	"github.com/btcsuite/btcd/btcutil"
	"github.com/btcsuite/btcd/chaincfg"
)

var (
	// headerSizeVBytes defined rough tx header size in vBytes.
	headerSizeVBytes = big.NewInt(11)
)

// PaymentDataFees holds additional info to build more accurate fee estimation.
type PaymentDataFees struct {
	InputSizeVBytes   *big.Int // defines input size in virtual bytes.
	WitnessSizeVBytes *big.Int // defines witness size in virtual bytes.
	OutputSizeVBytes  *big.Int // defines output size in virtual bytes.
}

// NewDefaultPaymentDataFees builds default estimation data for provided address.
// NOTE: Default estimation data assumes the next type to script mapping:
//
//	P2PKH  - single signature
//	P2SH   - 2-of-3 multisig
//	P2WPKH - single signature
//	P2WSH  - 2-of-3 multisig
//	P2TR   - key-spend path (single signature)
//
// INFO: Sizes were taken from this calculator: https://bitcoinops.org/en/tools/calc-size/.
func NewDefaultPaymentDataFees(address btcutil.Address) (*PaymentDataFees, error) {
	var inputSizeVByte, witnessSizeVBytes, outputSizeVBytes int64
	switch address.(type) {
	case *btcutil.AddressPubKey:
		return nil, errors.New("unsupported address type: too old (P2PK)")
	case *btcutil.AddressPubKeyHash:
		inputSizeVByte, witnessSizeVBytes, outputSizeVBytes = 41, 107, 34
	case *btcutil.AddressScriptHash:
		inputSizeVByte, witnessSizeVBytes, outputSizeVBytes = 43, 254, 32
	case *btcutil.AddressWitnessPubKeyHash:
		inputSizeVByte, witnessSizeVBytes, outputSizeVBytes = 41, 27, 31
	case *btcutil.AddressWitnessScriptHash:
		inputSizeVByte, witnessSizeVBytes, outputSizeVBytes = 41, 64, 43
	case *btcutil.AddressTaproot:
		inputSizeVByte, witnessSizeVBytes, outputSizeVBytes = 41, 17, 43
	default:
		return nil, errors.New("unsupported address type")
	}

	return &PaymentDataFees{
		InputSizeVBytes:   big.NewInt(inputSizeVByte),
		WitnessSizeVBytes: big.NewInt(witnessSizeVBytes),
		OutputSizeVBytes:  big.NewInt(outputSizeVBytes),
	}, nil
}

// Filled returns true is struct was properly initialized and all fields are filled.
func (pdf *PaymentDataFees) Filled() bool {
	return pdf != nil && pdf.InputSizeVBytes != nil && pdf.WitnessSizeVBytes != nil && pdf.OutputSizeVBytes != nil
}

// RoughTxSize holds data to make rough estimation of tx in vBytes.
type RoughTxSize struct {
	IncludeHeader bool                 // to include or not tx header vBytes size into a count.
	Elements      []*EstimationElement // estimation elements.
	ExtraExpenses *big.Int             // defines extra expenses (in vBytes) from previous estimation or for another logic needs. optional.
}

// EstimationElement holds data needed to estimate fee part for PaymentData group.
type EstimationElement struct {
	*PaymentDataFees
	InputsNumber  int // number of inputs of PaymentData address.
	OutputsNumber int // number of outputs of PaymentData address.
}

// Estimate returns rough estimation of tx in vBytes.
func (params *RoughTxSize) Estimate() *big.Int {
	size := big.NewInt(0)
	if params.IncludeHeader {
		size.Add(size, headerSizeVBytes)
	}

	for _, element := range params.Elements {
		signedInputSize := new(big.Int).Add(element.InputSizeVBytes, element.WitnessSizeVBytes)
		inputsFee := new(big.Int).Mul(signedInputSize, big.NewInt(int64(element.InputsNumber)))
		size.Add(size, inputsFee)

		outputsFee := new(big.Int).Mul(element.OutputSizeVBytes, big.NewInt(int64(element.OutputsNumber)))
		size.Add(size, outputsFee)
	}

	if params.ExtraExpenses != nil {
		size.Add(size, params.ExtraExpenses)
	}

	return size
}

// NewEstimationElementFromStringAddress is a constructor for EstimationElement that additionally parses address from string.
func NewEstimationElementFromStringAddress(address string, inputs, outputs int, networkParams *chaincfg.Params) (*EstimationElement, error) {
	addr, err := btcutil.DecodeAddress(address, networkParams)
	if err != nil {
		return nil, err
	}

	addressDataFees, err := NewDefaultPaymentDataFees(addr)
	if err != nil {
		return nil, err
	}

	if inputs < 0 || outputs < 0 {
		return nil, errors.New("inputs and outputs must be >= 0")
	}

	return &EstimationElement{
		PaymentDataFees: addressDataFees,
		InputsNumber:    inputs,
		OutputsNumber:   outputs,
	}, nil
}

// CalculateTxFee calculates tx network fee from tx size in vBytes and sat/kVb price.
func CalculateTxFee(txSizeVBytes, satoshiPerKVByte *big.Int) (fee *big.Int) {
	fee = new(big.Int).Mul(txSizeVBytes, satoshiPerKVByte)
	fee.Quo(fee, big.NewInt(1000))

	return fee
}
