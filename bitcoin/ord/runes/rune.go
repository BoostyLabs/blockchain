// Copyright (C) 2024 Creditor Corp. Group.
// See LICENSE for copying information.

package runes

import (
	"errors"
	"math/big"
	"strings"

	"github.com/BoostyLabs/blockchain/internal/numbers"
)

// DefaultSpacer defines default spacer for Rune name.
const DefaultSpacer = '•'

const (
	// SubsidyHalvingInterval defines how may blocks between halvings.
	SubsidyHalvingInterval uint64 = 210_000

	// ProtocolBlockStart defines the block when protocol was launched.
	ProtocolBlockStart uint64 = 840_000
	// UnlockNamePeriod defines interval in blocks to unlock shorter name.
	UnlockNamePeriod uint64 = 17_500

	// StartNameLength defines minimum name length on the ProtocolBlockStart.
	StartNameLength = 13
)

// INFO: [Rust impl] definition of name unlocking steps.
var steps = func() (steps [28]*big.Int) {
	steps[0], _ = new(big.Int).SetString("0", 10)
	steps[1], _ = new(big.Int).SetString("26", 10)
	steps[2], _ = new(big.Int).SetString("702", 10)
	steps[3], _ = new(big.Int).SetString("18278", 10)
	steps[4], _ = new(big.Int).SetString("475254", 10)
	steps[5], _ = new(big.Int).SetString("12356630", 10)
	steps[6], _ = new(big.Int).SetString("321272406", 10)
	steps[7], _ = new(big.Int).SetString("8353082582", 10)
	steps[8], _ = new(big.Int).SetString("217180147158", 10)
	steps[9], _ = new(big.Int).SetString("5646683826134", 10)
	steps[10], _ = new(big.Int).SetString("146813779479510", 10)
	steps[11], _ = new(big.Int).SetString("3817158266467286", 10)
	steps[12], _ = new(big.Int).SetString("99246114928149462", 10)
	steps[13], _ = new(big.Int).SetString("2580398988131886038", 10)
	steps[14], _ = new(big.Int).SetString("67090373691429037014", 10)
	steps[15], _ = new(big.Int).SetString("1744349715977154962390", 10)
	steps[16], _ = new(big.Int).SetString("45353092615406029022166", 10)
	steps[17], _ = new(big.Int).SetString("1179180408000556754576342", 10)
	steps[18], _ = new(big.Int).SetString("30658690608014475618984918", 10)
	steps[19], _ = new(big.Int).SetString("797125955808376366093607894", 10)
	steps[20], _ = new(big.Int).SetString("20725274851017785518433805270", 10)
	steps[21], _ = new(big.Int).SetString("538857146126462423479278937046", 10)
	steps[22], _ = new(big.Int).SetString("14010285799288023010461252363222", 10)
	steps[23], _ = new(big.Int).SetString("364267430781488598271992561443798", 10)
	steps[24], _ = new(big.Int).SetString("9470953200318703555071806597538774", 10)
	steps[25], _ = new(big.Int).SetString("246244783208286292431866971536008150", 10)
	steps[26], _ = new(big.Int).SetString("6402364363415443603228541259936211926", 10)
	steps[27], _ = new(big.Int).SetString("166461473448801533683942072758341510102", 10)
	return steps
}()

// base26 defines 26 as *big.Int.
var base26 = big.NewInt(26)

// FirstReservedRuneNameInt defines FirstReservedRuneName as number.
var FirstReservedRuneNameInt, _ = new(big.Int).SetString("6402364363415443603228541259936211926", 10)

// FirstReservedRuneName defines first reserved rune name AAAAAAAAAAAAAAAAAAAAAAAAAAA.
var FirstReservedRuneName = RuneReserve(RuneID{0, 0})

// intToChar defines conversion rules from integers to chars.
var intToChar = map[int64]byte{
	0:  'A',
	1:  'B',
	2:  'C',
	3:  'D',
	4:  'E',
	5:  'F',
	6:  'G',
	7:  'H',
	8:  'I',
	9:  'J',
	10: 'K',
	11: 'L',
	12: 'M',
	13: 'N',
	14: 'O',
	15: 'P',
	16: 'Q',
	17: 'R',
	18: 'S',
	19: 'T',
	20: 'U',
	21: 'V',
	22: 'W',
	23: 'X',
	24: 'Y',
	25: 'Z',
}

// Rune defines rune names and encodes as modified base-26 integers.
type Rune struct {
	value *big.Int
}

// NewRuneFromString creates new Rune from string name.
// NOTE: Valid symbols are A-Z only.
func NewRuneFromString(runeStr string) (*Rune, error) {
	var value = big.NewInt(0)
	for i, c := range runeStr {
		if i > 0 {
			value.Add(value, numbers.OneBigInt)
		}
		value = value.Mul(value, base26)
		if c < 'A' || c > 'Z' {
			return nil, errors.New("invalid symbol in the rune")
		}
		value = value.Add(value, big.NewInt(int64(c)-'A'))
	}

	if numbers.IsGreater(value, numbers.MaxUInt128Value) {
		return nil, errors.New("value overflows uint128")
	}
	if numbers.IsGreater(value, FirstReservedRuneNameInt) {
		return nil, errors.New("reserved name")
	}

	return &Rune{value: value}, nil
}

// NewRuneFromStringWithSpacer creates new Rune from string name with spacers scanned.
//
//	NOTE:
//	- Instead of empty spacer the default one will be used.
//	- If many spacers were provided, the first one will be used.
func NewRuneFromStringWithSpacer(runeStr string, spacer ...rune) (*Rune, uint32, error) {
	var s = DefaultSpacer
	if len(spacer) > 0 {
		s = spacer[0]
	}

	var (
		spacers uint32
		idx     uint
	)
	for _, char := range runeStr {
		if char == s {
			spacers |= 1 << (idx - 1)
		} else {
			idx++
		}
	}

	runeStr = strings.Map(func(r rune) rune {
		if r >= 'A' && r <= 'Z' {
			return r
		}

		return -1
	}, runeStr)
	rune_, err := NewRuneFromString(runeStr)
	if err != nil {
		return nil, 0, err
	}

	return rune_, spacers, nil
}

// NewRuneFromNumber creates new Rune from number.
func NewRuneFromNumber(number *big.Int) (*Rune, error) {
	if numbers.IsGreater(number, numbers.MaxUInt128Value) || number.Sign() < 0 {
		return nil, errors.New("invalid number")
	}
	if !numbers.IsLess(number, FirstReservedRuneNameInt) {
		return nil, errors.New("reserved name")
	}

	return &Rune{value: number}, nil
}

// Value returns Rune name as number.
func (r *Rune) Value() *big.Int {
	return r.value
}

// String returns Rune name as string.
func (r *Rune) String() string {
	var value = new(big.Int).Set(r.value)
	if numbers.IsEqual(value, numbers.MaxUInt128Value) {
		return "BCGDENLQRQWDSLRUGSNLBTMFIJAV"
	}

	value = value.Add(value, numbers.OneBigInt)
	var symbol string
	for value.Sign() > 0 {
		valueSubOne := new(big.Int).Sub(value, numbers.OneBigInt)
		idx := new(big.Int).Mod(valueSubOne, base26)

		symbol = string(intToChar[idx.Int64()]) + symbol

		value = valueSubOne.Div(valueSubOne, base26)
	}

	return symbol
}

// StringWithSeparator returns Rune name as string with provides spacer.
//
//	NOTE:
//	- Instead of empty spacer the default one will be used.
//	- If many spacers were provided, the first one will be used.
func (r *Rune) StringWithSeparator(spacers uint32, spacer ...rune) string {
	rune_ := r.String()

	var s = string(DefaultSpacer)
	if len(spacer) > 0 {
		s = string(spacer[0])
	}

	symbol := ""
	for idx, char := range rune_ {
		symbol += string(char)

		if idx < len(rune_)-1 && spacers&(1<<idx) != 0 {
			symbol += s
		}
	}

	return symbol
}

// RuneReserve returns allocated rune name in case it was omitted in etching.
func RuneReserve(runeID RuneID) *Rune {
	// INFO: [Rust impl] 6402364363415443603228541259936211926 + (u128::from(block) << 32 | u128::from(tx))
	reservedName := new(big.Int).Add(FirstReservedRuneNameInt, new(big.Int).Or(
		new(big.Int).Lsh(big.NewInt(int64(runeID.Block)), 32),
		big.NewInt(int64(runeID.TxID))))

	return &Rune{value: reservedName}
}

// MinNameLength returns unlocked rune name length depending on block.
func MinNameLength(currentBlock uint64) int {
	if currentBlock < ProtocolBlockStart {
		return StartNameLength
	}

	for i := uint64(1); i < StartNameLength; i++ {
		if ProtocolBlockStart+UnlockNamePeriod*(i-1) <= currentBlock && currentBlock < ProtocolBlockStart+UnlockNamePeriod*i {
			return StartNameLength - int(i)
		}
	}

	return 0
}

// MinAtHeight defines minimum unlocked rune name depending on block height.
// INFO: [Rust implementation].
func MinAtHeight(height uint64) *Rune {
	offset := height + 1
	start := ProtocolBlockStart
	end := ProtocolBlockStart + SubsidyHalvingInterval

	if offset < start {
		return &Rune{value: steps[12]}
	}

	if offset >= end {
		return &Rune{value: big.NewInt(0)}
	}

	progress := offset - start
	length := 12 - (progress / UnlockNamePeriod)
	end = steps[length-1].Uint64()
	start = steps[length].Uint64()
	remainder := progress % UnlockNamePeriod

	return &Rune{value: new(big.Int).SetUint64(start - ((start - end) * remainder / UnlockNamePeriod))}
}
