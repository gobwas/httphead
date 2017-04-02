package httphead

import (
	"bytes"
	"fmt"
	"math/rand"
	"sort"
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
	// Output: [{foo [bar:1]} {baz []}] true
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

func TestOptionCopy(t *testing.T) {
	for i, test := range []struct {
		pairs int
	}{
		{4},
		{16},
	} {

		name := []byte(fmt.Sprintf("test:%d", i))
		n := make([]byte, len(name))
		copy(n, name)
		opt := Option{Name: n}

		pairs := make([]pair, test.pairs)
		for i := 0; i < len(pairs); i++ {
			pair := pair{make([]byte, 8), make([]byte, 8)}
			randAscii(pair.key)
			randAscii(pair.value)
			pairs[i] = pair

			k, v := make([]byte, len(pair.key)), make([]byte, len(pair.value))
			copy(k, pair.key)
			copy(v, pair.value)

			opt.Parameters.Set(k, v)
		}

		cp := opt.Copy()

		memset(opt.Name, 'x')
		for _, p := range opt.Parameters.data() {
			memset(p.key, 'x')
			memset(p.value, 'x')
		}

		if !bytes.Equal(cp.Name, name) {
			t.Errorf("name was not copied properly: %q; want %q", string(cp.Name), string(name))
		}
		for i, p := range cp.Parameters.data() {
			exp := pairs[i]
			if !bytes.Equal(p.key, exp.key) || !bytes.Equal(p.value, exp.value) {
				t.Errorf(
					"%d-th pair was not copied properly: %q=%q; want %q=%q",
					i, string(p.key), string(p.value), string(exp.key), string(exp.value),
				)
			}
		}
	}
}

func memset(dst []byte, v byte) {
	copy(dst, bytes.Repeat([]byte{v}, len(dst)))
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

func randAscii(dst []byte) {
	for i := 0; i < len(dst); i++ {
		dst[i] = byte(rand.Intn('z'-'a')) + 'a'
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

var selectOptionsCases = []struct {
	label string
	flags SelectFlags
	in    []byte
	p     []Option
	check func(Option) bool
	exp   []Option
	ok    bool
}{
	{
		label: "simple",
		in:    []byte(`foo;a=1,foo;a=2`),
		p:     nil,
		flags: SelectCopy | SelectUnique,
		check: func(opt Option) bool { return true },
		exp: []Option{
			NewOption("foo", map[string]string{"a": "1"}),
		},
		ok: true,
	},
	{
		label: "simple_no_alloc",
		in:    []byte(`foo;a=1,foo;a=2`),
		p:     make([]Option, 0, 2),
		flags: SelectUnique,
		check: func(opt Option) bool { return true },
		exp: []Option{
			NewOption("foo", map[string]string{"a": "1"}),
		},
		ok: true,
	},
	{
		label: "multiparam_stack",
		in:    []byte(`foo;a=1;b=2;c=3;d=4;e=5;f=6;g=7;h=8,bar`),
		p:     make([]Option, 0, 2),
		flags: SelectUnique,
		check: func(opt Option) bool { return true },
		exp: []Option{
			NewOption("foo", map[string]string{
				"a": "1",
				"b": "2",
				"c": "3",
				"d": "4",
				"e": "5",
				"f": "6",
				"g": "7",
				"h": "8",
			}),
			NewOption("bar", nil),
		},
		ok: true,
	},
	{
		label: "multiparam_stack",
		in:    []byte(`foo;a=1;b=2;c=3;d=4;e=5;f=6;g=7;h=8,bar`),
		p:     make([]Option, 0, 2),
		flags: SelectUnique | SelectCopy,
		check: func(opt Option) bool { return true },
		exp: []Option{
			NewOption("foo", map[string]string{
				"a": "1",
				"b": "2",
				"c": "3",
				"d": "4",
				"e": "5",
				"f": "6",
				"g": "7",
				"h": "8",
			}),
			NewOption("bar", nil),
		},
		ok: true,
	},
	{
		label: "multiparam_heap",
		in:    []byte(`foo;a=1;b=2;c=3;d=4;e=5;f=6;g=7;h=8;i=9;j=10,bar`),
		p:     make([]Option, 0, 2),
		flags: SelectUnique | SelectCopy,
		check: func(opt Option) bool { return true },
		exp: []Option{
			NewOption("foo", map[string]string{
				"a": "1",
				"b": "2",
				"c": "3",
				"d": "4",
				"e": "5",
				"f": "6",
				"g": "7",
				"h": "8",
				"i": "9",
				"j": "10",
			}),
			NewOption("bar", nil),
		},
		ok: true,
	},
}

func TestSelectOptions(t *testing.T) {
	for _, test := range selectOptionsCases {
		t.Run(test.label+test.flags.String(), func(t *testing.T) {
			act, ok := SelectOptions(test.flags, test.in, test.p, test.check)
			if ok != test.ok {
				t.Errorf("SelectOptions(%q) wellformed sign is %v; want %v", string(test.in), ok, test.ok)
			}
			if !optionsEqual(act, test.exp) {
				t.Errorf("SelectOptions(%q) = %v; want %v", string(test.in), act, test.exp)
			}
		})
	}
}

func BenchmarkSelectOptions(b *testing.B) {
	for _, test := range selectOptionsCases {
		b.Run(test.label+test.flags.String(), func(b *testing.B) {
			for i := 0; i < b.N; i++ {
				_, _ = SelectOptions(test.flags, test.in, test.p, test.check)
			}
		})
	}
}

func optionsEqual(a, b []Option) bool {
	if len(a) != len(b) {
		return false
	}
	for i := 0; i < len(a); i++ {
		if !optionEqual(a[i], b[i]) {
			return false
		}
	}
	return true
}

func optionEqual(a, b Option) bool {
	if bytes.Equal(a.Name, b.Name) {
		return paramEqual(a.Parameters, b.Parameters)
	}
	return false
}

type pairs []pair

func (p pairs) Len() int           { return len(p) }
func (p pairs) Less(a, b int) bool { return bytes.Compare(p[a].key, p[b].key) == -1 }
func (p pairs) Swap(a, b int)      { p[a], p[b] = p[b], p[a] }

func paramEqual(a, b Parameters) bool {
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
