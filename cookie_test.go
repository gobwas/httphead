package httphead

import (
	"bytes"
	"fmt"
	"net/http"
	"testing"
)

type cookieTuple struct {
	name, value []byte
}

var cookieCases = []struct {
	label string
	in    []byte
	ok    bool
	exp   []cookieTuple

	c CookieScanner
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
		label: "duplicate",
		in:    []byte(`foo=bar; bar=baz; foo=bar`),
		ok:    true,
		exp: []cookieTuple{
			{[]byte(`foo`), []byte(`bar`)},
			{[]byte(`bar`), []byte(`baz`)},
			{[]byte(`foo`), []byte(`bar`)},
		},
	},
	{
		label: "quoted",
		in:    []byte(`foo="bar"`),
		ok:    true,
		exp: []cookieTuple{
			{[]byte(`foo`), []byte(`bar`)},
		},
	},
	{
		label: "empty value",
		in:    []byte(`foo=`),
		ok:    true,
		exp: []cookieTuple{
			{[]byte(`foo`), []byte{}},
		},
	},
	{
		label: "empty value",
		in:    []byte(`foo=; bar=baz`),
		ok:    true,
		exp: []cookieTuple{
			{[]byte(`foo`), []byte{}},
			{[]byte(`bar`), []byte(`baz`)},
		},
	},
	{
		label: "quote as value",
		in:    []byte(`foo="; bar=baz`),
		ok:    true,
		exp: []cookieTuple{
			{[]byte(`foo`), []byte{'"'}},
			{[]byte(`bar`), []byte(`baz`)},
		},
		c: CookieScanner{
			DisableValueValidation: true,
		},
	},
	{
		label: "quote as value",
		in:    []byte(`foo="; bar=baz`),
		ok:    true,
		exp: []cookieTuple{
			{[]byte(`bar`), []byte(`baz`)},
		},
	},
	{
		label: "skip_invalid_key",
		in:    []byte(`foo@example.com=1; bar=baz`),
		ok:    true,
		exp: []cookieTuple{
			{[]byte("bar"), []byte("baz")},
		},
	},
	{
		label: "skip_invalid_value",
		in:    []byte(`foo="1; bar=baz`),
		exp: []cookieTuple{
			{[]byte("bar"), []byte("baz")},
		},
		ok: true,
	},
	{
		label: "trailing_semicolon",
		in:    []byte(`foo=bar;`),
		ok:    true,
		exp: []cookieTuple{
			{[]byte(`foo`), []byte(`bar`)},
		},
	},
	{
		label: "trailing_semicolon_strict",
		in:    []byte(`foo=bar;`),
		exp: []cookieTuple{
			{[]byte(`foo`), []byte(`bar`)},
		},
		c: CookieScanner{
			Strict: true,
		},
	},
	{
		label: "want_space_between",
		in:    []byte(`foo=bar;bar=baz`),
		exp: []cookieTuple{
			{[]byte(`foo`), []byte(`bar`)},
			{[]byte(`bar`), []byte(`baz`)},
		},
		ok: true,
	},
	{
		label: "want_space_between_strict",
		in:    []byte(`foo=bar;bar=baz`),
		exp: []cookieTuple{
			{[]byte(`foo`), []byte(`bar`)},
		},
		c: CookieScanner{
			Strict: true,
		},
	},
	{
		label: "value_single_dquote",
		in:    []byte(`foo="bar`),
		ok:    true,
	},
	{
		label: "value_single_dquote",
		in:    []byte(`foo=bar"`),
		ok:    true,
	},
	{
		label: "value_single_dquote",
		in:    []byte(`foo="bar`),
		c: CookieScanner{
			BreakOnError: true,
		},
	},
	{
		label: "value_single_dquote",
		in:    []byte(`foo=bar"`),
		c: CookieScanner{
			BreakOnError: true,
		},
	},
	{
		label: "value_whitespace",
		in:    []byte(`foo=bar `),
		ok:    true,
		exp: []cookieTuple{
			{[]byte(`foo`), []byte(`bar`)},
		},
	},
	{
		label: "value_whitespace_strict",
		in:    []byte(`foo=bar `),
		c: CookieScanner{
			Strict:       true,
			BreakOnError: true,
		},
	},
	{
		label: "value_whitespace",
		in:    []byte(`foo=b ar`),
		ok:    true,
		exp: []cookieTuple{
			{[]byte(`foo`), []byte(`b ar`)},
		},
	},
	{
		label: "value_whitespace_strict",
		in:    []byte(`foo=b ar`),
		c: CookieScanner{
			Strict:       true,
			BreakOnError: true,
		},
	},
	{
		label: "value_whitespace_strict",
		in:    []byte(`foo= bar`),
		c: CookieScanner{
			Strict:       true,
			BreakOnError: true,
		},
	},
	{
		label: "value_quoted_whitespace",
		in:    []byte(`foo="b ar"`),
		ok:    true,
		exp: []cookieTuple{
			{[]byte(`foo`), []byte(`b ar`)},
		},
	},
	{
		label: "value_quoted_whitespace_strict",
		in:    []byte(`foo="b ar"`),
		c: CookieScanner{
			Strict:       true,
			BreakOnError: true,
		},
	},
}

func TestScanCookie(t *testing.T) {
	for _, test := range cookieCases {
		t.Run(test.label, func(t *testing.T) {
			var act []cookieTuple

			ok := test.c.Scan(test.in, func(k, v []byte) bool {
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
					if av := act[i]; !bytes.Equal(av.name, ev.name) || !bytes.Equal(av.value, ev.value) {
						t.Errorf(
							"unexpected %d-th tuple: %#q=%#q; want %#q=%#q", i,
							string(av.name), string(av.value),
							string(ev.name), string(ev.value),
						)
					}
				}
			}

			if test.c != DefaultCookieScanner {
				return
			}

			// Compare with standard library.
			req := http.Request{
				Header: http.Header{
					"Cookie": []string{string(test.in)},
				},
			}
			std := req.Cookies()
			if an, sn := len(act), len(std); an != sn {
				t.Errorf("length of result: %d; standard lib returns %d; details:\n%s", an, sn, dumpActStd(act, std))
			} else {
				for i := 0; i < an; i++ {
					if a, s := act[i], std[i]; string(a.name) != s.Name || string(a.value) != s.Value {
						t.Errorf("%d-th cookie not equal:\n%s", i, dumpActStd(act, std))
						break
					}
				}
			}
		})
	}
}

func BenchmarkScanCookie(b *testing.B) {
	for _, test := range cookieCases {
		b.Run(test.label, func(b *testing.B) {
			for i := 0; i < b.N; i++ {
				test.c.Scan(test.in, func(_, _ []byte) bool {
					return true
				})
			}
		})
		if test.c == DefaultCookieScanner {
			b.Run(test.label+"_std", func(b *testing.B) {
				r := http.Request{
					Header: http.Header{
						"Cookie": []string{string(test.in)},
					},
				}
				for i := 0; i < b.N; i++ {
					_ = r.Cookies()
				}
			})
		}
	}
}

func dumpActStd(act []cookieTuple, std []*http.Cookie) string {
	var buf bytes.Buffer
	fmt.Fprintf(&buf, "actual:\n")
	for i, p := range act {
		fmt.Fprintf(&buf, "\t#%d: %#q=%#q\n", i, p.name, p.value)
	}
	fmt.Fprintf(&buf, "standard:\n")
	for i, c := range std {
		fmt.Fprintf(&buf, "\t#%d: %#q=%#q\n", i, c.Name, c.Value)
	}
	return buf.String()
}
