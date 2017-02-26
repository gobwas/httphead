package httphead

import (
	"fmt"
	"testing"
)

func TestHeaderList(t *testing.T) {
	ok := HeaderList([]byte(`a,b,c`), func(v []byte) bool {
		fmt.Println(string(v))
		return true
	})
	fmt.Println("ok", ok)
}
