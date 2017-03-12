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

type Control byte

const (
	ControlContinue Control = iota
	ControlBreak
	ControlSkip
)

// Parameters parses data like this form:
//
// values = 1#value
// value = token *( ";" param )
// param = token [ "=" (token | quoted-string) ]
//
// It calls given callback with the index of the key and param-value pair for
// that key. That is, index is useful when header contains multiple choises for
// the same key.
//
// Given callback should return one of the defined Control* values.
// ControlSkip means that passed key is not in caller's interest. That is, all
// parameters of that key will be skipped.
// ControlBreak means that no more keys and parameters should be parsed. That
// is, it must break parsing immediately.
// ControlContinue means that caller want to receive next parameter and its
// value or the next key.
//
// It returns false if data is malformed.
func Parameters(data []byte, it func(index int, key, param, value []byte) Control) bool {
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
		index             int
		key, param, value []byte
		mustCall          bool
	)
	for lexer.Next() {
		var (
			call      bool
			growIndex int
		)

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
				// Nothing to do.

			case isComma(v) && state == stateParamBeforeName:
				state = stateKey
				// Make call only if we have not called this key yet.
				call = mustCall
				if !call {
					// If we have already called callback with the key
					// that just ended.
					index++
				} else {
					// Else grow the index after calling callback.
					growIndex = 1
				}

			case isComma(v) && state == stateParamBeforeValue:
				state = stateKey
				growIndex = 1
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
			switch it(index, key, param, value) {
			case ControlBreak:
				// User want to stop to parsing parameters.
				return true

			case ControlSkip:
				// User want to skip current param.
				lexer.Skip(',')

			case ControlContinue:
				// User is interested in rest of parameters.
				// Nothing to do.

			default:
				panic("unexpected control value")
			}
			ok = true
			param = nil
			value = nil
			mustCall = false
			index += growIndex
		}
	}
	if mustCall {
		ok = true
		it(index, key, param, value)
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
