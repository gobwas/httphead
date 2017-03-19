package httphead

import "bytes"

type ItemType int

const (
	ItemUndef ItemType = iota
	ItemToken
	ItemSeparator
	ItemString
	ItemComment
)

// Scanner represents header tokens scanner.
// See https://tools.ietf.org/html/rfc2616#section-2
type Scanner struct {
	data []byte
	pos  int

	itemType  ItemType
	itemBytes []byte

	err bool
}

func NewScanner(data []byte) *Scanner {
	return &Scanner{data: data}
}

// Next scans for next token. It returns true on successful scanning, and false
// on error or EOF.
func (l *Scanner) Next() bool {
	if l.err {
		return false
	}

	l.pos += skipSpace(l.data[l.pos:])
	if l.pos == len(l.data) {
		return false
	}

	switch l.data[l.pos] {
	case '"': // quoted-string;
		return l.fetchQuotedString()

	case '(': // comment;
		return l.fetchComment()

	case '\\', ')': // unexpected chars;
		l.err = true
		return false

	default:
		return l.readToken()
	}
}

func (l *Scanner) Skip(c byte) {
	if l.err {
		return
	}

	// Reset scanner state.
	l.itemType = ItemUndef
	l.itemBytes = nil

	if i := ScanUntil(l.data[l.pos:], c); i == -1 {
		// Reached the end of data.
		l.pos = len(l.data)
	} else {
		// Seek data to the next index after first matched char.
		l.pos += i + 1
	}
}

func (l *Scanner) Type() ItemType {
	return l.itemType
}

func (l *Scanner) Bytes() []byte {
	return l.itemBytes
}

func (l *Scanner) readToken() bool {
	n, t := fetchToken(l.data[l.pos:])
	if n == -1 {
		l.err = true
		return false
	}

	l.itemType = t
	l.itemBytes = l.data[l.pos : l.pos+n]
	l.pos += n

	return true
}

func (l *Scanner) fetchQuotedString() (ok bool) {
	l.pos++

	n := ScanUntil(l.data[l.pos:], '"')
	if n == -1 {
		l.err = true
		return false
	}

	l.itemType = ItemString
	l.itemBytes = RemoveByte(l.data[l.pos:l.pos+n], '\\')
	l.pos += n + 1

	return true
}

func (l *Scanner) fetchComment() (ok bool) {
	l.pos++

	n := ScanPairGreedy(l.data[l.pos:], '(', ')')
	if n == -1 {
		l.err = true
		return false
	}

	l.itemType = ItemComment
	l.itemBytes = RemoveByte(l.data[l.pos:l.pos+n], '\\')
	l.pos += n + 1

	return true
}

// ScanUntil scans for first non-escaped character c in given data.
// It returns index of matched c and -1 if c is not found.
func ScanUntil(data []byte, c byte) (n int) {
	for {
		i := bytes.IndexByte(data[n:], c)
		if i == -1 {
			return -1
		}
		n += i
		if n == 0 || data[n-1] != '\\' {
			break
		}
		n++
	}
	return
}

// ScanPairGreedy scans for complete pair of opening and closing chars in greedy manner.
// Note that first opening byte must not be present in data.
func ScanPairGreedy(data []byte, open, close byte) (n int) {
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

// RemoveByte returns data without c. If c is not present in data it returns
// the same slice. If not, it copies data without c.
func RemoveByte(data []byte, c byte) []byte {
	j := bytes.IndexByte(data, c)
	if j == -1 {
		return data
	}

	n := len(data) - 1

	// If character is present, than allocate slice with n-1 capacity. That is,
	// resulting bytes could be at most n-1 length.
	result := make([]byte, n)
	k := copy(result, data[:j])

	for i := j + 1; i < n; {
		j = bytes.IndexByte(data[i:], c)
		if j != -1 {
			k += copy(result[k:], data[i:i+j])
			i = i + j + 1
		} else {
			k += copy(result[k:], data[i:])
			break
		}
	}

	return result[:k]
}

// skipSpace skips spaces and lws-sequences from p.
// It returns number ob bytes skipped.
func skipSpace(p []byte) (n int) {
	for len(p) > 0 {
		switch {
		case len(p) >= 3 &&
			p[0] == '\r' &&
			p[1] == '\n' &&
			OctetTypes[p[2]].IsSpace():
			p = p[3:]
			n += 3
		case OctetTypes[p[0]].IsSpace():
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
func fetchToken(p []byte) (n int, t ItemType) {
	if len(p) == 0 {
		return 0, ItemUndef
	}

	c := p[0]
	switch {
	case OctetTypes[c].IsSeparator():
		return 1, ItemSeparator

	case OctetTypes[c].IsToken():
		for n = 1; n < len(p); n++ {
			c := p[n]
			if !OctetTypes[c].IsToken() {
				break
			}
		}
		return n, ItemToken

	default:
		return -1, ItemUndef
	}
}
