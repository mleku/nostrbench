package ints

import (
	_ "embed"
)

// run this to regenerate (pointlessly) the base 10 array of 4 places per entry
//go:generate go run ./gen/.

//go:embed base10k.txt
var base10k []byte

const base = 10000

var powers = []int64{
	1,
	1_0000,
	1_0000_0000,
	1_0000_0000_0000,
	1_0000_0000_0000_0000,
}

const zero = '0'
const nine = '9'

// ExtractInt64FromByteString reads a string, which must be a positive integer
// no larger than math.MaxInt64, skipping any non-numeric content before it
func ExtractInt64FromByteString(b B) (n int64, rem B, err error) {
	var sLen int
	// count the digits
	for ; sLen < len(b) && b[sLen] >= zero && b[sLen] <= nine && b[sLen] != ','; sLen++ {
	}
	if sLen == 0 {
		err = errorf.E("zero length number")
		return
	}
	if sLen > 19 {
		err = errorf.E("too big number for int64")
		return
	}
	// the length of the string found
	rem = b[sLen:]
	b = b[:sLen]
	for _, ch := range b {
		ch -= zero
		n = n*10 + int64(ch)
	}
	return
}

// Int64AppendToByteString encodes an *positive* int64 into ASCII decimal format
// in a []byte. This is only for use with timestamp.T and kind.T.
func Int64AppendToByteString(dst []byte, n int64) (b []byte) {
	if n == 0 {
		return append(dst, '0')
	}
	var i int
	var trimmed bool
	k := len(powers)
	for k > 0 {
		k--
		q := n / powers[k]
		if !trimmed && q == 0 {
			continue
		}
		offset := q * 4
		bb := base10k[offset : offset+4]
		if !trimmed {
			for i = range bb {
				if bb[i] != '0' {
					bb = bb[i:]
					trimmed = true
					break
				}
			}
		}
		dst = append(dst, bb...)
		n = n - q*powers[k]
	}
	return dst
}
