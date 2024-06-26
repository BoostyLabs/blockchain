// Copyright (C) 2024 Creditor Corp. Group.
// See LICENSE for copying information.

package runes

import (
	"math/big"
	"slices"

	"github.com/BoostyLabs/blockchain/internal/sequencereader"
)

// Edict defines transfer values of the rune protocol.
type Edict struct {
	RuneID RuneID
	Amount *big.Int
	Output uint32
}

// ParseEdictsFromIntSeq parses vector of Edicts from number sequence.
func ParseEdictsFromIntSeq(sr *sequencereader.SequenceReader[*big.Int]) ([]Edict, error) {
	if sr.Len()%4 != 0 {
		return nil, ErrCenotaph
	}

	var prevRuneID RuneID
	edicts := make([]Edict, 0, sr.Len()/4)
	for sr.HasNext() {
		// skip errors due to previous mod/div 4 check.
		block, _ := sr.Next()
		tx, _ := sr.Next()
		amount, _ := sr.Next()
		output, _ := sr.Next()

		edict := Edict{
			RuneID: prevRuneID.Next(RuneID{
				Block: block.Uint64(),
				TxID:  uint32(tx.Uint64()),
			}),
			Amount: amount,
			Output: uint32(output.Uint64()),
		}

		prevRuneID.Set(edict.RuneID)
		edicts = append(edicts, edict)
	}

	return edicts, nil
}

// ToIntSeq returns Edict as sequence on integers.
func (edict *Edict) ToIntSeq() []*big.Int {
	return append(edict.RuneID.ToIntSeq(), new(big.Int).Set(edict.Amount), big.NewInt(int64(edict.Output)))
}

// SortEdicts sorts edicts by block number and transaction id.
func SortEdicts(edicts []Edict) {
	slices.SortFunc(edicts, func(a, b Edict) int {
		blockDiff := int(a.RuneID.Block) - int(b.RuneID.Block)
		if blockDiff != 0 {
			return blockDiff
		}

		return int(a.RuneID.TxID) - int(b.RuneID.TxID)
	})
}

// UseDelta converts list of Edits using delta encoding.
func UseDelta(sortedEdicts []Edict) []Edict {
	var (
		deltaEdicts   = make([]Edict, len(sortedEdicts))
		previousBlock uint64
		previousTx    uint32
		blockDelta    uint64
		txDelta       uint32
	)

	for idx, edict := range sortedEdicts {
		blockDelta = edict.RuneID.Block - previousBlock
		if blockDelta == 0 {
			txDelta = edict.RuneID.TxID - previousTx
		} else {
			txDelta = edict.RuneID.TxID
		}

		deltaEdicts[idx] = Edict{
			RuneID: RuneID{
				Block: blockDelta,
				TxID:  txDelta,
			},
			Amount: edict.Amount,
			Output: edict.Output,
		}

		previousBlock = edict.RuneID.Block
		previousTx = edict.RuneID.TxID
	}

	return deltaEdicts
}

// EdictsToIntSeq converts list of Edicts into in list of integers.
func EdictsToIntSeq(edicts []Edict) []*big.Int {
	sequence := make([]*big.Int, 0, len(edicts)*4)
	SortEdicts(edicts)
	for _, edict := range UseDelta(edicts) {
		sequence = append(sequence, edict.ToIntSeq()...)
	}

	return sequence
}
