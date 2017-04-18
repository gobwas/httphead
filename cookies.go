package httphead

func ScanCookies(data []byte, it func(index int, k, v []byte) Control) bool {
	lexer := &Scanner{data: data}

	var (
		index int

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
		var growIndex int

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
			if !validateCookieValue(value) {
				return false
			}

			state = stateBeforeKey
			growIndex = 1
		}

		switch it(index, key, value) {
		case ControlBreak:
			return true

		case ControlSkip:
			state = stateKey
			lexer.Skip(';')

		case ControlContinue:
			//

		default:
			panic("unexpected control value")
		}

		index += growIndex
		value = nil
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

func validateCookieValue(value []byte) bool {
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
