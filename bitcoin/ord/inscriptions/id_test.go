// Copyright (C) 2024 Creditor Corp. Group.
// See LICENSE for copying information.

package inscriptions_test

import (
	"blockchain/bitcoin/ord/inscriptions"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestID(t *testing.T) {
	t.Run("NewIDFromString", func(t *testing.T) {
		tests := []struct {
			value   string
			invalid bool
		}{
			{"521f8eccffa4c41a3a7728dd012ea5a4a02feed81f41159231251ecf1e5c79dai0", false},
			{"521f8eccffa4c41a3a7728ddi12ea5a4a02feed81f41159231251ecf1e5c79dai0", true},
			{"521f8eccffa4c41a3a7728dd012ea5a4a02feed81f411251ecf1e5c79dai0", true},
			{"521f8eccffa4c41a3a7728dd012ea5a4a02feed81f41159231251ecf1e5c79da", true},
		}
		for _, test := range tests {
			_, err := inscriptions.NewIDFromString(test.value)
			if test.invalid {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}
		}
	})

	t.Run("String", func(t *testing.T) {
		inscriptionID := "521f8eccffa4c41a3a7728dd012ea5a4a02feed81f41159231251ecf1e5c79dai0"
		id, err := inscriptions.NewIDFromString(inscriptionID)
		require.NoError(t, err)
		require.EqualValues(t, inscriptionID, id.String())
	})

	t.Run("IndexLETrailingZerosOmitted", func(t *testing.T) {
		tests := []struct {
			index uint32
			bytes []byte
		}{
			{0x0, []byte{}},
			{0x1, []byte{0x01}},
			{0xff, []byte{0xff}},
			{0x0100, []byte{0x00, 0x01}},
			{0x1000, []byte{0x00, 0x10}},
			{0xffff, []byte{0xff, 0xff}},
			{0x010000, []byte{0x00, 0x00, 0x01}},
			{0x100000, []byte{0x00, 0x00, 0x10}},
			{0xffffff, []byte{0xff, 0xff, 0xff}},
			{0x01000000, []byte{0x00, 0x00, 0x00, 0x01}},
			{0x10000000, []byte{0x00, 0x00, 0x00, 0x10}},
			{0xffffffff, []byte{0xff, 0xff, 0xff, 0xff}},
		}
		id := new(inscriptions.ID)
		for _, test := range tests {
			id.Index = test.index
			require.EqualValues(t, test.bytes, id.IndexLETrailingZerosOmitted())
		}
	})
}
