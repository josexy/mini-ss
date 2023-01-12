package address

import "testing"

var testAddress = []string{
	"www.baidu.com:80",
	"127.0.0.1:10086",
}

func TestParseAddress(t *testing.T) {
	for _, addr := range testAddress {
		t.Log(ParseAddress1(addr).String())
	}
}
