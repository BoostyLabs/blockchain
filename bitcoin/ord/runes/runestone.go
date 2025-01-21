// Copyright (C) 2024 Creditor Corp. Group.
// See LICENSE for copying information.

package runes

import (
	"bytes"
	"errors"
	"fmt"
	"math/big"

	"github.com/aviate-labs/leb128"
	"github.com/btcsuite/btcd/txscript"

	"github.com/BoostyLabs/blockchain/bitcoin/ord/runes/utils"
	"github.com/BoostyLabs/blockchain/internal/sequencereader"
)

const (
	// MaxDivisibility defines maximum divisibility for runes.
	MaxDivisibility byte = 38
	// MaxSpacers defines max value for spacers.
	MaxSpacers uint32 = 0b00000111_11111111_11111111_11111111
)

// ErrCenotaph defines invalid runestone produced malformed payload.
var ErrCenotaph = errors.New("cenotaph")

// ErrOverflow defines too large size of the payload.
var ErrOverflow = errors.New("payload overflow")

// ErrTruncated defines that payload is do not have required fields.
var ErrTruncated = errors.New("truncated payload")

// Runestone abstractly defines runestone fields.
type Runestone struct {
	Edicts  []Edict
	Etching *Etching
	Mint    *RuneID
	Pointer *uint32
}

// ParseRunestone parses Runestone from script code.
func ParseRunestone(script []byte) (runestone *Runestone, err error) {
	runestone = new(Runestone)
	payload, err := PreparePayload(script)
	if err != nil {
		return nil, err
	}

	sequence, err := PayloadIntoIntSequence(payload)
	if err != nil {
		return nil, err
	}

	return runestone, runestone.parse(sequencereader.New(sequence))
}

// parse parses runestone fields from integer sequence.
func (runestone *Runestone) parse(sr *sequencereader.SequenceReader[*big.Int]) error {
	message, err := ParseMessage(sr)
	if err != nil {
		return err
	}

	var etching, terms, turbo bool
	flags, ok := message.Fields[TagFlags]
	if ok {
		if len(flags) != 1 {
			return ErrCenotaph
		}

		etching = HasFlag(flags[0], FlagEtching)
		if etching {
			flags[0].Sub(flags[0], FlagEtching)
		}

		terms = HasFlag(flags[0], FlagTerms)
		if terms {
			flags[0].Sub(flags[0], FlagTerms)
		}

		turbo = HasFlag(flags[0], FlagTurbo)
		if turbo {
			flags[0].Sub(flags[0], FlagTurbo)
		}

		if flags[0].Sign() != 0 {
			return ErrCenotaph
		}

		delete(message.Fields, TagFlags)
	}

	if turbo {
		runestone.etching().Turbo = turbo
	}

	var failure bool
	for tag, ints := range message.Fields {
		switch tag {
		case TagMint:
			failure = !utils.IfLen(ints, 2).Then(func() error {
				runestone.mint().Block = ints[0].Uint64()
				runestone.mint().TxID = uint32(ints[1].Int64())
				return nil
			}).Ok()
		case TagPointer:
			failure = !utils.IfLen(ints, 1).Then(func() error {
				*runestone.pointer() = uint32(ints[0].Uint64())
				return nil
			}).Ok()
		case TagDivisibility:
			res := utils.IfLen(ints, 1).Then(func() error {
				divisibility := byte(ints[0].Uint64())
				runestone.etching().Divisibility = &divisibility

				if *runestone.Etching.Divisibility > MaxDivisibility {
					return errors.New("too large divisibility")
				}

				return nil
			})

			failure = !etching || !res.Ok()
			err = res.Error()
		case TagPremine:
			failure = !etching || !utils.IfLen(ints, 1).Then(func() error {
				runestone.etching().Premine = ints[0]
				return nil
			}).Ok()
		case TagRune:
			res := utils.IfLen(ints, 1).Then(func() error {
				rune, err := NewRuneFromNumber(ints[0])
				runestone.etching().Rune = rune
				return err
			})

			failure = !etching || !res.Ok()
			err = res.Error()
		case TagSpacers:
			res := utils.IfLen(ints, 1).Then(func() error {
				spacers := uint32(ints[0].Uint64())
				runestone.etching().Spacers = &spacers

				if *runestone.Etching.Spacers > MaxSpacers {
					return errors.New("too large spacers")
				}

				return nil
			})

			failure = !etching || !res.Ok()
			err = res.Error()
		case TagSymbol:
			failure = !etching || !utils.IfLen(ints, 1).Then(func() error {
				symbol := rune(ints[0].Int64())
				runestone.etching().Symbol = &symbol
				return nil
			}).Ok()
		case TagAmount:
			failure = !terms || !utils.IfLen(ints, 1).Then(func() error {
				runestone.terms().Amount = ints[0]
				return nil
			}).Ok()
		case TagCap:
			failure = !terms || !utils.IfLen(ints, 1).Then(func() error {
				runestone.terms().Cap = ints[0]
				return nil
			}).Ok()
		case TagHeightStart:
			failure = !terms || !utils.IfLen(ints, 1).Then(func() error {
				height := ints[0].Uint64()
				runestone.terms().HeightStart = &height
				return nil
			}).Ok()
		case TagHeightEnd:
			failure = !terms || !utils.IfLen(ints, 1).Then(func() error {
				height := ints[0].Uint64()
				runestone.terms().HeightEnd = &height
				return nil
			}).Ok()
		case TagOffsetStart:
			failure = !terms || !utils.IfLen(ints, 1).Then(func() error {
				offset := ints[0].Uint64()
				runestone.terms().OffsetStart = &offset
				return nil
			}).Ok()
		case TagOffsetEnd:
			failure = !terms || !utils.IfLen(ints, 1).Then(func() error {
				offset := ints[0].Uint64()
				runestone.terms().OffsetEnd = &offset
				return nil
			}).Ok()
		}

		if failure {
			return ErrCenotaph
		}

		if err != nil {
			return err
		}
	}

	runestone.Edicts = message.Edicts

	runestone.fillDefaultEtching()

	return nil
}

// IntoScript returns Runestone as script bytes.
func (runestone *Runestone) IntoScript() ([]byte, error) {
	payload, err := runestone.Serialize()
	if err != nil {
		return nil, err
	}

	payloadSize := len(payload)
	if payloadSize < txscript.OP_DATA_1 || payloadSize > txscript.OP_DATA_75 {
		return nil, errors.New("payload is out of PUSH_DATA bounds")
	}

	// OP_RETURN + OP_13 + OP_PUSH_<num> + payload.
	return append([]byte{txscript.OP_RETURN, txscript.OP_13, byte(payloadSize)}, payload...), nil
}

// Serialize returns Runestone as bytes array.
func (runestone *Runestone) Serialize() ([]byte, error) {
	message := Message{
		Edicts: runestone.Edicts,
		Fields: map[Tag][]*big.Int{},
	}
	flags := big.NewInt(0)
	if runestone.Etching != nil {
		flags = AddFlag(flags, FlagEtching)
		if runestone.Etching.Divisibility != nil {
			message.Fields[TagDivisibility] = []*big.Int{big.NewInt(int64(*runestone.Etching.Divisibility))}
		}
		if runestone.Etching.Premine != nil {
			message.Fields[TagPremine] = []*big.Int{runestone.Etching.Premine}
		}
		if runestone.Etching.Rune != nil {
			message.Fields[TagRune] = []*big.Int{runestone.Etching.Rune.Value()}
		}
		if runestone.Etching.Spacers != nil {
			message.Fields[TagSpacers] = []*big.Int{big.NewInt(int64(*runestone.Etching.Spacers))}
		}
		if runestone.Etching.Symbol != nil {
			message.Fields[TagSymbol] = []*big.Int{big.NewInt(int64(*runestone.Etching.Symbol))}
		}

		if runestone.Etching.Terms != nil {
			flags = AddFlag(flags, FlagTerms)
			if runestone.Etching.Terms.Cap != nil {
				message.Fields[TagCap] = []*big.Int{runestone.Etching.Terms.Cap}
			}
			if runestone.Etching.Terms.Amount != nil {
				message.Fields[TagAmount] = []*big.Int{runestone.Etching.Terms.Amount}
			}
			if runestone.Etching.Terms.HeightStart != nil {
				message.Fields[TagHeightStart] = []*big.Int{new(big.Int).SetUint64(*runestone.Etching.Terms.HeightStart)}
			}
			if runestone.Etching.Terms.HeightEnd != nil {
				message.Fields[TagHeightEnd] = []*big.Int{new(big.Int).SetUint64(*runestone.Etching.Terms.HeightEnd)}
			}
			if runestone.Etching.Terms.OffsetStart != nil {
				message.Fields[TagOffsetStart] = []*big.Int{new(big.Int).SetUint64(*runestone.Etching.Terms.OffsetStart)}
			}
			if runestone.Etching.Terms.OffsetEnd != nil {
				message.Fields[TagOffsetEnd] = []*big.Int{new(big.Int).SetUint64(*runestone.Etching.Terms.OffsetEnd)}
			}
		}

		if runestone.Etching.Turbo {
			flags = AddFlag(flags, FlagTurbo)
		}

		message.Fields[TagFlags] = []*big.Int{flags}
	}

	if runestone.Mint != nil {
		message.Fields[TagMint] = runestone.Mint.ToIntSeq()
	}

	if runestone.Pointer != nil {
		message.Fields[TagPointer] = []*big.Int{big.NewInt(int64(*runestone.Pointer))}
	}

	return IntSequenceIntoPayload(message.ToIntSeq())
}

// etching return Etching fieldType and initialize it if needed.
func (runestone *Runestone) etching() *Etching {
	if runestone.Etching == nil {
		runestone.Etching = new(Etching)
	}

	return runestone.Etching
}

// mint return Mint fieldType and initialize it if needed.
func (runestone *Runestone) mint() *RuneID {
	if runestone.Mint == nil {
		runestone.Mint = new(RuneID)
	}

	return runestone.Mint
}

// pointer return Pointer fieldType and initialize it if needed.
func (runestone *Runestone) pointer() *uint32 {
	if runestone.Pointer == nil {
		runestone.Pointer = new(uint32)
	}

	return runestone.Pointer
}

// terms return Etching.Terms fieldType and initialize it if needed.
func (runestone *Runestone) terms() *Terms {
	if runestone.etching().Terms == nil {
		runestone.etching().Terms = new(Terms)
	}

	return runestone.Etching.Terms
}

// fillDefaultEtching fills runestone etching fields to be valid for further processing.
func (runestone *Runestone) fillDefaultEtching() {
	if runestone.Etching != nil {
		if runestone.Etching.Premine == nil {
			runestone.Etching.Premine = big.NewInt(0)
		}
		if runestone.Etching.Divisibility == nil {
			runestone.Etching.Divisibility = new(byte)
		}
		if runestone.Etching.Spacers == nil {
			runestone.Etching.Spacers = new(uint32)
		}
		if runestone.Etching.Symbol == nil {
			runestone.Etching.Symbol = new(rune)
		}
	}
}

// IsValidEtching returns true if Etching is properly built.
func (runestone *Runestone) IsValidEtching(outputsNumber int) bool {
	// TODO: Add Terms validation.
	switch {
	case runestone.Etching == nil:
		return false
	case runestone.Pointer != nil && int(*runestone.Pointer) > outputsNumber:
		return false
	case runestone.Etching.Rune == nil:
		return false
	case runestone.Etching.Symbol == nil:
		return false
	case runestone.Etching.Divisibility == nil:
		return false
	case runestone.Etching.Spacers == nil:
		return false
	}

	return true
}

// IsValidMint returns true if Mint is properly built.
func (runestone *Runestone) IsValidMint(outputsNumber int) bool {
	switch {
	case runestone.Mint == nil:
		return false
	case runestone.Mint.Block == 0 && runestone.Mint.TxID != 0:
		return false
	case runestone.Pointer != nil && int(*runestone.Pointer) > outputsNumber:
		return false
	}

	return true
}

// IsValidEdicts returns true if Edicts are properly built.
func (runestone *Runestone) IsValidEdicts(outputsNumber int) bool {
	if len(runestone.Edicts) == 0 {
		return false
	}

	if runestone.Pointer != nil && int(*runestone.Pointer) > outputsNumber {
		return false
	}

	for _, edict := range runestone.Edicts {
		if edict.RuneID.Block == 0 && edict.RuneID.TxID != 0 {
			return false
		}

		if outputsNumber < int(edict.Output) {
			return false
		}
	}

	return true
}

// Verify verifies if Runestone contains rune protocol rules violation.
func (runestone *Runestone) Verify(outputsNumber int) error {
	switch {
	case runestone.Pointer != nil && int(*runestone.Pointer) > outputsNumber:
		return &CenotaphError{
			type_:   PointerCenotaphErrorType,
			message: fmt.Sprintf("the Pointer(%d) is out of output idxs range [0;%d)", *runestone.Pointer, outputsNumber),
		}
	case runestone.Etching != nil && (runestone.Etching.Rune == nil || runestone.Etching.Symbol == nil ||
		runestone.Etching.Divisibility == nil || runestone.Etching.Spacers == nil):
		return &CenotaphError{
			type_:   EtchingCenotaphErrorType,
			message: fmt.Sprintf("the Etching field id not full %+v", *runestone.Etching),
		}
	case runestone.Mint != nil && runestone.Mint.Block == 0 && runestone.Mint.TxID != 0:
		return &CenotaphError{
			type_:   MintCenotaphErrorType,
			message: fmt.Sprintf("invalid Mint(%s)", runestone.Mint.String()),
		}
	}
	for idx, edict := range runestone.Edicts {
		if (edict.RuneID.Block == 0 && edict.RuneID.TxID != 0) || int(edict.Output) > outputsNumber {
			return &CenotaphError{
				type_:   EdictsCenotaphErrorType,
				message: fmt.Sprintf("the Edict[%d] is malformed: %+v in output idxs range [0;%d]", idx, edict, outputsNumber),
			}
		}
	}

	return nil
}

// PreparePayload validates raw script payload, removes OP_<...> bytes,
// returns collected data from OP_PUSH_<...> commands.
func PreparePayload(rawPayload []byte) ([]byte, error) {
	if len(rawPayload) < 4 { // OP_RETURN + OP_13 + OP_PUSH_<num> + data(at least 1 byte).
		return nil, errors.New("payload too short")
	}

	if rawPayload[0] != txscript.OP_RETURN {
		return nil, errors.New("missing OP_RETURN")
	}

	if rawPayload[1] != txscript.OP_13 {
		return nil, errors.New("missing OP_13")
	}

	payload := make([]byte, 0, len(rawPayload)-3)
	buffer := bytes.NewReader(rawPayload[2:])
	for buffer.Len() > 0 {
		op, err := buffer.ReadByte()
		if err != nil {
			return nil, err
		}

		if op < txscript.OP_DATA_1 || op > txscript.OP_DATA_75 {
			return nil, errors.New("missing OP_DATA_<num>")
		}

		data := make([]byte, op)
		_, err = buffer.Read(data)
		if err != nil {
			return nil, err
		}

		payload = append(payload, data...)
	}

	// TODO: figure out where it must be.
	// if len(payload) > 18 {
	// 	return nil, ErrOverflow
	// }.

	return payload, nil
}

// IsPossibleRunestone returns true if the script starts with rune protocol bytes sequence.
func IsPossibleRunestone(script []byte) bool {
	switch {
	case len(script) < 4: // OP_RETURN + OP_13 + OP_PUSH_<num> + data(at least 1 byte).
		return false
	case script[0] != txscript.OP_RETURN:
		return false
	case script[1] != txscript.OP_13:
		return false
	case script[2] < txscript.OP_DATA_1 || script[2] > txscript.OP_DATA_75:
		return false
	}

	return true
}

// PayloadIntoIntSequence decodes payload in LEB128 into integer sequence.
func PayloadIntoIntSequence(payload []byte) ([]*big.Int, error) {
	sequence := make([]*big.Int, 0)
	data := bytes.NewReader(payload)
	for data.Len() > 0 {
		num, err := leb128.DecodeUnsigned(data)
		if err != nil {
			return nil, err
		}

		sequence = append(sequence, num)
	}

	return sequence, nil
}

// IntSequenceIntoPayload encodes integer sequence into payload in LEB128.
func IntSequenceIntoPayload(sequence []*big.Int) ([]byte, error) {
	payload := make([]byte, 0)
	for _, num := range sequence {
		bytes, err := leb128.EncodeUnsigned(num)
		if err != nil {
			return nil, err
		}

		payload = append(payload, bytes...)
	}

	return payload, nil
}
