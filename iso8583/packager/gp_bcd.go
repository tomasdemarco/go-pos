package packager

import (
	"bytes"
	"fmt"
	"github.com/tomasdemarco/iso8583/encoding"
)

// BCD implements the Encoder interface for BCD (Binary-Coded Decimal) encoding.
// It encodes and decodes decimal strings to/from BCD byte slices.
type BCD struct {
	length   int
	padRight bool
	odd      bool
}

// NewBcdEncoder creates a new BCD encoder.
// `padRight` indicates whether to right-pad the input string with a '0' if its length is odd
// before encoding to ensure an even number of digits for BCD conversion.
func NewBcdEncoder(padRight bool) encoding.Encoder {
	return &BCD{padRight: padRight}
}

// Encode converts a decimal string to a BCD byte slice.
// If `padLeft` is true and the source string has an odd length, it will be left-padded with '0'.
func (e *BCD) Encode(src string) ([]byte, error) {
	e.length = len(src)

	start := 0
	d := make([]byte, (len(src)+1)/2)

	if len(src)%2 == 1 && e.padRight {
		start = 1
	}

	for i := start; i < len(src)+start; i++ {
		n := i / 2
		digit := src[i-start] - '0'
		if i%2 == 1 {
			d[n] |= digit
		} else {
			d[n] |= digit << 4
		}
	}
	return d, nil
}

// Decode converts a BCD byte slice to a decimal string.
// It reads up to the configured length.
func (e *BCD) Decode(src []byte) (string, error) {
	if len(src) < e.length {
		return "", fmt.Errorf("%w: expected %d, got %d", encoding.ErrNotEnoughDataToDecode, e.length, len(src))
	}

	src = src[:e.length]
	start := 0
	var d bytes.Buffer

	for i := start; i < len(src)*2+start; i++ {
		shift := 0
		if i%2 == 1 {
			shift = 0
		} else {
			shift = 4
		}

		c := (src[i/2] >> shift) & 0xF
		var char rune
		if c < 10 {
			char = rune(c + '0')
		} else {
			char = rune(c - 10 + 'A')
		}

		if char == 'D' {
			char = '='
		}
		d.WriteRune(char)
	}

	str := d.String()
	if e.odd {
		if e.padRight {
			str = str[:len(str)-1]
		} else {
			str = str[1:]
		}
	}

	return str, nil
}

// SetLength sets the length for the BCD encoder.
func (e *BCD) SetLength(length int) {
	if length%2 != 0 {
		e.odd = true
		length++
	}

	e.length = length / 2
}

func (e *BCD) GetLength() int {
	return e.length
}

func (e *BCD) GetType() encoding.Encoding {
	return encoding.Bcd
}
