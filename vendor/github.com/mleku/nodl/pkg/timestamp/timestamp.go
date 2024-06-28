package timestamp

import (
	"encoding/binary"
	"time"
	"unsafe"

	"github.com/mleku/nodl/pkg/ints"
)

// T is a convenience type for UNIX 64 bit timestamps of 1 second
// precision.
type T int64

// Now returns the current UNIX timestamp of the current second.
func Now() T { return T(time.Now().Unix()) }

// U64 returns the current UNIX timestamp of the current second as uint64.
func (t T) U64() uint64 { return uint64(t) }

// I64 returns the current UNIX timestamp of the current second as int64.
func (t T) I64() int64 { return int64(t) }

// Time converts a timestamp.Time value into a canonical UNIX 64 bit 1 second
// precision timestamp.
func (t T) Time() time.Time { return time.Unix(int64(t), 0) }

// Int returns the timestamp as an int.
func (t T) Int() int { return int(t) }

func (t T) Bytes() (b B) {
	b = make(B, 8)
	binary.BigEndian.PutUint64(b, uint64(t))
	return
}

// FromTime returns a T from a time.Time
func FromTime(t time.Time) T { return T(t.Unix()) }

// FromUnix converts from a standard int64 unix timestamp.
func FromUnix(t int64) T { return T(t) }

// FromBytes converts from a string of raw bytes.
func FromBytes(b []byte) T { return T(binary.BigEndian.Uint64(b)) }

func FromVarint(b B) (t T, rem B, err error) {
	n, read := binary.Varint(b)
	if read < 1 {
		err = errorf.E("failed to decode varint timestamp %v", b)
		return
	}
	t = T(n)
	rem = b[:read]
	return
}

func ToVarint(dst B, t T) B { return binary.AppendVarint(dst, int64(t)) }

func (t T) FromVarint(dst B) (b B) { return ToVarint(dst, t) }

func (t T) String() S {
	b := make([]byte, 0, 19)
	b = ints.Int64AppendToByteString(b, t.I64())
	return unsafe.String(&b[0], len(b))
}

func (t T) Marshal(dst B) (b B) {
	return ints.Int64AppendToByteString(dst, t.I64())
}

func Unmarshal(b B) (t T, rem B, err error) {
	var n int64
	n, rem, err = ints.ExtractInt64FromByteString(b)
	t = T(n)
	return
}
