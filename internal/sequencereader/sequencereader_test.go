// Copyright (C) 2024 Creditor Corp. Group.
// See LICENSE for copying information.

package sequencereader_test

import (
	"math/big"
	"testing"

	"github.com/stretchr/testify/require"

	"blockchain/internal/sequencereader"
)

func TestSequenceReader(t *testing.T) {
	seq := []*big.Int{big.NewInt(1), big.NewInt(2), big.NewInt(3), big.NewInt(4)}
	t.Run("HasNext", func(t *testing.T) {
		sr := sequencereader.New(seq)
		require.True(t, sr.HasNext())

		_, _ = sr.Next()
		_, _ = sr.Next()
		_, _ = sr.Next()
		require.True(t, sr.HasNext())

		_, _ = sr.Next()
		require.False(t, sr.HasNext())

	})

	t.Run("Next", func(t *testing.T) {
		sr := sequencereader.New(seq)
		size := 0
		for _, tVal := range seq {
			val, err := sr.Next()
			require.NoError(t, err)
			require.Equal(t, tVal, val)
			size++
		}
		require.Equal(t, len(seq), size)

		_, err := sr.Next()
		require.Error(t, err)
	})

	t.Run("Len", func(t *testing.T) {
		sr := sequencereader.New(seq)
		require.True(t, sr.HasNext())
		require.Equal(t, len(seq), sr.Len())

		_, _ = sr.Next()
		require.True(t, sr.HasNext())
		require.Equal(t, len(seq)-1, sr.Len())

		_, _ = sr.Next()
		require.True(t, sr.HasNext())
		require.Equal(t, len(seq)-2, sr.Len())

		_, _ = sr.Next()
		require.True(t, sr.HasNext())
		require.Equal(t, len(seq)-3, sr.Len())

		_, _ = sr.Next()
		require.False(t, sr.HasNext())
		require.Equal(t, 0, sr.Len())

	})

	t.Run("loop", func(t *testing.T) {
		sr := sequencereader.New(seq)
		idx := 0
		for sr.HasNext() {
			val, err := sr.Next()
			require.NoError(t, err)
			require.Equal(t, seq[idx], val)
			idx++
		}
		require.Equal(t, len(seq), idx)

		_, err := sr.Next()
		require.Error(t, err)
	})

	t.Run("SequenceReader for string type", func(t *testing.T) {
		strSeq := []string{"a", "ab", "abc", "abcd"}
		sr := sequencereader.New[string](strSeq)
		require.EqualValues(t, 4, sr.Len())
		for i := 0; sr.HasNext(); i++ {
			val, err := sr.Next()
			require.NoError(t, err)
			require.EqualValues(t, strSeq[i], val)
		}
		_, err := sr.Next()
		require.Error(t, err)
	})
}
