package httphead

import (
	"bytes"
	"testing"
)

type readCase struct {
	label string
	in    []byte
	out   []byte
	err   bool
}

var quotedStringCases = []readCase{
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

var commentCases = []readCase{
	{
		label: "nonterm",
		in:    []byte(`(hello`),
		out:   []byte(``),
		err:   true,
	},
	{
		label: "empty",
		in:    []byte(`()`),
		out:   []byte(``),
	},
	{
		label: "simple",
		in:    []byte(`(hello)`),
		out:   []byte(`hello`),
	},
	{
		label: "quoted",
		in:    []byte(`(hello\)\(world)`),
		out:   []byte(`hello)(world`),
	},
	{
		label: "nested",
		in:    []byte(`(hello(world))`),
		out:   []byte(`hello(world)`),
	},
}

type readTest struct {
	label string
	cases []readCase
	fn    func(*lexer) bool
}

var readTests = []readTest{
	{
		"ReadString",
		quotedStringCases,
		(*lexer).readString,
	},
	{
		"ReadComment",
		commentCases,
		(*lexer).readComment,
	},
}

func TestLexerRead(t *testing.T) {
	for _, bunch := range readTests {
		for _, test := range bunch.cases {
			t.Run(bunch.label+" "+test.label, func(t *testing.T) {
				l := &lexer{data: []byte(test.in)}
				if ok := bunch.fn(l); ok != !test.err {
					t.Errorf("l.%s() = %v; want %v", bunch.label, ok, !test.err)
					return
				}
				if !bytes.Equal(test.out, l.itemBytes) {
					t.Errorf("l.%s() = %s; want %s", bunch.label, string(l.itemBytes), string(test.out))
				}
			})
		}

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

func BenchmarkLexerReadComment(b *testing.B) {
	for _, bench := range commentCases {
		b.Run(bench.label, func(b *testing.B) {
			for i := 0; i < b.N; i++ {
				l := &lexer{data: []byte(bench.in)}
				_ = l.readComment()
			}
		})
	}
}
