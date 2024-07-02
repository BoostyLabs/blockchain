// Copyright (C) 2024 Creditor Corp. Group.
// See LICENSE for copying information.

package txbuilder_test

import (
	"encoding/base64"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/BoostyLabs/blockchain/bitcoin/txbuilder"
)

func TestExtractAddressTypeInputIndexesFromPSBT(t *testing.T) {
	tests := []struct {
		psbt     string
		expected map[txbuilder.InputsHelpingKey][]int
	}{
		{
			"cHNidP8BAPICAAAAAkZXKFP369ZOSUKg4F+781Lp64ePDidu1UPsQxzWUorXBAAAAAD/////RlcoU/fr1k5JQqDgX7vzUunrh48OJ27VQ+xDHNZSitcCAAAAAP////8EAAAAAAAAAAAMal0JFgIA4ghNnRoBIgIAAAAAAAAiUSAu6vu/kq8tH14IZsvr1he5lWJfN2J6Y4yQTd0mhUTDECICAAAAAAAAIlEgyTbXlQM2cHAjy50YCG0+l5N+McVx/87HcNiEC44gWmQb8AwAAAAAACJRIMk215UDNnBwI8udGAhtPpeTfjHFcf/Ox3DYhAuOIFpkAAAAAAEQAQABIAEBAAEBKiICAAAAAAAAIV9iaXRjb2luX3RyYW5zYWN0aW9uX3J1bmVfc2NyaXB0XwEDBAEAAAABFyAp+mEcNhNVsILuWT/rNoAJqpxr0e02yZg+3NET+42jPwABASVQ+AwAAAAAABxfYml0Y29pbl90cmFuc2FjdGlvbl9zY3JpcHRfAQMEAQAAAAEEFgAU8+s8RTsBFB5gK+stEzX2vlB7gTgAAAAAAA==",
			map[txbuilder.InputsHelpingKey][]int{txbuilder.TaprootInputsHelpingKey: {0}, txbuilder.PaymentInputsHelpingKey: {1}},
		},
		{
			"cHNidP8BAH4CAAAAAUZXKFP369ZOSUKg4F+781Lp64ePDidu1UPsQxzWUorXAgAAAAD/////AjxzAAAAAAAAIlEgLur7v5KvLR9eCGbL69YXuZViXzdiemOMkE3dJoVEwxDvgQwAAAAAABepFKpYjpRh5/yszRC1NNtHIt1yMSLBhwAAAAABIAEAAAEBJVD4DAAAAAAAHF9iaXRjb2luX3RyYW5zYWN0aW9uX3NjcmlwdF8BAwQBAAAAAQQWABTz6zxFOwEUHmAr6y0TNfa+UHuBOAAAAA==",
			map[txbuilder.InputsHelpingKey][]int{txbuilder.PaymentInputsHelpingKey: {0}},
		},
	}
	for _, test := range tests {
		data, err := base64.StdEncoding.DecodeString(test.psbt)
		require.NoError(t, err)

		result, err := txbuilder.ExtractAddressTypeInputIndexesFromPSBT(data)
		require.NoError(t, err)
		require.EqualValues(t, test.expected, result)
	}
}
