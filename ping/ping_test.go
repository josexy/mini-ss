package ping

import "testing"

func TestPingWithoutRoot(t *testing.T) {
	t.Log(PingWithoutRoot("www.baidu.com", 2))
	t.Log(PingWithoutRoot("www.example.com", 2))
	t.Log(PingWithoutRoot("127.0.0.1", 2))
	t.Log(PingWithoutRoot("192.168.1.1", 2))
	t.Log(PingWithoutRoot("8.8.8.8", 2))
}
