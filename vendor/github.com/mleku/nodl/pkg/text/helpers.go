package text

import (
	"io"

	"github.com/mleku/nodl/pkg/hex"
	"github.com/mleku/nodl/pkg/ints"
	"github.com/mleku/nodl/pkg/kind"
	"github.com/mleku/nodl/pkg/kinds"
	"github.com/templexxx/xhex"
)

// JSONKey generates the JSON format for an object key and terminates with the
// semicolon.
func JSONKey(dst, k B) (b B) {
	dst = append(dst, '"')
	dst = append(dst, k...)
	dst = append(dst, '"', ':')
	b = dst
	return
}

// UnmarshalHex takes a byte string that should contain a quoted hexadecimal
// encoded value, decodes it in-place using a SIMD hex codec and returns the
// decoded truncated bytes (the other half will be as it was but no allocation
// is required).
func UnmarshalHex(b B) (h B, rem B, err error) {
	rem = b[:]
	var inQuote bool
	var start int
	for i := 0; i < len(b); i++ {
		if !inQuote {
			if b[i] == '"' {
				inQuote = true
				start = i + 1
			}
		} else {
			if b[i] == '"' {
				h = b[start:i]
				rem = b[i+1:]
				break
			}
		}
	}
	if !inQuote {
		err = io.EOF
		return
	}
	l := len(h)
	if l%2 != 0 {
		err = errorf.E("invalid length for hex: %d, %0x", len(h), h)
		return
	}
	if err = xhex.Decode(h, h); chk.E(err) {
		return
	}
	h = h[:l/2]
	return
}

func UnmarshalQuoted(b B) (content, rem B, err error) {
	rem = b[:]
	for ; len(rem) >= 0; rem = rem[1:] {
		// advance to open quotes
		if rem[0] == '"' {
			rem = rem[1:]
			break
		}
	}
	if len(rem) == 0 {
		err = io.EOF
		return
	}
	var escaping bool
	for len(rem) > 0 {
		if rem[0] == '\\' {
			escaping = true
			content = append(content, rem[0])
			rem = rem[1:]
		} else if rem[0] == '"' {
			if !escaping {
				rem = rem[1:]
				content = NostrUnescape(content)
				return
			}
			content = append(content, rem[0])
			rem = rem[1:]
			escaping = false
		} else {
			escaping = false
			content = append(content, rem[0])
			rem = rem[1:]
		}
	}
	return
}

func MarshalHexArray(dst B, ha []B) (b B) {
	dst = append(dst, '[')
	for i := range ha {
		dst = AppendQuote(dst, ha[i], hex.EncAppend)
		if i != len(ha)-1 {
			dst = append(dst, ',')
		}
	}
	dst = append(dst, ']')
	b = dst
	return
}

// UnmarshalHexArray unpacks a JSON array containing strings with hexadecimal,
// and checks all values have the specified byte size..
func UnmarshalHexArray(b B, size int) (t []B, rem B, err error) {
	rem = b
	var openBracket bool
	for ; len(rem) > 0; rem = rem[1:] {
		if rem[0] == '[' {
			openBracket = true
		} else if openBracket {
			if rem[0] == ',' {
				continue
			} else if rem[0] == ']' {
				rem = rem[1:]
				log.I.F("%s", rem)
				return
			} else if rem[0] == '"' {
				var h B
				if h, rem, err = UnmarshalHex(rem); chk.E(err) {
					return
				}
				if len(h) != size {
					err = errorf.E("invalid hex array size, got %d expect %d",
						len(h), size)
					return
				}
				t = append(t, h)
				if rem[0] == ']' {
					rem = rem[1:]
					// done
					return
				}
			}
		}
	}
	return
}

func MarshalKindsArray(dst B, ka kinds.T) (b B) {
	dst = append(dst, '[')
	for i := range ka {
		dst = ka[i].Marshal(dst)
		if i != len(ka)-1 {
			dst = append(dst, ',')
		}
	}
	dst = append(dst, ']')
	b = dst
	return
}

func UnmarshalKindsArray(b B) (k kinds.T, rem B, err error) {
	rem = b
	var openedBracket bool
	for ; len(rem) > 0; rem = rem[1:] {
		if !openedBracket && rem[0] == '[' {
			openedBracket = true
			continue
		} else if openedBracket {
			if rem[0] == ']' {
				// done
				return
			} else if rem[0] == ',' {
				continue
			}
			var kk int64
			if kk, rem, err = ints.ExtractInt64FromByteString(rem); chk.E(err) {
				return
			}
			k = append(k, kind.T(kk))
			if rem[0] == ']' {
				return
			}
		}
	}
	if !openedBracket {
		log.I.F("\n%v\n%s", k, rem)
		return nil, nil, errorf.E("kinds: failed to unmarshal\n%s\n%s\n%s", k,
			b, rem)
	}
	return
}
