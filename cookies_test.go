package httphead

import (
	"bytes"
	"testing"
)

type cookieTuple struct {
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
			{[]byte(`foo`), []byte(`bar`)},
		},
	},
	{
		label: "simple",
		in:    []byte(`foo=bar; bar=baz`),
		ok:    true,
		exp: []cookieTuple{
			{[]byte(`foo`), []byte(`bar`)},
			{[]byte(`bar`), []byte(`baz`)},
		},
	},
	{
		label: "simple_duplicate",
		in:    []byte(`foo=bar; bar=baz; foo=bar`),
		ok:    true,
		exp: []cookieTuple{
			{[]byte(`foo`), []byte(`bar`)},
			{[]byte(`bar`), []byte(`baz`)},
			{[]byte(`foo`), []byte(`bar`)},
		},
	},
	{
		label: "simple_quoted",
		in:    []byte(`foo="bar"`),
		ok:    true,
		exp: []cookieTuple{
			{[]byte(`foo`), []byte(`bar`)},
		},
	},
	{
		label: "error_trailing_semicolon",
		in:    []byte(`foo=bar;`),
		exp: []cookieTuple{
			{[]byte(`foo`), []byte(`bar`)},
		},
	},
	{
		label: "error_want_space_between",
		in:    []byte(`foo=bar;bar=baz`),
		exp: []cookieTuple{
			{[]byte(`foo`), []byte(`bar`)},
		},
	},
	{
		label: "error_value_dquote",
		in:    []byte(`foo="bar`),
	},
	{
		label: "error_value_dquote",
		in:    []byte(`foo=bar"`),
	},
	{
		label: "error_value_whitespace",
		in:    []byte(`foo=bar `),
	},
	{
		label: "error_value_whitespace",
		in:    []byte(`foo=b ar`),
	},
	{
		label: "error_value_quoted_whitespace",
		in:    []byte(`foo="b ar"`),
	},
}

func TestScanCookies(t *testing.T) {
	for _, test := range cookiesCases {
		t.Run(test.label, func(t *testing.T) {
			var act []cookieTuple
			ok := ScanCookies(test.in, true, func(k, v []byte) bool {
				act = append(act, cookieTuple{k, v})
				return true
			})
			if ok != test.ok {
				t.Errorf("unexpected result: %v; want %v", ok, test.ok)
			}

			if an, en := len(act), len(test.exp); an != en {
				t.Errorf("unexpected length of result: %d; want %d", an, en)
			} else {
				for i, ev := range test.exp {
					if av := act[i]; !bytes.Equal(av.key, ev.key) || !bytes.Equal(av.value, ev.value) {
						t.Errorf(
							"unexpected %d-th tuple: %#q=%#q; want %#q=%#q", i,
							string(av.key), string(av.value),
							string(ev.key), string(ev.value),
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
				ScanCookies(test.in, true, func(_, _ []byte) bool {
					return true
				})
			}
		})
	}
}
