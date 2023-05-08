package ping

import "testing"

func TestTCPing(t *testing.T) {
	res, err := TCPing("www.baidu.com:80", 2)
	if err != nil {
		t.Fatal(err)
	}
	t.Log(res)
}
