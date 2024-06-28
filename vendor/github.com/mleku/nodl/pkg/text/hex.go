package text

import (
	"encoding/hex"
)

func AppendHexEncode(dst, src B) B {
	return hex.AppendEncode(dst,
		src)
}

func AppendHexFromBinary(dst, src B, quote bool) (b B) {
	if quote {
		dst = AppendQuote(dst, src, AppendHexEncode)
	} else {
		dst = AppendHexEncode(dst, src)
	}
	b = dst
	return
}

func AppendBinaryFromHex(dst, src B, unquote bool) (b B,
	err error) {
	if unquote {
		if dst, err = hex.AppendDecode(dst,
			Unquote(src)); chk.E(err) {
			return
		}
	} else {
		if dst, err = hex.AppendDecode(dst, src); chk.E(err) {
			return
		}
	}
	b = dst
	return
}
