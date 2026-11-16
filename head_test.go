package httphead

import (
	"bytes"
	"strconv"
	"strings"
	"testing"
)

func TestParseRequestLine(t *testing.T) {
	for _, test := range []struct {
		name string
		line string
		exp  RequestLine
		fail bool
	}{
		{
			line: "",
			fail: true,
		},
		{
			line: "GET",
			fail: true,
		},
		{
			line: "GET ",
			fail: true,
		},
		{
			line: "GET  ",
			fail: true,
		},
		{
			line: "GET   ",
			fail: true,
		},
		{
			line: "GET / HTTP/1.1",
			exp: RequestLine{
				Method:  []byte("GET"),
				URI:     []byte("/"),
				Version: Version{1, 1},
			},
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			r, ok := ParseRequestLine([]byte(test.line))
			if test.fail && ok {
				t.Fatalf("unexpected successful parsing")
			}
			if !test.fail && !ok {
				t.Fatalf("unexpected parse error")
			}
			if test.fail {
				return
			}
			if act, exp := r.Method, test.exp.Method; !bytes.Equal(act, exp) {
				t.Errorf("unexpected parsed method: %q; want %q", act, exp)
			}
			if act, exp := r.URI, test.exp.URI; !bytes.Equal(act, exp) {
				t.Errorf("unexpected parsed uri: %q; want %q", act, exp)
			}
			if act, exp := r.Version, test.exp.Version; act != exp {
				t.Errorf("unexpected parsed version: %+v; want %+v", act, exp)
			}
		})
	}
}

func TestParseResponseLine(t *testing.T) {
	for _, test := range []struct {
		name string
		line string
		exp  ResponseLine
		fail bool
	}{
		{
			line: "",
			fail: true,
		},
		{
			line: "HTTP/1.1",
			fail: true,
		},
		{
			line: "HTTP/1.1 ",
			fail: true,
		},
		{
			line: "HTTP/1.1  ",
			fail: true,
		},
		{
			line: "HTTP/1.1   ",
			fail: true,
		},
		{
			line: "HTTP/1.1 200 OK",
			exp: ResponseLine{
				Version: Version{1, 1},
				Status:  200,
				Reason:  []byte("OK"),
			},
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			r, ok := ParseResponseLine([]byte(test.line))
			if test.fail && ok {
				t.Fatalf("unexpected successful parsing")
			}
			if !test.fail && !ok {
				t.Fatalf("unexpected parse error")
			}
			if test.fail {
				return
			}
			if act, exp := r.Version, test.exp.Version; act != exp {
				t.Errorf("unexpected parsed version: %+v; want %+v", act, exp)
			}
			if act, exp := r.Status, test.exp.Status; act != exp {
				t.Errorf("unexpected parsed status: %d; want %d", act, exp)
			}
			if act, exp := r.Reason, test.exp.Reason; !bytes.Equal(act, exp) {
				t.Errorf("unexpected parsed reason: %q; want %q", act, exp)
			}
		})
	}
}

var versionCases = []struct {
	in    []byte
	major int
	minor int
	ok    bool
}{
	{[]byte("HTTP/1.1"), 1, 1, true},
	{[]byte("HTTP/1.0"), 1, 0, true},
	{[]byte("HTTP/1.2"), 1, 2, true},
	{[]byte("HTTP/42.1092"), 42, 1092, true},
}

func TestParseHttpVersion(t *testing.T) {
	for _, c := range versionCases {
		t.Run(string(c.in), func(t *testing.T) {
			major, minor, ok := ParseVersion(c.in)
			if major != c.major || minor != c.minor || ok != c.ok {
				t.Errorf(
					"parseHttpVersion([]byte(%q)) = %v, %v, %v; want %v, %v, %v",
					string(c.in), major, minor, ok, c.major, c.minor, c.ok,
				)
			}
		})
	}
}

func BenchmarkParseHttpVersion(b *testing.B) {
	for _, c := range versionCases {
		b.Run(string(c.in), func(b *testing.B) {
			for i := 0; i < b.N; i++ {
				_, _, _ = ParseVersion(c.in)
			}
		})
	}
}

func TestIntFromASCII(t *testing.T) {
	max := int(^uint(0) >> 1)
	for _, test := range []struct {
		input string
		want  int
		ok    bool
	}{
		{"", 0, false},
		{"0", 0, true},
		{"1234567890", 1234567890, true},
		{"00000101", 101, true},
		{strings.Repeat("0", 100), 0, true},
		{strconv.Itoa(max), max, true},
		{"000" + strconv.Itoa(max), max, true},
		{strconv.FormatUint(uint64(max)+1, 10), 0, false},
		{strings.Repeat("9", 100), 0, false},
		{"-1", 0, false},
		{"+1", 0, false},
		{"1 ", 0, false},
		{"1.0", 0, false},
		{"1/0", 0, false},
		{"1@0", 0, false},
	} {
		t.Run(test.input, func(t *testing.T) {
			got, ok := IntFromASCII([]byte(test.input))
			if got != test.want || ok != test.ok {
				t.Fatalf("IntFromASCII(%q) = %d, %t; want %d, %t", test.input, got, ok, test.want, test.ok)
			}
		})
	}
	for c := byte(':'); c <= '?'; c++ {
		input := []byte{'1', c, '0'}
		if got, ok := IntFromASCII(input); got != 0 || ok {
			t.Errorf("IntFromASCII(%q) = %d, %t; want 0, false", input, got, ok)
		}
	}
}

func TestParseLinesRejectInvalidDecimalFields(t *testing.T) {
	invalid := []string{"1:", "1;", "1<", "1=", "1>", "1?", strconv.FormatUint(uint64(^uint(0)>>1)+1, 10)}
	for _, value := range invalid {
		t.Run(value, func(t *testing.T) {
			for _, version := range []string{"HTTP/" + value + ".1", "HTTP/1." + value} {
				if _, _, ok := ParseVersion([]byte(version)); ok {
					t.Errorf("ParseVersion(%q) succeeded", version)
				}
				request := "GET /resource?q=value " + version
				if _, ok := ParseRequestLine([]byte(request)); ok {
					t.Errorf("ParseRequestLine(%q) succeeded", request)
				}
				response := version + " 200 OK"
				if _, ok := ParseResponseLine([]byte(response)); ok {
					t.Errorf("ParseResponseLine(%q) succeeded", response)
				}
			}
			response := "HTTP/1.1 " + value + " reason"
			if _, ok := ParseResponseLine([]byte(response)); ok {
				t.Errorf("ParseResponseLine(%q) succeeded", response)
			}
		})
	}
}

func TestParseLinesPreserveDecimalFields(t *testing.T) {
	request, ok := ParseRequestLine([]byte("GET /resource?q=value HTTP/42.1092"))
	if !ok || string(request.Method) != "GET" || string(request.URI) != "/resource?q=value" || request.Version != (Version{42, 1092}) {
		t.Fatalf("unexpected request: %+v, %t", request, ok)
	}
	response, ok := ParseResponseLine([]byte("HTTP/01.002 00101 Switching Protocols"))
	if !ok || response.Version != (Version{1, 2}) || response.Status != 101 || string(response.Reason) != "Switching Protocols" {
		t.Fatalf("unexpected response: %+v, %t", response, ok)
	}
}
