package address

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/assert"
)

type testcase struct {
	host   string
	port   int
	expect string
	buffer []byte
	msg    string
}

var testcaseSucceed = []testcase{
	{"www.example.com", 80, "www.example.com:80", make([]byte, 19), "domain name test ok"},
	{"123.12.12.123", 10086, "123.12.12.123:10086", make([]byte, 7), "ipv4 test ok"},
	{"12:12::aa", 10086, "[12:12::aa]:10086", make([]byte, 19), "ipv6 test ok"},
}

var testcaseFailed = []testcase{
	{"www.example.com", 80, "", make([]byte, 18), "domain name short buffer test fail"},
	{"123.12.12.123", 10086, "", make([]byte, 6), "ipv4 short buffer test fail"},
	{"12:12::aa", 10086, "", make([]byte, 18), "ipv6 short buffer test fail"},

	{"www.example.com", 80, "", make([]byte, 0), "domain name short buffer test fail"},
	{"123.12.12.123", 10086, "", make([]byte, 0), "ipv4 short buffer test fail"},
	{"12:12::aa", 10086, "", make([]byte, 0), "ipv6 short buffer test fail"},

	{"", 1000, "", make([]byte, 19), "invalid host test fail"},
	{"localhost", -1000, "", make([]byte, 19), "invalid port test fail"},
}

func TestParseAddress(t *testing.T) {
	for _, tc := range testcaseSucceed {
		addr, err := ParseAddressFromHostPort(tc.host, tc.port, tc.buffer)
		assert.Nil(t, err)
		assert.Equal(t, tc.expect, addr.String())

		addr, err = ParseAddressFromBuffer(addr)
		assert.Nil(t, err)
		assert.Equal(t, tc.expect, addr.String())

		addr, err = ParseAddressFromReader(bytes.NewReader(addr), tc.buffer)
		assert.Nil(t, err)
		assert.Equal(t, tc.expect, addr.String())
		t.Log(tc.msg)
	}

	for _, tc := range testcaseFailed {
		addr, err := ParseAddressFromHostPort(tc.host, tc.port, tc.buffer)
		assert.NotNil(t, err)
		assert.NotEqual(t, tc.expect, addr.String())

		addr, err = ParseAddressFromBuffer(addr)
		assert.NotNil(t, err)
		assert.NotEqual(t, tc.expect, addr.String())

		addr, err = ParseAddressFromReader(bytes.NewReader(addr), tc.buffer)
		assert.NotNil(t, err)
		assert.NotEqual(t, tc.expect, addr.String())
		t.Log(tc.msg)
	}
}
