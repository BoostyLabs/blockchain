// Copyright (C) 2024 Creditor Corp. Group.
// See LICENSE for copying information.

package inscriptions

import (
	"encoding/hex"
	"errors"
	"math/big"
	"strings"

	"github.com/btcsuite/btcd/txscript"

	"blockchain/bitcoin/runes"
	"blockchain/internal/reverse"
	"blockchain/internal/sequencereader"
)

// ErrMalformedInscription defines that inscription is malformed and failed to parse.
var ErrMalformedInscription = errors.New("inscription is malformed")

// ErrRepeatedFieldData defines that already filled field met while parsing.
var ErrRepeatedFieldData = errors.New("field already field")

// inscriptionOrdTag defines ord tag for inscription to disambiguate inscriptions from other uses of envelopes.
const inscriptionOrdTag string = "ord"

// inscriptionStartASM defines the start of the inscription script in disASM.
// OP_FALSE OP_IF OP_PUSH "ord" ...
const inscriptionStartDisASM string = "0 OP_IF 6f7264"

// inscriptionEndASM defines the end of the inscription script in disASM.
// ... OP_ENDIF.
const inscriptionEndDisASM string = "OP_ENDIF"

// maxBodyDataPushLen defines maximum size of the data push for bitcoin scripts.
const maxBodyDataPushLen int = 520

// Inscription describes inscription type of the inscription protocol,
// which inscribe sats with arbitrary content, creating bitcoin-native digital artifacts.
type Inscription struct {
	ID              ID
	Body            []byte
	ContentEncoding string
	ContentType     string
	Delegate        *ID
	Metadata        []byte
	Metaprotocol    []byte
	Parents         []*ID
	Pointer         *big.Int
	Rune            *runes.Rune
}

// IsPossibleInscriptionWitnessData returns true if witness data is possible to be parsed to inscription.
func IsPossibleInscriptionWitnessData(data []byte) bool {
	_, _, _, err := disasmWitnessDataWithBoundsIndexes(data)

	return err == nil
}

// disasmWitnessDataWithBoundsIndexes returns disassembled witness data with start and end indexes of inscription script.
func disasmWitnessDataWithBoundsIndexes(data []byte) (disasm string, start int, end int, err error) {
	disasm, err = txscript.DisasmString(data)
	if err != nil {
		return disasm, start, end, ErrMalformedInscription
	}

	start = strings.Index(disasm, inscriptionStartDisASM)
	end = strings.Index(disasm, inscriptionEndDisASM)
	if start == -1 || end == -1 || end <= start {
		return disasm, start, end, ErrMalformedInscription
	}

	return disasm, start, end, nil
}

// ParseInscriptionFromWitnessData parses witness data into Inscription.
func ParseInscriptionFromWitnessData(data []byte) (*Inscription, error) {
	disasm, start, end, err := disasmWitnessDataWithBoundsIndexes(data)
	if err != nil {
		return nil, err
	}

	sr := sequencereader.New[string](strings.Split(disasm[start:end+len(inscriptionEndDisASM)], " "))
	// At least OP_FALSE OP_IF OP_PUSH "ord" OP_ENDIF.
	if sr.Len() < 4 {
		return nil, ErrMalformedInscription
	}

	// Skip OP_FALSE OP_IF OP_PUSH "ord" due to previous checks (inscriptionStartDisASM).
	_, _ = sr.Next()
	_, _ = sr.Next()
	_, _ = sr.Next()

	inscription := new(Inscription)
	for sr.HasNext() {
		tag, _ := sr.Next() // skip error due to the loop condition check.
		if tag == "0" {     // OP_0, means that all next data pushes are body parts.
			err = inscription.fillBody(sr)
		} else {
			var value string
			value, err = sr.Next()
			if err != nil {
				return nil, ErrMalformedInscription
			}

			err = inscription.fillFieldByTag(tag, value)
		}
		if err != nil {
			return nil, err
		}
	}

	return inscription, nil
}

// fillBody fills Body field with body data pushes.
func (i *Inscription) fillBody(sr *sequencereader.SequenceReader[string]) (err error) {
	var payload string
	for sr.HasNext() {
		value, _ := sr.Next() // skip error due to the loop condition check.
		if value == inscriptionEndDisASM {
			break
		}

		payload += value
	}

	i.Body, err = hex.DecodeString(payload)
	if err != nil {
		return err
	}

	return nil
}

// fillFieldByTag fills Inscription fields by provided tag.
func (i *Inscription) fillFieldByTag(tag string, value string) (err error) {
	valueBytes, err := hex.DecodeString(value)
	if err != nil {
		return err
	}

	switch tag {
	case TagContentType.HexString():
		if len(i.ContentType) != 0 {
			return ErrRepeatedFieldData
		}

		i.ContentType = string(valueBytes)
	case TagPointer.HexString():
		if i.Pointer != nil {
			return ErrRepeatedFieldData
		}

		i.Pointer = new(big.Int).SetBytes(reverse.Bytes(valueBytes))
	case TagParent.HexString():
		id, err := NewIDFromString(string(valueBytes))
		if err != nil {
			return err
		}

		i.Parents = append(i.Parents, id)
	case TagMetadata.HexString():
		if len(i.Metadata) != 0 {
			return ErrRepeatedFieldData
		}

		i.Metadata = valueBytes
	case TagMetaprotocol.HexString():
		if len(i.Metaprotocol) != 0 {
			return ErrRepeatedFieldData
		}

		i.Metaprotocol = valueBytes
	case TagContentEncoding.HexString():
		if len(i.ContentEncoding) != 0 {
			return ErrRepeatedFieldData
		}

		i.ContentEncoding = string(valueBytes)
	case TagDelegate.HexString():
		if i.Delegate != nil {
			return ErrRepeatedFieldData
		}

		i.Delegate, err = NewIDFromString(string(valueBytes))
		if err != nil {
			return err
		}
	case TagRune.HexString():
		i.Rune, err = runes.NewRuneFromNumber(new(big.Int).SetBytes(reverse.Bytes(valueBytes)))
		if err != nil {
			return err
		}
	default:
		return ErrMalformedInscription
	}

	return nil
}

// IntoScript returns Inscription as a script.
func (i *Inscription) IntoScript() ([]byte, error) {
	scriptBuilder := txscript.NewScriptBuilder()

	// inscription protocol start.
	scriptBuilder.AddOp(txscript.OP_FALSE)
	scriptBuilder.AddOp(txscript.OP_IF)
	scriptBuilder.AddData([]byte(inscriptionOrdTag))

	// tags and content.
	if len(i.ContentType) != 0 {
		scriptBuilder.AddOps(TagContentType.IntoDataPush())
		scriptBuilder.AddData([]byte(i.ContentType))
	}

	if i.Pointer != nil {
		scriptBuilder.AddOps(TagPointer.IntoDataPush())
		scriptBuilder.AddData(reverse.Bytes(i.Pointer.Bytes()))
	}

	for _, parent := range i.Parents {
		scriptBuilder.AddOps(TagParent.IntoDataPush())
		scriptBuilder.AddData(parent.IntoDataPush())
	}

	if len(i.Metadata) != 0 {
		scriptBuilder.AddOps(TagMetadata.IntoDataPush())
		scriptBuilder.AddData(i.Metadata)
	}

	if len(i.Metaprotocol) != 0 {
		scriptBuilder.AddOps(TagMetaprotocol.IntoDataPush())
		scriptBuilder.AddData(i.Metaprotocol)
	}

	if len(i.ContentEncoding) != 0 {
		scriptBuilder.AddOps(TagContentEncoding.IntoDataPush())
		scriptBuilder.AddData([]byte(i.ContentEncoding))
	}

	if i.Delegate != nil {
		scriptBuilder.AddOps(TagDelegate.IntoDataPush())
		scriptBuilder.AddData(i.Delegate.IntoDataPush())
	}

	if i.Rune != nil {
		scriptBuilder.AddOps(TagRune.IntoDataPush())
		scriptBuilder.AddData(reverse.Bytes(i.Rune.Value().Bytes()))
	}

	if len(i.Body) != 0 {
		scriptBuilder.AddOp(txscript.OP_0)
		for _, bytes := range i.PrepareBody() {
			scriptBuilder.AddData(bytes)
		}
	}

	// inscription protocol end.
	scriptBuilder.AddOp(txscript.OP_ENDIF)

	return scriptBuilder.Script()
}

// PrepareBody returns Inscription body as array of bytes arrays with maxBodyDataPushLen size.
func (i *Inscription) PrepareBody() [][]byte {
	buffer := make([][]byte, 0, (len(i.Body)/maxBodyDataPushLen)+1)
	start := 0
	end := maxBodyDataPushLen
	for len(i.Body) >= end {
		buffer = append(buffer, i.Body[start:end])
		start = end
		end += maxBodyDataPushLen
	}

	if start < len(i.Body) {
		buffer = append(buffer, i.Body[start:])
	}

	return buffer
}

// IntoScriptForWitness returns Inscription as a script with pubKey verify at the beginning for witness data.
func (i *Inscription) IntoScriptForWitness(serializedPubKey []byte) ([]byte, error) {
	scriptBuilder := txscript.NewScriptBuilder()
	scriptBuilder.AddData(serializedPubKey)
	scriptBuilder.AddOp(txscript.OP_CHECKSIG)
	script, err := scriptBuilder.Script()
	if err != nil {
		return nil, err
	}

	inscription, err := i.IntoScript()
	if err != nil {
		return nil, err
	}

	return append(script, inscription...), nil
}
