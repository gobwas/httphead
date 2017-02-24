package httphead

import (
	"bytes"
	"testing"
)

type quotedStringCase struct {
	label string
	in    []byte
	out   []byte
	err   bool
}

var quotedStringCases = []quotedStringCase{
	{
		label: "nonterm",
		in:    []byte(`"`),
		out:   []byte(``),
		err:   true,
	},
	{
		label: "empty",
		in:    []byte(`""`),
		out:   []byte(``),
	},
	{
		label: "simple",
		in:    []byte(`"hello, world!"`),
		out:   []byte(`hello, world!`),
	},
	{
		label: "quoted",
		in:    []byte(`"hello, \"world\"!"`),
		out:   []byte(`hello, "world"!`),
	},
	{
		label: "quoted",
		in:    []byte(`"\"hello\", \"world\"!"`),
		out:   []byte(`"hello", "world"!`),
	},
}

func TestLexerReadString(t *testing.T) {
	for _, test := range quotedStringCases {
		t.Run(test.label, func(t *testing.T) {
			l := &lexer{data: []byte(test.in)}
			if ok := l.readString(); ok != !test.err {
				t.Errorf("l.ReadString() = %v; want %v", ok, !test.err)
				return
			}
			if !bytes.Equal(test.out, l.token) {
				t.Errorf("l.ReadString() = %s; want %s", string(l.token), string(test.out))
			}
		})
	}
}

func BenchmarkLexerReadString(b *testing.B) {
	for _, bench := range quotedStringCases {
		b.Run(bench.label, func(b *testing.B) {
			for i := 0; i < b.N; i++ {
				l := &lexer{data: []byte(bench.in)}
				_ = l.readString()
			}
		})
	}
}
