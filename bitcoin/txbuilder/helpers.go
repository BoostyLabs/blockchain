// Copyright (C) 2024 Creditor Corp. Group.
// See LICENSE for copying information.

package txbuilder

import (
	"bytes"
	"errors"

	"github.com/btcsuite/btcd/btcutil/psbt"
)

// ExtractAddressTypeInputIndexesFromPSBT returns map with address types and indexes to sign.
func ExtractAddressTypeInputIndexesFromPSBT(data []byte) (map[InputsHelpingKey][]int, error) {
	var result = make(map[InputsHelpingKey][]int, 2)
	p, err := psbt.NewFromRawBytes(bytes.NewBuffer(data), false)
	if err != nil {
		return nil, err
	}

	for _, unknown := range p.Unknowns {
		if len(unknown.Key) != 1 {
			continue
		}

		var key InputsHelpingKey
		switch unknown.Key[0] {
		case TaprootInputsHelpingKey.Byte():
			key = TaprootInputsHelpingKey
		case PaymentInputsHelpingKey.Byte():
			key = PaymentInputsHelpingKey
		default:
			return nil, errors.New("unknown input key")
		}

		result[key] = make([]int, len(unknown.Value))
		for idx, val := range unknown.Value {
			result[key][idx] = int(val)
		}
	}

	return result, nil
}
