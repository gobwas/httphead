package httphead

// octetType desribes character type.
//
// From the "Basic Rules" chapter of RFC2616
// See https://tools.ietf.org/html/rfc2616#section-2.2
//
// OCTET          = <any 8-bit sequence of data>
// CHAR           = <any US-ASCII character (octets 0 - 127)>
// UPALPHA        = <any US-ASCII uppercase letter "A".."Z">
// LOALPHA        = <any US-ASCII lowercase letter "a".."z">
// ALPHA          = UPALPHA | LOALPHA
// DIGIT          = <any US-ASCII digit "0".."9">
// CTL            = <any US-ASCII control character (octets 0 - 31) and DEL (127)>
// CR             = <US-ASCII CR, carriage return (13)>
// LF             = <US-ASCII LF, linefeed (10)>
// SP             = <US-ASCII SP, space (32)>
// HT             = <US-ASCII HT, horizontal-tab (9)>
// <">            = <US-ASCII double-quote mark (34)>
// CRLF           = CR LF
// LWS            = [CRLF] 1*( SP | HT )
//
// Many HTTP/1.1 header field values consist of words separated by LWS
// or special characters. These special characters MUST be in a quoted
// string to be used within a parameter value (as defined in section
// 3.6).
//
// token          = 1*<any CHAR except CTLs or separators>
// separators     = "(" | ")" | "<" | ">" | "@"
// | "," | ";" | ":" | "\" | <">
// | "/" | "[" | "]" | "?" | "="
// | "{" | "}" | SP | HT
type octetType byte

func (t octetType) isChar() bool      { return t&octetChar != 0 }
func (t octetType) isControl() bool   { return t&octetControl != 0 }
func (t octetType) isSeparator() bool { return t&octetSeparator != 0 }
func (t octetType) isSpace() bool     { return t&octetSpace != 0 }
func (t octetType) isToken() bool     { return t&octetToken != 0 }

const (
	octetChar octetType = 1 << iota
	octetControl
	octetSpace
	octetSeparator
	octetToken
)

var octetTypes [256]octetType

func init() {
	for c := 32; c < 256; c++ {
		var t octetType
		if c <= 127 {
			t |= octetChar
		}
		if 0 <= c && c <= 31 || c == 127 {
			t |= octetControl
		}
		switch c {
		case '(', ')', '<', '>', '@', ',', ';', ':', '"', '/', '[', ']', '?', '=', '{', '}', '\\':
			t |= octetSeparator
		case ' ', '\t':
			t |= octetSpace | octetSeparator
		}

		if t.isChar() && !t.isControl() && !t.isSeparator() && !t.isSpace() {
			t |= octetToken
		}

		octetTypes[c] = t
	}
}
