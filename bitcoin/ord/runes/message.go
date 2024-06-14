// Copyright (C) 2024 Creditor Corp. Group.
// See LICENSE for copying information.

package runes

import (
	"math/big"
	"slices"

	"blockchain/internal/sequencereader"
)

// fieldType defines helping struct for ordering map.
type fieldType struct {
	Tag  Tag
	Nums []*big.Int
}

// Message defines helping struct for serialising and deserializing Runestone.
type Message struct {
	Edicts []Edict
	Fields map[Tag][]*big.Int
}

// ParseMessage parses Message from integer sequence.
func ParseMessage(sr *sequencereader.SequenceReader[*big.Int]) (*Message, error) {
	message := &Message{
		Fields: make(map[Tag][]*big.Int),
	}

	for sr.HasNext() {
		var (
			err          error
			tagBigInt, _ = sr.Next() // skip error due to loop condition check.
			tag          = Tag(tagBigInt.Uint64())
		)
		if TagBody == tag {
			message.Edicts, err = ParseEdictsFromIntSeq(sr)
			if err != nil {
				return nil, err
			}

			break
		}

		value, err := sr.Next()
		if err != nil {
			return nil, ErrTruncated
		}

		message.Fields[tag] = append(message.Fields[tag], value)
	}

	if len(message.Fields) == 0 {
		message.Fields = nil
	}

	return message, nil
}

// ToIntSeq returns Message as sequence on integers.
func (message *Message) ToIntSeq() []*big.Int {
	ordered := make([]fieldType, 0, len(message.Fields))
	for tag, ints := range message.Fields {
		ordered = append(ordered, fieldType{tag, ints})
	}

	// sort ordered for immutability.
	slices.SortFunc(ordered, func(a, b fieldType) int {
		return int(a.Tag) - int(b.Tag)
	})

	// key/value -> 2 ints + 1 extra for mint 2nd value + edicts*4 for
	// edicts values - 1 because edicts key value is group of 4 ints.
	sequence := make([]*big.Int, 0, len(message.Fields)*2+len(message.Edicts)*4)
	for _, field := range ordered {
		for _, val := range field.Nums {
			sequence = append(sequence, field.Tag.BigInt(), val)
		}
	}

	if message.Edicts != nil {
		sequence = append(sequence, TagBody.BigInt())
		sequence = append(sequence, EdictsToIntSeq(message.Edicts)...)
	}

	return sequence
}
