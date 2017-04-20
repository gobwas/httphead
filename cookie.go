package httphead

// ScanCookie maps data to key-value pairs that could lay inside the Cookie header.
//
// If validate is true, then it validates each value bytes to be valid RFC6265
// cookie-octet. If validate is false, then it only strips the double quotes
// (if both first and last byte is double quote) of value.
// You could validate cookie value manually by calling ValidCookieValue().
//
// See https://tools.ietf.org/html/rfc6265#section-4.1.1
func ScanCookie(data []byte, validate bool, it func(key, value []byte) bool) bool {
	lexer := &Scanner{data: data}

	var (
		key   []byte
		value []byte
		state int
	)
	const (
		stateKey = iota
		stateBeforeKey
		stateValue
	)
	for lexer.Next() {
		switch lexer.Type() {
		case ItemToken:
			if state != stateKey {
				return false
			}

			key = lexer.Bytes()
			state = stateValue

		case ItemSeparator:
			if state == stateBeforeKey {
				// Pairs separated by ";" and space, according to the RFC6265:
				//   cookie-pair *( ";" SP cookie-pair )
				if !isSemicolon(lexer.Bytes()) {
					return false
				}
				if lexer.Peek() != ' ' {
					return false
				}

				state = stateKey
				continue
			}

			if state != stateValue || !isEquality(lexer.Bytes()) {
				return false
			}
			if !lexer.NextOctet(';') {
				return false
			}

			value = lexer.Bytes()
			value = stripQuotes(value)
			if validate && !ValidCookieValue(value) {
				return false
			}

			if !it(key, value) {
				return true
			}

			state = stateBeforeKey

		}
	}
	if state != stateBeforeKey {
		return false
	}

	return true
}

func stripQuotes(bts []byte) []byte {
	if last := len(bts) - 1; bts[0] == '"' && bts[last] == '"' {
		return bts[1:last]
	}
	return bts
}

// ValidCookieValue reports whether given value is a valid RFC6265 value
// octets.
func ValidCookieValue(value []byte) bool {
	for _, c := range value {
		if t := OctetTypes[c]; t.IsControl() || t.IsSpace() {
			return false
		}
		switch c {
		case '"', ';', '\\', ',':
			return false
		default:
			if c >= 0x7f {
				return false
			}
		}
	}
	return true
}
