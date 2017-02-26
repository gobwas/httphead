package httphead

import (
	"bytes"
	"fmt"
	"testing"
)

func ExampleList() {
	var values []string
	List([]byte(`a,b,c`), func(v []byte) bool {
		values = append(values, string(v))
		return v[0] != 'b'
	})

	fmt.Println(values)
	// Output: [a b]
}

func ExampleParameters() {
	foo := map[string]string{}
	Parameters([]byte(`foo;bar=1;baz`), func(key, param, value []byte) bool {
		if bytes.Equal(key, []byte("foo")) {
			foo[string(param)] = string(value)
		}
		return true
	})

	fmt.Println(foo)
	// Output: map[bar:1 baz:]
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

func TestList(t *testing.T) {
	for _, test := range listCases {
		t.Run(test.label, func(t *testing.T) {
			var act [][]byte
			ok := List(test.in, func(v []byte) bool {
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

func BenchmarkList(b *testing.B) {
	for _, bench := range listCases {
		b.Run(bench.label, func(b *testing.B) {
			for i := 0; i < b.N; i++ {
				_ = List(bench.in, func(v []byte) bool { return true })
			}
		})
	}
}

type tuple struct {
	key, param, value []byte
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
			{key: []byte(`a`)},
			{key: []byte(`b`)},
			{key: []byte(`c`)},
		},
	},
	{
		label: "simple",
		in:    []byte(`a,b,c;foo=1;bar=2`),
		ok:    true,
		exp: []tuple{
			{key: []byte(`a`)},
			{key: []byte(`b`)},
			{key: []byte(`c`), param: []byte(`foo`), value: []byte(`1`)},
			{key: []byte(`c`), param: []byte(`bar`), value: []byte(`2`)},
		},
	},
	{
		label: "simple",
		in:    []byte(`c;foo;bar=2`),
		ok:    true,
		exp: []tuple{
			{key: []byte(`c`), param: []byte(`foo`)},
			{key: []byte(`c`), param: []byte(`bar`), value: []byte(`2`)},
		},
	},
	{
		label: "simple",
		in:    []byte(`foo;bar=1;baz`),
		ok:    true,
		exp: []tuple{
			{key: []byte(`foo`), param: []byte(`bar`), value: []byte(`1`)},
			{key: []byte(`foo`), param: []byte(`baz`)},
		},
	},
	{
		label: "simple_quoted",
		in:    []byte(`c;bar="2"`),
		ok:    true,
		exp: []tuple{
			{key: []byte(`c`), param: []byte(`bar`), value: []byte(`2`)},
		},
	},
	{
		label: "all",
		in:    []byte(`foo;a=1;b=2;c=3,bar;z,baz`),
		ok:    true,
		exp: []tuple{
			{key: []byte(`foo`), param: []byte(`a`), value: []byte(`1`)},
			{key: []byte(`foo`), param: []byte(`b`), value: []byte(`2`)},
			{key: []byte(`foo`), param: []byte(`c`), value: []byte(`3`)},
			{key: []byte(`bar`), param: []byte(`z`)},
			{key: []byte(`baz`)},
		},
	},
	{
		label: "comma",
		in:    []byte(`foo;a=1,, , ,bar;b=2`),
		ok:    true,
		exp: []tuple{
			{key: []byte(`foo`), param: []byte(`a`), value: []byte(`1`)},
			{key: []byte(`bar`), param: []byte(`b`), value: []byte(`2`)},
		},
	},
}

func TestParameters(t *testing.T) {
	for _, test := range parametersCases {
		t.Run(test.label, func(t *testing.T) {
			var act []tuple

			ok := Parameters(test.in, func(key, param, value []byte) bool {
				act = append(act, tuple{key, param, value})
				return true
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

				if !bytes.Equal(a.key, e.key) || !bytes.Equal(a.param, e.param) || !bytes.Equal(a.value, e.value) {
					t.Errorf(
						"unexpected %d-th tuple: %#q[%#q = %#q]; want %#q[%#q = %#q]",
						i, string(a.key), string(a.param), string(a.value), string(e.key), string(e.param), string(e.value),
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
				_ = Parameters(bench.in, func(_, _, _ []byte) bool { return true })
			}
		})
	}
}
