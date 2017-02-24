package httphead

import "bytes"

type item int

const (
	itemToken = iota
	itemSeparator
	itemComment
)

type lexer struct {
	data []byte
	pos  int

	token []byte

	err bool
}

func (l *lexer) next() bool {
	l.pos += skipSpace(l.data[l.pos:])
	if l.pos == len(l.data) {
		return false
	}

	switch l.data[l.pos] {
	case '"': // quoted-string;
		return l.readString()

	case '(': // comment;
		return l.readComment()

	case '\\', ')': // unexpected chars;
		l.err = true
		return false

	default:
		return l.readToken()
	}
}

func (l *lexer) readToken() bool {
	n := fetchToken(l.data[l.pos:])
	l.token = l.data[l.pos:n]
	return true
}

func (l *lexer) readString() bool {
	l.pos++
	return l.readUntil('"')
}

func (l *lexer) readComment() bool {
	l.pos++
	// TODO
	return l.readUntil(')')
}

func findUnescaped(data []byte, c byte) (n int, ok bool) {
	var i int
	for {
		i = bytes.IndexByte(data[n:], c)
		if i == -1 {
			return
		}
		n += i
		if n == 0 || data[n-1] != '\\' {
			return n, true
		}
		n++
	}
	return n, true
}

func (l *lexer) readUntil(c byte) bool {
	var i, n int
	data := l.data[l.pos:]
	for {
		i = bytes.IndexByte(data[n:], c)
		if i == -1 {
			return false
		}
		n += i
		if n == 0 || data[n-1] != '\\' {
			// TODO
			// if no opening before than ok
			// else increment nested objects counter
			break
		}
		n++
	}
	data = data[:n]

	j := bytes.IndexByte(data, '\\')
	if j == -1 {
		l.token = data
		return true
	}

	token := make([]byte, j, n)
	copy(token, data[:j])

	for i = j + 1; i < n; {
		j = bytes.IndexByte(data[i:], '\\')
		if j != -1 {
			token = append(token, data[i:i+j]...)
			i = i + j + 1
		} else {
			token = append(token, data[i:]...)
			break
		}
	}

	l.token = token
	return true
}

// skipSpace skips spaces and lws-sequences from p.
// It returns number ob bytes skipped.
func skipSpace(p []byte) (n int) {
	for len(p) > 0 {
		switch {
		case len(p) >= 3 &&
			p[0] == '\r' &&
			p[1] == '\n' &&
			octetTypes[p[2]].isSpace():
			p = p[3:]
			n += 3
		case octetTypes[p[0]].isSpace():
			p = p[1:]
			n += 1
		default:
			return
		}
	}
	return
}

// fetchToken fetches token from p. It returns starting position and length of
// the token.
func fetchToken(p []byte) (n int) {
	if len(p) == 0 {
		return 0
	}
	for n := 0; n < len(p); n++ {
		c := p[n]
		if !octetTypes[c].isToken() {
			break
		}
	}
	return n + 1
}

//func httpBtsHeaderList(header []byte, it func([]byte) bool) bool {
//	t := tokenizer{src: header}
//	wantMore := false
//	for {
//		token, sep := t.next()
//		wantMore = sep == ','
//		if t.err {
//			return false
//		}
//		if token != nil && !it(token) {
//			return true
//		}
//		if t.empty() {
//			return !wantMore
//		}
//		if token == nil && sep != ',' {
//			return false
//		}
//	}
//}
//
//func httpStrHeaderList(h string, it func(string) bool) bool {
//	// TODO(gobwas): make httpStrTokenizer
//	t := tokenizer{src: strToBytes(h)}
//	wantMore := false
//	for {
//		token, sep := t.next()
//		wantMore = sep == ','
//		if t.err {
//			return false
//		}
//		if token != nil && !it(string(token)) {
//			return true
//		}
//		if t.empty() {
//			return !wantMore
//		}
//		if token == nil && sep != ',' {
//			return false
//		}
//	}
//}
