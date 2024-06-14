package reverse_test

import (
	"bytes"
	"testing"
	"unicode/utf8"

	"blockchain/internal/reverse"
)

func FuzzReverse(f *testing.F) {
	f.Add([]byte("some_data_here"))

	f.Fuzz(func(t *testing.T, orig []byte) {
		rev := reverse.Bytes(orig)
		doubleRev := reverse.Bytes(rev)

		if !bytes.Equal(orig, doubleRev) {
			t.Errorf("Before: %q, after: %q", orig, doubleRev)
		}
		if utf8.Valid(orig) && !utf8.Valid(rev) {
			t.Errorf("Reverse produced invalid UTF-8 string %q", rev)
		}
	})
}
