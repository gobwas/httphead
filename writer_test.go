package httphead

import (
	"bufio"
	"bytes"
	"reflect"
	"strings"
	"testing"
)

func TestWriteOptions(t *testing.T) {
	for _, test := range []struct {
		options []Option
		exp     string
	}{
		{
			options: []Option{
				{"foo", map[string]string{"bar": "baz"}},
			},
			exp: "foo;bar=baz",
		},
		{
			options: []Option{
				{"foo", map[string]string{"bar": "baz"}},
				{"a", nil},
				{"b", map[string]string{"c": "10"}},
			},
			exp: "foo;bar=baz,a,b;c=10",
		},
		{
			options: []Option{
				{"foo", map[string]string{"a b c": "10,2"}},
			},
			exp: `foo;"a\ b\ c"="10\,2"`,
		},
	} {
		buf := bytes.Buffer{}
		bw := bufio.NewWriter(&buf)

		WriteOptions(bw, test.options)

		if err := bw.Flush(); err != nil {
			t.Fatal(err)
		}
		if act := buf.String(); act != test.exp {
			t.Errorf("WriteOptions = %#q; want %#q", act, test.exp)
		}
	}
}

func TestSanitize(t *testing.T) {
	for _, test := range []struct {
		in  string
		out []byte
	}{
		{"hello-world", nil},
		{"hello, world!", []byte(`"hello\,\ world!"`)},
		{"a,b,c,d,e,f,g,h,i,j!", []byte(`"a\,b\,c\,d\,e\,f\,g\,h\,i\,j!"`)},
		{
			strings.Repeat(",", 7),
			[]byte(`"` + strings.Repeat(`\,`, 7) + `"`),
		},
		{
			strings.Repeat(",", 10),
			[]byte(`"` + strings.Repeat(`\,`, 10) + `"`),
		},
	} {
		t.Run(test.in, func(t *testing.T) {
			act := sanitize(test.in)
			if !reflect.DeepEqual(act, test.out) {
				t.Errorf("sanitize(%#q) = %#q; want %#q", test.in, string(act), test.out)
			}
		})
	}
}

func BenchmarkSanitize(b *testing.B) {
	for _, test := range []string{
		"hello-world",
		"hello, world!",
		"a,b,c,d,e,f,g,h,i,j!",
	} {
		b.Run(test, func(b *testing.B) {
			for i := 0; i < b.N; i++ {
				_ = sanitize(test)
			}
		})
	}
}
