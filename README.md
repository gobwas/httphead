# httphead.[go](https://golang.org)

[![GoDoc][godoc-image]][godoc-url] 

> Tiny HTTP header value parsing library in go.

## Install

```shell
    go get github.com/gobwas/glob
```

## Example

The example below shows how multiple-choise HTTP header value could be parsed with this library:

```go
	type pair struct {
		Key, Value string
	}

	// The right part of full header line like:
	//
	// X-My-Header: key;foo=bar;baz,key;baz
	//
	header := []byte(`key;foo=bar;baz,key;baz`)

	choises := make([][]pair, 2)
	Parameters(header, func(i int, key, param, value []byte) Control {
		choises[i] = append(choises[i], pair{string(param), string(value)})
		return ControlContinue
	})

	fmt.Println(choises)
	// Output: [[{foo bar} {baz }] [{baz }]]
```

For more usage examples please see [docs](godoc-url) or package tests.

[godoc-image]: https://godoc.org/github.com/gobwas/httphead?status.svg
[godoc-url]: https://godoc.org/github.com/gobwas/httphead
[travis-image]: https://travis-ci.org/gobwas/httphead.svg?branch=master
[travis-url]: https://travis-ci.org/gobwas/httphead
