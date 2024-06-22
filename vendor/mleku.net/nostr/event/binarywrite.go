package event

import (
	"encoding/binary"
	"errors"
	"strconv"
	"strings"

	"mleku.net/ec/schnorr"
	"mleku.net/nostr/eventid"
	"mleku.net/nostr/hex"
	"mleku.net/nostr/kind"
	"mleku.net/nostr/tags"
	"mleku.net/nostr/timestamp"
)

// WriteBuffer the buffer should have its capacity pre-allocated but its length
// initialized to zero, and the length of the buffer acts as its effective
// cursor position.
type WriteBuffer struct {
	Buf []byte
}

func EstimateSize(ev *T) (size int) {
	// id
	size += FieldSizes[ID]
	// pubkey
	size += FieldSizes[PubKey]
	// created_at timestamp
	size += FieldSizes[CreatedAt]
	// kind is most efficient as 2 bytes fixed length integer
	size += FieldSizes[Kind]
	// number of tags is a byte because it is unlikely that this many will ever
	// be used
	size++
	// next a byte for the length of each tag list
	for i := range ev.Tags {
		size++
		for j := range ev.Tags[i] {
			// plus a varint16 for each tag length prefix (very often will be 1
			// byte, occasionally 2, but no more than this
			size += binary.MaxVarintLen16
			// and the length of the actual tag
			size += len(ev.Tags[i][j])
		}
	}
	// length prefix of the content field
	size += binary.MaxVarintLen32
	// the length of the content field
	size += len(ev.Content)
	// and the signature
	size += FieldSizes[Signature]
	return
}

func NewBufForEvent(ev *T) (buf *WriteBuffer) {
	return NewWriteBuffer(EstimateSize(ev))
}

// NewWriteBuffer allocates a slice with zero length and capacity at the given
// length. Use with EstimateSize to get a buffer that will not require a
// secondary allocation step.
func NewWriteBuffer(l int) (buf *WriteBuffer) {
	return &WriteBuffer{Buf: make([]byte, 0, l)}
}

func (w *WriteBuffer) Bytes() []byte { return w.Buf }

func (w *WriteBuffer) Len() int { return len(w.Buf) }

func (w *WriteBuffer) WriteID(id *eventid.T) (err error) {
	w.Buf = append(w.Buf, id.Bytes()...)
	return
}

func (w *WriteBuffer) WritePubKey(pk string) (err error) {
	if len(pk) != 2*schnorr.PubKeyBytesLen {
		return errors.New("pubkey hex must be 64 characters")
	}
	if w.Buf, err = hex.DecAppend(w.Buf, []byte(pk)); chk.E(err) {
		return
	}
	return
}

func (w *WriteBuffer) WriteCreatedAt(t timestamp.T) (err error) {
	w.Buf = binary.AppendUvarint(w.Buf, t.U64())
	return
}

func (w *WriteBuffer) WriteKind(k kind.T) (err error) {
	w.Buf = binary.LittleEndian.AppendUint16(w.Buf, k.ToUint16())
	return
}

func (w *WriteBuffer) WriteTags(t tags.T) (err error) {
	// first a byte for the number of tags
	w.Buf = binary.AppendUvarint(w.Buf, uint64(len(t)))
	for i := range t {
		var secondIsHex, secondIsDecimalHex bool
		// first the length of the tag
		w.Buf = binary.AppendUvarint(w.Buf, uint64(len(t[i])))
	scanning:
		for j := range t[i] {
			// we know from this first tag certain conditions that allow
			// data optimizations
			ts := t[i][j]
			switch {
			case j == 0 && len(ts) == 1:
				for k := range HexInSecond {
					if ts[0] == HexInSecond[k] {
						secondIsHex = true
					}
				}
				for k := range DecimalHexInSecond {
					if ts[0] == DecimalHexInSecond[k] {
						secondIsDecimalHex = true
						// log.I.Ln("second is decimal:hex:string")
					}
				}
			case j == 1:
				switch {
				case secondIsHex:
					// log.I.Ln(t[i][j-1])
					w.Buf = binary.AppendUvarint(w.Buf, uint64(32))
					if w.Buf, err = hex.DecAppend(w.Buf,
						[]byte(ts)); chk.E(err) {
						// the value MUST be hex by the spec
						log.W.Ln(t[i])
						return
					}
					continue scanning
				case secondIsDecimalHex:
					split := strings.Split(t[i][j], ":")
					// append the lengths accordingly
					// first is 2 bytes size
					var n int
					if n, err = strconv.Atoi(split[0]); chk.E(err) {
						return
					}
					// second is a 32 byte value encoded in hex
					if len(split[1]) != 64 {
						err = log.E.Err("invalid length pubkey in a tag: %d")
						return
					}
					// prepend with the appropriate length prefix (we don't need
					// a separate length prefix for the string component)
					w.Buf = binary.AppendUvarint(w.Buf,
						uint64(2+32+len(split[2])))
					// encode a 16 bit kind value
					w.Buf = binary.LittleEndian.
						AppendUint16(w.Buf, uint16(n))
					// encode the 32 byte binary value
					if w.Buf, err = hex.DecAppend(w.Buf,
						[]byte(split[1])); chk.E(err) {
						return
					}
					w.Buf = append(w.Buf, split[2]...)
					continue scanning
				}
			}
			w.Buf = binary.AppendUvarint(w.Buf, uint64(len(ts)))
			w.Buf = append(w.Buf, ts...)
		}
	}
	return
}

func (w *WriteBuffer) WriteContent(s string) (err error) {
	w.Buf = binary.AppendUvarint(w.Buf, uint64(len(s)))
	w.Buf = append(w.Buf, s...)
	return
}

func (w *WriteBuffer) WriteSignature(sig string) (err error) {
	if len(sig) != 2*schnorr.SignatureSize {
		return errors.New("signature must be 128 characters")
	}
	if w.Buf, err = hex.DecAppend(w.Buf, []byte(sig)); chk.E(err) {
		return
	}
	return
}

func (w *WriteBuffer) WriteEvent(ev *T) (err error) {
	if err = w.WriteID(ev.ID); chk.E(err) {
		return
	}
	if err = w.WritePubKey(ev.PubKey); chk.E(err) {
		return
	}
	if err = w.WriteCreatedAt(ev.CreatedAt); chk.E(err) {
		return
	}
	if err = w.WriteKind(ev.Kind); chk.E(err) {
		return
	}
	if err = w.WriteTags(ev.Tags); chk.E(err) {
		return
	}
	if err = w.WriteContent(ev.Content); chk.E(err) {
		return
	}
	if err = w.WriteSignature(ev.Sig); chk.E(err) {
		return
	}
	return
}

func EventToBinary(ev *T) (b []byte, err error) {
	w := NewBufForEvent(ev)
	if err = w.WriteEvent(ev); chk.E(err) {
		return
	}
	b = w.Bytes()
	return
}
