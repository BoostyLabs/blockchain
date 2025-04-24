package utils_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/BoostyLabs/blockchain/bitcoin/utils"
)

func TestCompactSize(t *testing.T) {
	data := make([]byte, 1000)
	tests := []struct {
		data []byte
		size int
	}{
		{data: data[:2], size: 2 + 1},
		{data: data[:252], size: 252 + 1},
		{data: data[:253], size: 253 + 3},
		{data: data[:515], size: 515 + 3},
		{data: data[:1000], size: 1000 + 3},
	}
	for _, test := range tests {
		require.EqualValues(t, test.size, utils.CompactSize(test.data))
	}
}
