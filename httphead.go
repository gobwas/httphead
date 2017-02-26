// Package httphead contains utils for parsing HTTP and HTTP-grammar compatible
// text protocols headers.
//
// That is, this package first aim is to bring ability to easily parse
// constructions, described here https://tools.ietf.org/html/rfc2616#section-2
package httphead

func HeaderList(data []byte, it func([]byte) bool) bool {
	lexer := &lexer{data: data}

	for lexer.next() {
		switch lexer.itemType {
		case itemToken, itemString:
			if !it(lexer.itemBytes) {
				return true
			}
		case itemSeparator:
			if len(lexer.itemBytes) != 1 || lexer.itemBytes[0] != ',' {
				return false
			}
		default:
			return false
		}
	}

	if lexer.err {
		panic("error")
	}
	return true
}
