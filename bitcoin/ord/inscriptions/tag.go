// Copyright (C) 2024 Creditor Corp. Group.
// See LICENSE for copying information.

package inscriptions

import (
	"fmt"

	"github.com/btcsuite/btcd/txscript"
)

// Tag defines special tag for distinguishing inscription field type.
type Tag byte

const (
	// TagPointer defines pointer tag in the inscription protocol.
	// Points on the sat at the given position in the outputs for the inscription to be made.
	TagPointer Tag = 2
	// TagUnbound defines unbound tag in the inscription protocol.
	TagUnbound Tag = 66
	// TagContentType defines content-type tag in the inscription protocol.
	// Defines content-type of the inscription content. The value is the MIME type of the body.
	TagContentType Tag = 1
	// TagParent defines parent tag in the inscription protocol.
	// For creating child inscriptions. Points to parent inscription.
	TagParent Tag = 3
	// TagMetadata defines metadata tag in the inscription protocol.
	// Additional metadata in CBOR encoding. Not more than 520 bytes of the data for 1 data push.
	// Concatenate all Metadata-Tag pushes before decoding.
	TagMetadata Tag = 5
	// TagMetaprotocol defines meta-protocol tag in the inscription protocol.
	// The value is the metaprotocol identifier.
	TagMetaprotocol Tag = 7
	// TagContentEncoding defines content-encoding tag in the inscription protocol.
	// The value is the encoding of the body.
	TagContentEncoding Tag = 9
	// TagDelegate defines delegate tag in the inscription protocol.
	// This can be used to cheaply create copies of an inscription.
	TagDelegate Tag = 11
	// TagRune defines Rune tag in the inscription protocol.
	// For runes protocol usage.
	TagRune Tag = 13
	// TagNote defines Note tag in the inscription protocol.
	TagNote Tag = 15
	// TagNop defines Nop tag in the inscription protocol.
	TagNop Tag = 255
)

// IntoDataPush returns Tag as bytes array with OP_PUSH command.
func (t Tag) IntoDataPush() []byte {
	return []byte{txscript.OP_DATA_1, byte(t)}
}

// HexString returns Tag as hexadecimal string with leading zero if needed.
func (t Tag) HexString() string {
	return fmt.Sprintf("%02x", byte(t))
}
