package httphead

import "bufio"

// WriteOptions write options list to the dest.
// It uses the same form as {Scan,Parse}Options functions:
// values = 1#value
// value = token *( ";" param )
// param = token [ "=" (token | quoted-string) ]
//
// It wraps valuse into the quoted-string sequence if it contains any
// non-token characters.
func WriteOptions(dest *bufio.Writer, options []Option) {
	for i, opt := range options {
		if i > 0 {
			dest.WriteByte(',')
		}

		writeTokenSanitized(dest, opt.Name)

		for key, val := range opt.Parameters {
			dest.WriteByte(';')
			writeTokenSanitized(dest, key)
			if val != "" {
				dest.WriteByte('=')
				writeTokenSanitized(dest, val)
			}
		}
	}
}

// writeTokenSanitized writes token as is or as quouted string if it contains
// non-token characters.
//
// Note that is is not expects LWS sequnces be in s, cause LWS is used only as
// header field continuation:
// "A CRLF is allowed in the definition of TEXT only as part of a header field
// continuation. It is expected that the folding LWS will be replaced with a
// single SP before interpretation of the TEXT value."
// See https://tools.ietf.org/html/rfc2616#section-2
//
// That is we sanitizing s for writing, so there could not be any header field
// continuation.
// That is any CRLF will be escaped as any other control characters not allowd in TEXT.
func writeTokenSanitized(bw *bufio.Writer, s string) {
	var qt bool
	var pos int
	for i := 0; i < len(s); i++ {
		c := s[i]

		if !OctetTypes[c].IsToken() && !qt {
			qt = true
			bw.WriteByte('"')
		}
		if OctetTypes[c].IsControl() || c == '"' {
			if !qt {
				qt = true
				bw.WriteByte('"')
			}
			bw.WriteString(s[pos:i])
			bw.WriteByte('\\')
			bw.WriteByte(c)
			pos = i + 1
		}
	}
	if !qt {
		bw.WriteString(s)
	} else {
		bw.WriteString(s[pos:])
		bw.WriteByte('"')
	}
}
