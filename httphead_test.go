package httphead

import (
	"bytes"
	"fmt"
	"testing"
)

func ExampleScanTokens() {
	var values []string

	ScanTokens([]byte(`a,b,c`), func(v []byte) bool {
		values = append(values, string(v))
		return v[0] != 'b'
	})

	fmt.Println(values)
	// Output: [a b]
}

func ExampleScanOptions() {
	foo := map[string]string{}

	ScanOptions([]byte(`foo;bar=1;baz`), func(index int, key, param, value []byte) Control {
		foo[string(param)] = string(value)
		return ControlContinue
	})

	fmt.Printf("bar:%s baz:%s", foo["bar"], foo["baz"])
	// Output: bar:1 baz:
}

func ExampleScanOptionsChoise() {
	type pair struct {
		Key, Value string
	}

	// The right part of full header line like:
	//
	// X-My-Header: key;foo=bar;baz,key;baz
	//
	header := []byte(`key;foo=bar;baz,key;baz`)

	choises := make([][]pair, 2)
	ScanOptions(header, func(i int, key, param, value []byte) Control {
		choises[i] = append(choises[i], pair{string(param), string(value)})
		return ControlContinue
	})

	fmt.Println(choises)
	// Output: [[{foo bar} {baz }] [{baz }]]
}

func ExampleParseOptions() {
	options, ok := ParseOptions([]byte(`foo;bar=1,baz`), nil)
	fmt.Println(options, ok)
	// Output: [{foo map[bar:1]} {baz map[]}] true
}

var listCases = []struct {
	label string
	in    []byte
	ok    bool
	exp   [][]byte
}{
	{
		label: "simple",
		in:    []byte(`a,b,c`),
		ok:    true,
		exp: [][]byte{
			[]byte(`a`),
			[]byte(`b`),
			[]byte(`c`),
		},
	},
	{
		label: "simple",
		in:    []byte(`a,b,,c`),
		ok:    true,
		exp: [][]byte{
			[]byte(`a`),
			[]byte(`b`),
			[]byte(`c`),
		},
	},
	{
		label: "simple",
		in:    []byte(`a,b;c`),
		ok:    false,
		exp: [][]byte{
			[]byte(`a`),
			[]byte(`b`),
		},
	},
}

func TestScanTokens(t *testing.T) {
	for _, test := range listCases {
		t.Run(test.label, func(t *testing.T) {
			var act [][]byte
			ok := ScanTokens(test.in, func(v []byte) bool {
				act = append(act, v)
				return true
			})
			if ok != test.ok {
				t.Errorf("unexpected result: %v; want %v", ok, test.ok)
			}
			if an, en := len(act), len(test.exp); an != en {
				t.Errorf("unexpected length of result: %d; want %d", an, en)
			} else {
				for i, ev := range test.exp {
					if av := act[i]; !bytes.Equal(av, ev) {
						t.Errorf("unexpected %d-th value: %#q; want %#q", i, string(av), string(ev))
					}
				}
			}
		})
	}
}

func BenchmarkScanTokens(b *testing.B) {
	for _, bench := range listCases {
		b.Run(bench.label, func(b *testing.B) {
			for i := 0; i < b.N; i++ {
				_ = ScanTokens(bench.in, func(v []byte) bool { return true })
			}
		})
	}
}

type tuple struct {
	index                    int
	option, attribute, value []byte
}

var parametersCases = []struct {
	label string
	in    []byte
	ok    bool
	exp   []tuple
}{
	{
		label: "simple",
		in:    []byte(`a,b,c`),
		ok:    true,
		exp: []tuple{
			{index: 0, option: []byte(`a`)},
			{index: 1, option: []byte(`b`)},
			{index: 2, option: []byte(`c`)},
		},
	},
	{
		label: "simple",
		in:    []byte(`a,b,c;foo=1;bar=2`),
		ok:    true,
		exp: []tuple{
			{index: 0, option: []byte(`a`)},
			{index: 1, option: []byte(`b`)},
			{index: 2, option: []byte(`c`), attribute: []byte(`foo`), value: []byte(`1`)},
			{index: 2, option: []byte(`c`), attribute: []byte(`bar`), value: []byte(`2`)},
		},
	},
	{
		label: "simple",
		in:    []byte(`c;foo;bar=2`),
		ok:    true,
		exp: []tuple{
			{index: 0, option: []byte(`c`), attribute: []byte(`foo`)},
			{index: 0, option: []byte(`c`), attribute: []byte(`bar`), value: []byte(`2`)},
		},
	},
	{
		label: "simple",
		in:    []byte(`foo;bar=1;baz`),
		ok:    true,
		exp: []tuple{
			{index: 0, option: []byte(`foo`), attribute: []byte(`bar`), value: []byte(`1`)},
			{index: 0, option: []byte(`foo`), attribute: []byte(`baz`)},
		},
	},
	{
		label: "simple_quoted",
		in:    []byte(`c;bar="2"`),
		ok:    true,
		exp: []tuple{
			{index: 0, option: []byte(`c`), attribute: []byte(`bar`), value: []byte(`2`)},
		},
	},
	{
		label: "simple_dup",
		in:    []byte(`c;bar=1,c;bar=2`),
		ok:    true,
		exp: []tuple{
			{index: 0, option: []byte(`c`), attribute: []byte(`bar`), value: []byte(`1`)},
			{index: 1, option: []byte(`c`), attribute: []byte(`bar`), value: []byte(`2`)},
		},
	},
	{
		label: "all",
		in:    []byte(`foo;a=1;b=2;c=3,bar;z,baz`),
		ok:    true,
		exp: []tuple{
			{index: 0, option: []byte(`foo`), attribute: []byte(`a`), value: []byte(`1`)},
			{index: 0, option: []byte(`foo`), attribute: []byte(`b`), value: []byte(`2`)},
			{index: 0, option: []byte(`foo`), attribute: []byte(`c`), value: []byte(`3`)},
			{index: 1, option: []byte(`bar`), attribute: []byte(`z`)},
			{index: 2, option: []byte(`baz`)},
		},
	},
	{
		label: "comma",
		in:    []byte(`foo;a=1,, , ,bar;b=2`),
		ok:    true,
		exp: []tuple{
			{index: 0, option: []byte(`foo`), attribute: []byte(`a`), value: []byte(`1`)},
			{index: 1, option: []byte(`bar`), attribute: []byte(`b`), value: []byte(`2`)},
		},
	},
}

func TestParameters(t *testing.T) {
	for _, test := range parametersCases {
		t.Run(test.label, func(t *testing.T) {
			var act []tuple

			ok := ScanOptions(test.in, func(index int, key, param, value []byte) Control {
				act = append(act, tuple{index, key, param, value})
				return ControlContinue
			})

			if ok != test.ok {
				t.Errorf("unexpected result: %v; want %v", ok, test.ok)
			}
			if an, en := len(act), len(test.exp); an != en {
				t.Errorf("unexpected length of result: %d; want %d", an, en)
				return
			}

			for i, e := range test.exp {
				a := act[i]

				if a.index != e.index || !bytes.Equal(a.option, e.option) || !bytes.Equal(a.attribute, e.attribute) || !bytes.Equal(a.value, e.value) {
					t.Errorf(
						"unexpected %d-th tuple: #%d %#q[%#q = %#q]; want #%d %#q[%#q = %#q]",
						i,
						a.index, string(a.option), string(a.attribute), string(a.value),
						e.index, string(e.option), string(e.attribute), string(e.value),
					)
				}
			}
		})
	}
}

func BenchmarkParameters(b *testing.B) {
	for _, bench := range parametersCases {
		b.Run(bench.label, func(b *testing.B) {
			for i := 0; i < b.N; i++ {
				_ = ScanOptions(bench.in, func(_ int, _, _, _ []byte) Control { return ControlContinue })
			}
		})
	}
}
