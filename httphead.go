// Package httphead contains utils for parsing HTTP and HTTP-grammar compatible
// text protocols headers.
//
// That is, this package first aim is to bring ability to easily parse
// constructions, described here https://tools.ietf.org/html/rfc2616#section-2
package httphead

// List parses data in this form:
//
// list = 1#token
//
// It returns false if data is malformed.
func List(data []byte, it func([]byte) bool) bool {
	lexer := &Scanner{data: data}

	var ok bool
	for lexer.Next() {
		switch lexer.Type() {
		case ItemToken:
			ok = true
			if !it(lexer.Bytes()) {
				return true
			}
		case ItemSeparator:
			if !isComma(lexer.Bytes()) {
				return false
			}
		default:
			return false
		}
	}

	return ok && !lexer.err
}

// Parameters parses data like this form:
//
// values = 1#value
// value = token *( ";" param )
// param = token [ "=" (token | quoted-string) ]
//
// It returns false if data is malformed.
func Parameters(data []byte, it func(key, param, value []byte) bool) bool {
	lexer := &Scanner{data: data}

	var ok bool
	var state int
	const (
		stateKey = iota
		stateParamBeforeName
		stateParamName
		stateParamBeforeValue
		stateParamValue
	)

	var (
		key, param, value []byte
		mustCall          bool
	)
	for lexer.Next() {
		var call bool

		t := lexer.Type()
		v := lexer.Bytes()

		switch t {
		case ItemToken:
			switch state {
			case stateKey, stateParamBeforeName:
				key = v
				state = stateParamBeforeName
				mustCall = true
			case stateParamName:
				param = v
				state = stateParamBeforeValue
				mustCall = true
			case stateParamValue:
				value = v
				state = stateParamBeforeName
				call = true
			default:
				return false
			}

		case ItemString:
			if state != stateParamValue {
				return false
			}
			value = v
			state = stateParamBeforeName
			call = true

		case ItemSeparator:
			switch {
			case isComma(v) && state == stateKey:
				// do nothing

			case isComma(v) && state == stateParamBeforeName:
				state = stateKey
				// Make call only if we have not called this key yet.
				call = mustCall

			case isComma(v) && state == stateParamBeforeValue:
				state = stateKey
				call = true

			case isSemicolon(v) && state == stateParamBeforeName:
				state = stateParamName

			case isSemicolon(v) && state == stateParamBeforeValue:
				state = stateParamName
				call = true

			case isEquality(v) && state == stateParamBeforeValue:
				state = stateParamValue

			default:
				return false
			}

		default:
			return false
		}

		if call {
			ok = true
			if !it(key, param, value) {
				return true
			}
			param = nil
			value = nil
			mustCall = false
		}
	}
	if mustCall {
		ok = true
		it(key, param, value)
	}

	return ok && !lexer.err
}

func isComma(b []byte) bool {
	return len(b) == 1 && b[0] == ','
}
func isSemicolon(b []byte) bool {
	return len(b) == 1 && b[0] == ';'
}
func isEquality(b []byte) bool {
	return len(b) == 1 && b[0] == '='
}
