package text

// NostrEscape for JSON encoding according to RFC8259.
//
// This is the efficient implementation based on the NIP-01 specification:
//
// To prevent implementation differences from creating a different event ID for
// the same event, the following rules MUST be followed while serializing:
//
//	No whitespace, line breaks or other unnecessary formatting should be included
//	in the output JSON. No characters except the following should be escaped, and
//	instead should be included verbatim:
//
//	- A line break, 0x0A, as \n
//	- A double quote, 0x22, as \"
//	- A backslash, 0x5C, as \\
//	- A carriage return, 0x0D, as \r
//	- A tab character, 0x09, as \t
//	- A backspace, 0x08, as \b
//	- A form feed, 0x0C, as \f
//	UTF-8 should be used for encoding.
func NostrEscape(dst, src B) B {
	l := len(src)
	for i := 0; i < l; i++ {
		c := src[i]
		switch {
		case c == '"':
			dst = append(dst, '\\', '"')
		case c == '\\':
			dst = append(dst, '\\', '\\')
		case c == '\b':
			dst = append(dst, '\\', 'b')
		case c == '\t':
			dst = append(dst, '\\', 't')
		case c == '\n':
			dst = append(dst, '\\', 'n')
		case c == '\f':
			dst = append(dst, '\\', 'f')
		case c == '\r':
			dst = append(dst, '\\', 'r')
		default:
			dst = append(dst, c)
		}
	}
	return dst
}

// NostrUnescape reverses the operation of NostrEscape except instead of
// appending it to the provided slice, it rewrites it, eliminating a memory
// copy. Keep in mind that the original JSON will be mangled by this operation,
// but the resultant slices will cost zero allocations.
func NostrUnescape(dst B) B {
	var r, w int
	for ; r < len(dst); r++ {
		if dst[r] == '\\' {
			r++
			c := dst[r]
			switch {
			case c == '"':
				dst[w] = '"'
				w++
			case c == '\\':
				dst[w] = '\\'
				w++
			case c == 'b':
				dst[w] = '\b'
				w++
			case c == 't':
				dst[w] = '\t'
				w++
			case c == 'n':
				dst[w] = '\n'
				w++
			case c == 'f':
				dst[w] = '\f'
				w++
			case c == 'r':
				dst[w] = '\r'
				w++
			default:
				dst[w] = dst[r]
				w++
				dst[w] = c
			}
		} else {
			dst[w] = dst[r]
			w++
		}
	}
	return dst[:w]
}
