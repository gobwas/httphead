// Package httphead contains utils for parsing HTTP and HTTP-grammar compatible
// text protocols headers.
//
// That is, this package first aim is to bring ability to easily parse
// constructions, described here https://tools.ietf.org/html/rfc2616#section-2
package httphead

import (
	"bytes"
	"sort"
	"strings"
)

// ScanTokens parses data in this form:
//
// list = 1#token
//
// It returns false if data is malformed.
func ScanTokens(data []byte, it func([]byte) bool) bool {
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

type Option struct {
	Name       []byte
	Parameters Parameters
}

// Size returns number of bytes need to be allocated for use in opt.Copy.
func (opt Option) Size() int {
	return len(opt.Name) + opt.Parameters.bytes
}

// Copy copies all underlying []byte slices into p and returns new Option.
// Note that p must be at least of opt.Size() length.
func (opt Option) Copy(p []byte) Option {
	n := copy(p, opt.Name)
	opt.Name = p[:n]
	opt.Parameters, p = opt.Parameters.Copy(p[n:])
	return opt
}

func (opt Option) String() string {
	return "{" + string(opt.Name) + " " + opt.Parameters.String() + "}"
}

func NewOption(name string, data map[string]string) Option {
	p := Parameters{}
	for k, v := range data {
		p.Set([]byte(k), []byte(v))
	}
	return Option{
		Name:       []byte(name),
		Parameters: p,
	}
}

type pair struct {
	key, value []byte
}

func (p pair) copy(dst []byte) (pair, []byte) {
	n := copy(dst, p.key)
	p.key = dst[:n]
	m := n + copy(dst[n:], p.value)
	p.value = dst[n:m]

	dst = dst[m:]

	return p, dst
}

func (a Option) Equal(b Option) bool {
	if bytes.Equal(a.Name, b.Name) {
		return a.Parameters.Equal(b.Parameters)
	}
	return false
}

type pairs []pair

func (p pairs) Len() int           { return len(p) }
func (p pairs) Less(a, b int) bool { return bytes.Compare(p[a].key, p[b].key) == -1 }
func (p pairs) Swap(a, b int)      { p[a], p[b] = p[b], p[a] }

func (a Parameters) Equal(b Parameters) bool {
	switch {
	case a.dyn == nil && b.dyn == nil:
	case a.dyn != nil && b.dyn != nil:
	default:
		return false
	}

	ad, bd := a.data(), b.data()
	if len(ad) != len(bd) {
		return false
	}

	sort.Sort(pairs(ad))
	sort.Sort(pairs(bd))

	for i := 0; i < len(ad); i++ {
		av, bv := ad[i], bd[i]
		if !bytes.Equal(av.key, bv.key) || !bytes.Equal(av.value, bv.value) {
			return false
		}
	}
	return true
}

type Parameters struct {
	pos   int
	bytes int
	arr   [8]pair
	dyn   []pair
}

func (p *Parameters) Size() int {
	return p.bytes
}

func (p *Parameters) Copy(dst []byte) (Parameters, []byte) {
	ret := Parameters{
		pos:   p.pos,
		bytes: p.bytes,
	}
	if p.dyn != nil {
		ret.dyn = make([]pair, len(p.dyn))
		for i, v := range p.dyn {
			ret.dyn[i], dst = v.copy(dst)
		}
	} else {
		for i, p := range p.arr {
			ret.arr[i], dst = p.copy(dst)
		}
	}
	return ret, dst
}

func (p *Parameters) Get(key string) (value []byte, ok bool) {
	for _, v := range p.data() {
		if string(v.key) == key {
			return v.value, true
		}
	}
	return nil, false
}

func (p *Parameters) Set(key, value []byte) {
	p.bytes += len(key) + len(value)

	if p.pos < len(p.arr) {
		p.arr[p.pos] = pair{key, value}
		p.pos++
		return
	}

	if p.dyn == nil {
		p.dyn = make([]pair, len(p.arr), len(p.arr)+1)
		copy(p.dyn, p.arr[:])
	}
	p.dyn = append(p.dyn, pair{key, value})
}

func (p *Parameters) ForEach(cb func(k, v []byte) bool) {
	for _, v := range p.data() {
		if !cb(v.key, v.value) {
			break
		}
	}
}

func (p *Parameters) String() (ret string) {
	ret = "["
	for i, v := range p.data() {
		if i > 0 {
			ret += " "
		}
		ret += string(v.key) + ":" + string(v.value)
	}
	return ret + "]"
}

func (p *Parameters) data() []pair {
	if p.dyn != nil {
		return p.dyn
	}
	return p.arr[:p.pos]
}

// ParseOptions parses header data and appends it to given slice of Option.
// It also returns flag of successful (wellformed input) parsing.
func ParseOptions(data []byte, options []Option) ([]Option, bool) {
	var i int
	index := -1
	return options, ScanOptions(data, func(idx int, name, attr, val []byte) Control {
		if idx != index {
			index = idx
			i = len(options)
			options = append(options, Option{Name: name})
		}
		if attr != nil {
			options[i].Parameters.Set(attr, val)
		}
		return ControlContinue
	})
}

type SelectFlag byte

func (f SelectFlag) String() string {
	var flags [2]string
	var n int
	if f&SelectCopy != 0 {
		flags[n] = "copy"
		n++
	}
	if f&SelectUnique != 0 {
		flags[n] = "unique"
		n++
	}
	return "[" + strings.Join(flags[:n], "|") + "]"
}

const (
	SelectCopy SelectFlag = 1 << iota
	SelectUnique
)

// OptionSelector contains configuration for selecting Options from header value.
type OptionSelector struct {
	// Check is a filter function that applied to every Option that possibly
	// could be selected.
	// If Check is nil all options are passed.
	Check func(Option) bool

	Flags SelectFlag

	// Alloc used to allocate slice of bytes when selector is configured with
	// SelectCopy flag. It will be called with number of bytes needed for copy
	// of single Option.
	// If Alloc is nil make is used.
	Alloc func(n int) []byte
}

// Select parses header data and appends it to given slice of Option.
// It also returns flag of successful (wellformed input) parsing.
func (s OptionSelector) Select(data []byte, options []Option) ([]Option, bool) {
	var current Option
	var has bool
	index := -1

	alloc := s.Alloc
	if alloc == nil {
		alloc = defaultAlloc
	}
	check := s.Check
	if check == nil {
		check = defaultCheck
	}

	ok := ScanOptions(data, func(idx int, name, attr, val []byte) Control {
		if idx != index {
			if has && check(current) {
				if s.Flags&SelectCopy != 0 {
					current = current.Copy(alloc(current.Size()))
				}
				options = append(options, current)
				has = false
			}
			if s.Flags&SelectUnique != 0 {
				for i := len(options) - 1; i >= 0; i-- {
					if bytes.Equal(options[i].Name, name) {
						return ControlSkip
					}
				}
			}
			index = idx
			current = Option{Name: name}
			has = true
		}
		if attr != nil {
			current.Parameters.Set(attr, val)
		}

		return ControlContinue
	})
	if has && check(current) {
		if s.Flags&SelectCopy != 0 {
			current = current.Copy(alloc(current.Size()))
		}
		options = append(options, current)
	}

	return options, ok
}

func defaultAlloc(n int) []byte { return make([]byte, n) }
func defaultCheck(Option) bool  { return true }

// ScanOptions parses data in this form:
//
// values = 1#value
// value = token *( ";" param )
// param = token [ "=" (token | quoted-string) ]
//
// It calls given callback with the index of the option, option itself and its
// parameter (attribute and its value, both could be nil). Index is useful when
// header contains multiple choises for the same named option.
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
func ScanOptions(data []byte, it func(index int, option, attribute, value []byte) Control) bool {
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
				state = stateKey
				lexer.SkipEscaped(',')

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
