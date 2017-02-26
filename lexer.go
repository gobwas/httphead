package httphead

import "bytes"

type itemType int

const (
	itemUndef itemType = iota
	itemToken
	itemSeparator
	itemString
	itemComment
)

type lexer struct {
	data []byte
	pos  int

	itemType  itemType
	itemBytes []byte

	err bool
}

func (l *lexer) next() bool {
	if l.err {
		return false
	}

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
	n, t := fetchToken(l.data[l.pos:])
	if n == -1 {
		l.err = true
		return false
	}

	l.itemType = t
	l.itemBytes = l.data[l.pos:n]
	l.pos += n

	return true
}

func (l *lexer) readString() (ok bool) {
	l.pos++

	n := readUntil(l.data[l.pos:], '"')
	if n == -1 {
		l.err = true
		return false
	}

	l.itemType = itemString
	l.itemBytes = removeBackslash(l.data[l.pos : l.pos+n])
	l.pos += n

	return true
}

func (l *lexer) readComment() (ok bool) {
	l.pos++

	n := readUntilGreedy(l.data[l.pos:], '(', ')')
	if n == -1 {
		l.err = true
		return false
	}

	l.itemType = itemComment
	l.itemBytes = removeBackslash(l.data[l.pos : l.pos+n])
	l.pos += n

	return true
}

func readUntil(data []byte, c byte) (n int) {
	for {
		i := bytes.IndexByte(data[n:], c)
		if i == -1 {
			return -1
		}
		n += i
		// If found index is not escaped then it is the end.
		if n == 0 || data[n-1] != '\\' {
			break
		}
		n++
	}
	return
}

func readUntilGreedy(data []byte, open, close byte) (n int) {
	var m int
	opened := 1
	for {
		i := bytes.IndexByte(data[n:], close)
		if i == -1 {
			return -1
		}
		n += i
		// If found index is not escaped then it is the end.
		if n == 0 || data[n-1] != '\\' {
			opened--
		}

		for m < i {
			j := bytes.IndexByte(data[m:i], open)
			if j == -1 {
				break
			}
			m += j + 1
			opened++
		}

		if opened == 0 {
			break
		}

		n++
		m = n
	}
	return
}

func removeBackslash(data []byte) []byte {
	// Next search for backslash characters. If no such chars, then set token
	// bytes as slice of data, avoiding copying and allocations.
	j := bytes.IndexByte(data, '\\')
	if j == -1 {
		return data
	}

	n := len(data) - 1

	// If backslashes are present, than allocate slice with n-1 capacity and j
	// length for token. That is, token could be at most n-1 bytes (n minus at
	// least one backslash). Then we copy j bytes which are before first
	// backslash.
	token := make([]byte, n)
	k := copy(token, data[:j])

	for i := j + 1; i < n; {
		j = bytes.IndexByte(data[i:], '\\')
		if j != -1 {
			k += copy(token[k:], data[i:i+j])
			i = i + j + 1
		} else {
			k += copy(token[k:], data[i:])
			break
		}
	}

	return token[:k]
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
// the token. P must be trimmed left from whitespace.
func fetchToken(p []byte) (n int, t itemType) {
	if len(p) == 0 {
		return 0, itemUndef
	}

	c := p[0]
	switch {
	case octetTypes[c].isSeparator():
		return 1, itemSeparator

	case octetTypes[c].isToken():
		for n := 1; n < len(p); n++ {
			c := p[n]
			if !octetTypes[c].isToken() {
				break
			}
		}
		return n + 1, itemToken

	default:
		return -1, itemUndef
	}
}

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
