package httphead

import (
	"bufio"
	"bytes"
	"fmt"
	"testing"
)

func ExampleWriteOptions() {
	opts := []Option{
		{"foo", map[string]string{
			"param": "hello, world!",
		}},
		{"bar", nil},
		{"b a z", nil},
	}

	buf := bytes.Buffer{}
	bw := bufio.NewWriter(&buf)

	WriteOptions(bw, opts)
	bw.Flush()

	// Output: foo;param="hello, world!",bar,"b a z"
	fmt.Println(buf.String())
}

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
			exp: `foo;"a b c"="10,2"`,
		},
		{
			options: []Option{
				{`"foo"`, nil},
				{`"bar"`, nil},
			},
			exp: `"\"foo\"","\"bar\""`,
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
