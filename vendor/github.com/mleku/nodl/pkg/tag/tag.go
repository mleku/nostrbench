package tag

import (
	"bytes"
	"unsafe"

	"github.com/mleku/nodl/pkg/normalize"
	"github.com/mleku/nodl/pkg/text"
)

// The tag position meanings so they are clear when reading.
const (
	Key = iota
	Value
	Relay
)

// T marker strings for e (reference) tags.
const (
	MarkerReply   = "reply"
	MarkerRoot    = "root"
	MarkerMention = "mention"
)

// T is a list of strings with a literal ordering.
//
// Not a set, there can be repeating elements.
type T []B

func New(fields ...string) (t T) {
	t = make(T, len(fields))
	for i, field := range fields {
		t[i] = B(field)
	}
	return
}

// StartsWith checks a tag has the same initial set of elements.
//
// The last element is treated specially in that it is considered to match if
// the candidate has the same initial substring as its corresponding element.
func (t T) StartsWith(prefix T) bool {
	prefixLen := len(prefix)

	if prefixLen > len(t) {
		return false
	}
	// check initial elements for equality
	for i := 0; i < prefixLen-1; i++ {
		if !bytes.Equal(prefix[i], t[i]) {
			return false
		}
	}
	// check last element just for a prefix
	return bytes.HasPrefix(t[prefixLen-1], prefix[prefixLen-1])
}

// Key returns the first element of the tags.
func (t T) Key() B {
	if len(t) > Key {
		return t[Key]
	}
	return nil
}

// Value returns the second element of the tag.
func (t T) Value() B {
	if len(t) > Value {
		return t[Value]
	}
	return nil
}

var etag, ptag = B("e"), B("p")

// Relay returns the third element of the tag.
func (t T) Relay() (s B) {
	if (bytes.Equal(t.Key(), etag) ||
		bytes.Equal(t.Key(), ptag)) &&
		len(t) >= Relay {

		return normalize.URL(t[Relay])
	}
	return
}

// Marshal appends the JSON form to the passed bytes.
func (t T) Marshal(dst B) (b B) {
	dst = append(dst, '[')
	for i, s := range t {
		if i > 0 {
			dst = append(dst, ',')
		}
		dst = text.AppendQuote(dst, s, text.NostrEscape)
	}
	dst = append(dst, ']')
	return dst
}

// Unmarshal decodes the provided JSON tag list (array of strings), and returns
// any remainder after the close bracket has been encountered.
func Unmarshal(b B) (t T, rem B, err error) {
	var inQuotes, openedBracket bool
	var quoteStart int
	for i := 0; i < len(b); i++ {
		if !openedBracket && b[i] == '[' {
			openedBracket = true
		} else if !inQuotes {
			if b[i] == '"' {
				inQuotes, quoteStart = true, i+1
			} else if b[i] == ']' {
				return t, b[i+1:], err
			}
		} else if b[i] == '\\' && i < len(b)-1 {
			i++
		} else if b[i] == '"' {
			inQuotes, t = false, append(t, text.NostrUnescape(b[quoteStart:i]))
		}
	}
	if !openedBracket || inQuotes {
		log.I.F("\n%v\n%s", t, rem)
		return nil, nil, errorf.E("tag: failed to parse tag")
	}
	return
}

func (t T) String() string {
	b := t.Marshal(nil)
	return unsafe.String(&b[0], len(b))
}

// Clone makes a new tag.T with the same members.
func (t T) Clone() (c T) {
	c = make(T, len(t))
	for i := range t {
		c[i] = t[i]
	}
	return
}

// Contains returns true if the provided element is found in the tag slice.
func (t T) Contains(s B) bool {
	for i := range t {
		if bytes.Equal(t[i], s) {
			return true
		}
	}
	return false
}

// Equal checks that the provided tag list matches.
func (t T) Equal(t1 T) bool {
	if len(t) != len(t1) {
		return false
	}
	for i := range t {
		if !bytes.Equal(t[i], t1[i]) {
			return false
		}
	}
	return true
}
