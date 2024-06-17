// Copyright (C) 2024 Creditor Corp. Group.
// See LICENSE for copying information.

package txbuilder_test

import (
	"blockchain/bitcoin/txbuilder"

	"github.com/stretchr/testify/require"

	"testing"
)

func TestInputsHelpingKey(t *testing.T) {
	t.Run("InputsHelpingKeyFromBytes", func(t *testing.T) {
		tests := []struct {
			bytes []byte
			key   txbuilder.InputsHelpingKey
			err   error
		}{
			{[]byte{txbuilder.TaprootInputsHelpingKey.Byte()}, txbuilder.TaprootInputsHelpingKey, nil},
			{[]byte{txbuilder.PaymentInputsHelpingKey.Byte()}, txbuilder.PaymentInputsHelpingKey, nil},
			{[]byte{}, 0, txbuilder.ErrUnknownInputsHelpingKey},
			{[]byte{0x50}, 0, txbuilder.ErrUnknownInputsHelpingKey},
			{[]byte{0x01, 0x02}, 0, txbuilder.ErrUnknownInputsHelpingKey},
		}
		for _, test := range tests {
			key, err := txbuilder.InputsHelpingKeyFromBytes(test.bytes)
			require.Equal(t, test.err, err)
			require.Equal(t, test.key, key)
		}
	})

	t.Run("Byte&Bytes", func(t *testing.T) {
		tests := []struct {
			key   txbuilder.InputsHelpingKey
			byte  byte
			bytes []byte
		}{
			{txbuilder.TaprootInputsHelpingKey, byte(txbuilder.TaprootInputsHelpingKey), []byte{byte(txbuilder.TaprootInputsHelpingKey)}},
			{txbuilder.PaymentInputsHelpingKey, byte(txbuilder.PaymentInputsHelpingKey), []byte{byte(txbuilder.PaymentInputsHelpingKey)}},
		}
		for _, test := range tests {
			require.Equal(t, test.byte, test.key.Byte())
			require.Equal(t, test.bytes, test.key.Bytes())
		}
	})
}
