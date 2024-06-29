package event

import (
	"github.com/minio/sha256-simd"
	"github.com/mleku/nodl/pkg/ec/schnorr"
	k1 "github.com/mleku/nodl/pkg/ec/secp256k1"
	"github.com/mleku/nodl/pkg/hex"
	"github.com/mleku/nodl/pkg/ints"
	"github.com/mleku/nodl/pkg/kind"
	"github.com/mleku/nodl/pkg/tags"
	"github.com/mleku/nodl/pkg/text"
	"github.com/mleku/nodl/pkg/timestamp"
	"lukechampine.com/frand"
)

// T is the primary datatype of nostr. This is the form of the structure that
// defines its JSON string based format.
type T struct {
	// ID is the SHA256 hash of the canonical encoding of the event
	ID B `json:"id"`
	// PubKey is the public key of the event creator
	PubKey B `json:"pubkey"`
	// CreatedAt is the UNIX timestamp of the event according to the event
	// creator (never trust a timestamp!)
	CreatedAt timestamp.T `json:"created_at"`
	// Kind is the nostr protocol code for the type of event. See kind.T
	Kind kind.T `json:"kind"`
	// Tags are a list of tags, which are a list of strings usually structured
	// as a 3 layer scheme indicating specific features of an event.
	Tags tags.T `json:"tags"`
	// Content is an arbitrary string that can contain anything, but usually
	// conforming to a specification relating to the Kind and the Tags.
	Content B `json:"content"`
	// Sig is the signature on the ID hash that validates as coming from the
	// Pubkey.
	Sig B `json:"sig"`
}

var (
	Id        = B("id")
	Pubkey    = B("pubkey")
	CreatedAt = B("created_at")
	Kind      = B("kind")
	Tags      = B("tags")
	Content   = B("content")
	Sig       = B("sig")
)

type C chan *T

// Ascending is a slice of events that sorts in ascending chronological order
type Ascending []*T

func (ev Ascending) Len() int           { return len(ev) }
func (ev Ascending) Less(i, j int) bool { return ev[i].CreatedAt < ev[j].CreatedAt }
func (ev Ascending) Swap(i, j int)      { ev[i], ev[j] = ev[j], ev[i] }

// Descending sorts a slice of events in reverse chronological order (newest
// first)
type Descending []*T

func (e Descending) Len() int           { return len(e) }
func (e Descending) Less(i, j int) bool { return e[i].CreatedAt > e[j].CreatedAt }

func (e Descending) Swap(i, j int) { e[i], e[j] = e[j], e[i] }

func (ev *T) Marshal(dst B) (b B, err error) {
	// open parentheses
	dst = append(dst, '{')
	// ID
	dst = text.JSONKey(dst, Id)
	dst = text.AppendQuote(dst, ev.ID, hex.EncAppend)
	dst = append(dst, ',')
	// PubKey
	dst = text.JSONKey(dst, Pubkey)
	dst = text.AppendQuote(dst, ev.PubKey, hex.EncAppend)
	dst = append(dst, ',')
	// CreatedAt
	dst = text.JSONKey(dst, CreatedAt)
	dst = ints.Int64AppendToByteString(dst, ev.CreatedAt.I64())
	dst = append(dst, ',')
	// Kind
	dst = text.JSONKey(dst, Kind)
	dst = ev.Kind.Marshal(dst)
	dst = append(dst, ',')
	// Tags
	dst = text.JSONKey(dst, Tags)
	dst = ev.Tags.Marshal(dst)
	dst = append(dst, ',')
	// Content
	dst = text.JSONKey(dst, Content)
	dst = text.AppendQuote(dst, ev.Content, text.NostrEscape)
	dst = append(dst, ',')
	// Sig
	dst = text.JSONKey(dst, Sig)
	dst = text.AppendQuote(dst, ev.Sig, hex.EncAppend)
	// close parentheses
	dst = append(dst, '}')
	b = dst
	return
}

func (ev *T) MarshalJSON() (b B, err error) {
	b, err = ev.Marshal(nil)
	return
}

func (ev *T) Serialize() (b B) {
	b, _ = ev.Marshal(nil)
	return
}

// states of the unmarshaler
const (
	beforeOpen = iota
	openParen
	inKey
	inKV
	inVal
	betweenKV
	afterClose
)

func Unmarshal(b B) (ev *T, rem B, err error) {
	ev = &T{}
	rem = b[:]
	var key B
	var state int
	for ; len(rem) >= 0; rem = rem[1:] {
		switch state {
		case beforeOpen:
			if rem[0] == '{' {
				state = openParen
			}
		case openParen:
			if rem[0] == '"' {
				state = inKey
			}
		case inKey:
			if rem[0] == '"' {
				state = inKV
			} else {
				key = append(key, rem[0])
			}
		case inKV:
			if rem[0] == ':' {
				state = inVal
			}
		case inVal:
			switch key[0] {
			case Id[0]:
				if len(key) < len(Id) {
					goto invalid
				}
				var id B
				if id, rem, err = text.UnmarshalHex(rem); chk.E(err) {
					return
				}
				if len(id) != sha256.Size {
					err = errorf.E("invalid ID, require %d got %d", sha256.Size,
						len(id))
					return
				}
				ev.ID = id
				state = betweenKV
			case Pubkey[0]:
				if len(key) < len(Pubkey) {
					goto invalid
				}
				var pk B
				if pk, rem, err = text.UnmarshalHex(rem); chk.E(err) {
					return
				}
				if len(pk) != schnorr.PubKeyBytesLen {
					err = errorf.E("invalid pubkey, require %d got %d",
						schnorr.PubKeyBytesLen, len(pk))
					return
				}
				ev.PubKey = pk
				state = betweenKV
			case Kind[0]:
				if len(key) < len(Kind) {
					goto invalid
				}
				if ev.Kind, rem, err = kind.Unmarshal(rem); chk.E(err) {
					return
				}
				state = betweenKV
			case Tags[0]:
				if len(key) < len(Tags) {
					goto invalid
				}
				if ev.Tags, rem, err = tags.Unmarshal(rem); chk.E(err) {
					return
				}
				state = betweenKV
			case Sig[0]:
				if len(key) < len(Sig) {
					goto invalid
				}
				var sig B
				if sig, rem, err = text.UnmarshalHex(rem); chk.E(err) {
					return
				}
				if len(sig) != schnorr.SignatureSize {
					err = errorf.E("invalid sig length, require %d got %d '%s'",
						schnorr.SignatureSize, len(sig), rem)
					return
				}
				ev.Sig = sig
				state = betweenKV
			case Content[0]:
				// this can be one of two, but minimum of the shortest
				if len(key) < len(Content) {
					goto invalid
				}
				if key[1] == Content[1] {
					if ev.Content, rem, err = text.UnmarshalQuoted(rem); chk.E(err) {
						return
					}
					state = betweenKV
				} else if key[1] == CreatedAt[1] {
					if len(key) < len(CreatedAt) {
						goto invalid
					}
					if ev.CreatedAt, rem, err = timestamp.Unmarshal(rem); chk.E(err) {
						return
					}
					state = betweenKV
				} else {
					goto invalid
				}
			default:
				goto invalid
			}
			key = key[:0]
		case betweenKV:
			if len(rem) == 0 {
				return
			}
			if rem[0] == '}' {
				state = afterClose
				rem = rem[1:]
			} else if rem[0] == ',' {
				state = openParen
			} else if rem[0] == '"' {
				state = inKey
			}
		}
	}
	if len(rem) == 0 && state != afterClose {
		err = errorf.E("invalid event,'%s'", S(b))
	}
	return
invalid:
	err = errorf.E("invalid key,\n'%s'\n'%s'\n'%s'", S(b), S(b[:len(rem)]),
		S(rem))
	return
}

func (ev *T) ToCanonical() (b B) {
	b = append(b, "[0,\""...)
	b = hex.EncAppend(b, ev.PubKey)
	b = append(b, "\","...)
	b = ints.Int64AppendToByteString(b, ev.CreatedAt.I64())
	b = append(b, ',')
	b = ints.Int64AppendToByteString(b, int64(ev.Kind))
	b = append(b, ',')
	b = ev.Tags.Marshal(b)
	b = append(b, ',')
	b = text.AppendQuote(b, ev.Content, text.NostrEscape)
	b = append(b, ']')
	return
}

func Hash(in []byte) (out []byte) {
	h := sha256.Sum256(in)
	return h[:]
}

// GetIDBytes returns the raw SHA256 hash of the canonical form of an T.
func (ev *T) GetIDBytes() []byte { return Hash(ev.ToCanonical()) }

// Sign signs an event with a given Secret Key encoded in hexadecimal.
func (ev *T) Sign(skStr string, so ...schnorr.SignOption) (err error) {
	// secret key hex must be 64 characters.
	if len(skStr) != 64 {
		err = log.E.Err("invalid secret key length, 64 required, got %d: %s",
			len(skStr), skStr)
		log.D.Ln(err)
		return
	}
	// decode secret key hex to bytes
	var skBytes []byte
	if skBytes, err = hex.Dec(skStr); chk.D(err) {
		err = log.E.Err("sign called with invalid secret key '%s': %w", skStr,
			err)
		log.D.Ln(err)
		return
	}
	// parse bytes to get secret key (size checks have been done).
	sk := k1.SecKeyFromBytes(skBytes)
	ev.PubKey = schnorr.SerializePubKey(sk.PubKey())
	err = ev.SignWithSecKey(sk, so...)
	chk.D(err)
	return
}

// SignWithSecKey signs an event with a given *secp256xk1.SecretKey.
func (ev *T) SignWithSecKey(sk *k1.SecretKey,
	so ...schnorr.SignOption) (err error) {

	// sign the event.
	var sig *schnorr.Signature
	ev.ID = ev.GetIDBytes()
	if sig, err = schnorr.Sign(sk, ev.ID, so...); chk.D(err) {
		return
	}
	// we know secret key is good so we can generate the public key.
	ev.PubKey = schnorr.SerializePubKey(sk.PubKey())
	ev.Sig = sig.Serialize()
	return
}

func (ev *T) CheckSignature() (valid bool, err error) {
	// parse pubkey bytes.
	var pk *k1.PublicKey
	if pk, err = schnorr.ParsePubKey(ev.PubKey); chk.D(err) {
		err = log.E.Err("event has invalid pubkey '%0x': %w", ev.PubKey, err)
		log.D.Ln(err)
		return
	}
	// parse signature bytes.
	var sig *schnorr.Signature
	if sig, err = schnorr.ParseSignature(ev.Sig); chk.D(err) {
		err = log.E.Err("failed to parse signature: %w", err)
		log.D.Ln(err)
		return
	}
	// check signature.
	valid = sig.Verify(ev.GetIDBytes(), pk)
	return
}

func GenerateRandomTextNoteEvent(sec *k1.SecretKey, maxSize int) (ev *T,
	err error) {

	l := frand.Intn(maxSize * 6 / 8) // account for base64 expansion
	ev = &T{
		PubKey:    schnorr.SerializePubKey(sec.PubKey()),
		Kind:      kind.TextNote,
		CreatedAt: timestamp.Now(),
		Content:   text.NostrEscape(nil, frand.Bytes(l)),
	}
	if err = ev.SignWithSecKey(sec); chk.E(err) {
		return
	}
	return
}
