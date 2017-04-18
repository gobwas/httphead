package httphead

import (
	"bytes"
	"testing"
)

type cookieTuple struct {
	index      int
	key, value []byte
}

var cookiesCases = []struct {
	label string
	in    []byte
	ok    bool
	exp   []cookieTuple
}{
	{
		label: "simple",
		in:    []byte(`foo=bar`),
		ok:    true,
		exp: []cookieTuple{
			{0, []byte(`foo`), nil},
			{0, []byte(`foo`), []byte(`bar`)},
		},
	},
	{
		label: "simple",
		in:    []byte(`foo=bar; bar=baz`),
		ok:    true,
		exp: []cookieTuple{
			{0, []byte(`foo`), nil},
			{0, []byte(`foo`), []byte(`bar`)},
			{1, []byte(`bar`), nil},
			{1, []byte(`bar`), []byte(`baz`)},
		},
	},
	{
		label: "simple_quoted",
		in:    []byte(`foo="bar"`),
		ok:    true,
		exp: []cookieTuple{
			{0, []byte(`foo`), nil},
			{0, []byte(`foo`), []byte(`bar`)},
		},
	},
	{
		label: "error_trailing_semicolon",
		in:    []byte(`foo=bar;`),
		exp: []cookieTuple{
			{0, []byte(`foo`), nil},
			{0, []byte(`foo`), []byte(`bar`)},
		},
	},
	{
		label: "error_want_space_between",
		in:    []byte(`foo=bar;bar=baz`),
		exp: []cookieTuple{
			{0, []byte(`foo`), nil},
			{0, []byte(`foo`), []byte(`bar`)},
		},
	},
	{
		label: "error_value_dquote",
		in:    []byte(`foo="bar`),
		exp: []cookieTuple{
			{0, []byte(`foo`), nil},
		},
	},
	{
		label: "error_value_dquote",
		in:    []byte(`foo=bar"`),
		exp: []cookieTuple{
			{0, []byte(`foo`), nil},
		},
	},
	{
		label: "error_value_whitespace",
		in:    []byte(`foo=bar `),
		exp: []cookieTuple{
			{0, []byte(`foo`), nil},
		},
	},
	{
		label: "error_value_whitespace",
		in:    []byte(`foo=b ar`),
		exp: []cookieTuple{
			{0, []byte(`foo`), nil},
		},
	},
	{
		label: "error_value_quoted_whitespace",
		in:    []byte(`foo="b ar"`),
		exp: []cookieTuple{
			{0, []byte(`foo`), nil},
		},
	},
}

func TestScanCookies(t *testing.T) {
	for _, test := range cookiesCases {
		t.Run(test.label, func(t *testing.T) {
			var act []cookieTuple
			ok := ScanCookies(test.in, func(i int, k, v []byte) Control {
				act = append(act, cookieTuple{i, k, v})
				return ControlContinue
			})
			if ok != test.ok {
				t.Errorf("unexpected result: %v; want %v", ok, test.ok)
			}

			if an, en := len(act), len(test.exp); an != en {
				t.Errorf("unexpected length of result: %d; want %d", an, en)
			} else {
				for i, ev := range test.exp {
					if av := act[i]; av.index != ev.index || !bytes.Equal(av.key, ev.key) || !bytes.Equal(av.value, ev.value) {
						t.Errorf(
							"unexpected %d-th tuple: #%d %#q=%#q; want #%d %#q=%#q", i,
							av.index, string(av.key), string(av.value),
							ev.index, string(ev.key), string(ev.value),
						)
					}
				}
			}
		})
	}
}

func BenchmarkScanCookies(b *testing.B) {
	for _, test := range cookiesCases {
		b.Run(test.label, func(b *testing.B) {
			for i := 0; i < b.N; i++ {
				ScanCookies(test.in, func(i int, _, _ []byte) Control {
					return ControlContinue
				})
			}
		})
	}
}
