package event

import (
	"encoding/binary"
	"errors"
	"fmt"
	"unsafe"

	"github.com/minio/sha256-simd"
	"mleku.net/ec/schnorr"
	"mleku.net/nostr/eventid"
	"mleku.net/nostr/hex"
	"mleku.net/nostr/kind"
	"mleku.net/nostr/pubkey"
	"mleku.net/nostr/tag"
	"mleku.net/nostr/tags"
	"mleku.net/nostr/timestamp"
)

const (
	ID = iota
	PubKey
	CreatedAt
	Kind
	Tags
	Content
	Signature
)

var FieldSizes = []int{
	ID:        sha256.Size,
	PubKey:    schnorr.PubKeyBytesLen,
	CreatedAt: binary.MaxVarintLen64,
	Kind:      2,
	Tags:      -1, // -1 indicates variable
	Content:   -1,
	Signature: schnorr.SignatureSize,
}

// HexInSecond is the list of first tag fields that the second is pure hex
var HexInSecond = []byte{'e', 'p'}

// DecimalHexInSecond is the list of first tag fields that have "decimal:hex:"
var DecimalHexInSecond = []byte{'a'}

// ReadBuffer is a control structure for reading and writing buffers.
//
// It keeps track of the current cursor position, and each read
// function increments it to reflect the position of the next field in the data.
//
// All strings extracted from a ReadBuffer will be directly converted to strings
// using unsafe.String and will be garbage collected only once these strings
// fall out of scope.
//
// Thus the buffers cannot effectively be reused, the memory can only be reused
// via GC processing. This avoids data copy as the content fields are the
// biggest in the event.T structure and dominate the size of the whole event
// anyway, so either way this is done there is a tradeoff. This can be mitigated
// by changing the event.T to be a []byte instead.
type ReadBuffer struct {
	Pos int
	Buf []byte
}

var EOF = errors.New("truncated buffer")

// NewReadBuffer returns a new buffer containing the provided slice.
func NewReadBuffer(b []byte) (buf *ReadBuffer) {
	return &ReadBuffer{Buf: b}
}

func (r *ReadBuffer) Bytes() []byte { return r.Buf }

func (r *ReadBuffer) ReadID() (id *eventid.T, err error) {
	end := r.Pos + FieldSizes[ID]
	if len(r.Buf) < end {
		err = log.E.Err("%v", EOF)
		return
	}
	if id, err = eventid.NewFromBytes(r.Buf[r.Pos:end]); chk.E(err) {
		// this actually can't fail after the previous check
		return
	}
	r.Pos = end
	return
}

func (r *ReadBuffer) ReadPubKey() (pk *pubkey.T, err error) {
	end := r.Pos + FieldSizes[PubKey]
	if len(r.Buf) < end {
		err = log.E.Err("%v", EOF)
		return
	}
	if pk, err = pubkey.NewFromBytes(r.Buf[r.Pos:end]); chk.E(err) {
		return
	}
	r.Pos = end
	return
}

func (r *ReadBuffer) ReadCreatedAt() (t timestamp.T, err error) {
	n, advance := binary.Uvarint(r.Buf[r.Pos:])
	if advance <= 0 {
		err = log.E.Err("%v", EOF)
		return
	}
	r.Pos += advance
	t = timestamp.T(n)
	return
}

func (r *ReadBuffer) ReadKind() (k kind.T, err error) {
	end := r.Pos + 2
	if len(r.Buf) < end {
		err = log.E.Err("%v", EOF)
		return
	}
	k = kind.T(binary.LittleEndian.Uint16(r.Buf[r.Pos:]))
	r.Pos = end
	return
}

func (r *ReadBuffer) ReadTags() (t tags.T, err error) {
	// first get the count of tags
	vi, read := binary.Uvarint(r.Buf[r.Pos:])
	if read < 1 {
		err = log.E.Err("%v", EOF)
		return
	}
	nTags := int(vi)
	var end int
	r.Pos += read
	t = make(tags.T, nTags)
	// iterate through the individual tags
	for i := 0; i < nTags; i++ {
		vi, read = binary.Uvarint(r.Buf[r.Pos:])
		if read < 1 {
			err = log.E.Err("%v", EOF)
			return
		}
		lenTag := int(vi)
		r.Pos += read
		t[i] = make(tag.T, 0, lenTag)
		// extract the individual tag strings
		var secondIsHex, secondIsDecimalHex bool
	reading:
		for j := 0; j < lenTag; j++ {
			// get the length prefix
			vi, read = binary.Uvarint(r.Buf[r.Pos:])
			if read < 1 {
				err = log.E.Err("%v %j", EOF)
				log.I.S()
				return
			}
			r.Pos += read
			// now read it off
			end = r.Pos + int(vi)
			if len(r.Buf) < end {
				err = log.E.Err("%v %0x", EOF)
				return
			}
			// we know from this first tag certain conditions that allow
			// data optimizations
			switch {
			case j == 0:
				if vi != 1 {
					break
				}
				for k := range HexInSecond {
					if r.Buf[r.Pos] == HexInSecond[k] {
						secondIsHex = true
					}
				}
				for k := range DecimalHexInSecond {
					if r.Buf[r.Pos] == DecimalHexInSecond[k] {
						secondIsDecimalHex = true
					}
				}
			case j == 1:
				switch {
				case secondIsHex:
					t[i] = append(t[i], hex.Enc(r.Buf[r.Pos:end]))
					r.Pos = end
					continue reading
				case secondIsDecimalHex:
					var k uint16
					var pk []byte
					fieldEnd := r.Pos + 2
					if fieldEnd > end {
						err = log.E.Err("%v", EOF)
						return
					}
					k = binary.LittleEndian.Uint16(r.Buf[r.Pos:fieldEnd])
					r.Pos += 2
					fieldEnd += schnorr.PubKeyBytesLen
					if fieldEnd > end {
						err = log.E.Err("%v got %d expect %d", EOF, fieldEnd,
							end)
						return
					}
					pk = r.Buf[r.Pos:fieldEnd]
					r.Pos = fieldEnd
					t[i] = append(t[i], fmt.Sprintf("%d:%0x:%s",
						k,
						hex.Enc(pk),
						string(r.Buf[r.Pos:end])))
					r.Pos = end
				}
			}
			t[i] = append(t[i], unsafe.String(&r.Buf[r.Pos], vi))
			r.Pos = end
		}
	}
	return
}

func (r *ReadBuffer) ReadContent() (s string, err error) {
	// get the length prefix
	vi, n := binary.Uvarint(r.Buf[r.Pos:])
	if n < 1 {
		err = log.E.Err("%v", EOF)
		return
	}
	r.Pos += n
	end := r.Pos + int(vi)
	if end > len(r.Buf) {
		err = log.E.Err("%v expect %d got %d", EOF, end, len(r.Buf))
		return
	}
	// extract the string
	s = string(r.Buf[r.Pos : r.Pos+int(vi)])
	r.Pos = end
	return
}

func (r *ReadBuffer) ReadSignature() (sig string, err error) {
	end := r.Pos + FieldSizes[Signature]
	if len(r.Buf) < end {
		err = log.E.Err("%v", EOF)
		return
	}
	sig = hex.Enc(r.Buf[r.Pos:end])
	r.Pos = end
	return
}

func (r *ReadBuffer) ReadEvent() (ev *T, err error) {
	ev = &T{}
	if ev.ID, err = r.ReadID(); chk.E(err) {
		return
	}
	if ev.PubKey, err = r.ReadPubKey(); chk.E(err) {
		return
	}
	if ev.CreatedAt, err = r.ReadCreatedAt(); chk.E(err) {
		return
	}
	if ev.Kind, err = r.ReadKind(); chk.E(err) {
		return
	}
	if ev.Tags, err = r.ReadTags(); chk.E(err) {
		return
	}
	if ev.Content, err = r.ReadContent(); chk.E(err) {
		return
	}
	if ev.Sig, err = r.ReadSignature(); chk.E(err) {
		return
	}
	return
}

func BinaryToEvent(b []byte) (ev *T, err error) {
	r := NewReadBuffer(b)
	if ev, err = r.ReadEvent(); chk.E(err) {
		return
	}
	return
}
