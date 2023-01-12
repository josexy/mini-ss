package ping

import (
	"testing"
	"time"
)

func TestSpeedTest(t *testing.T) {
	sp := NewSpeedTest()
	sp.SetTestProxy(socksMode, "", "127.0.0.1:10086")
	go func() {
		time.Sleep(time.Second * 10)
		t.Log("stop speed test by user")
		sp.Stop()
	}()
	err := sp.Start(10 * time.Minute)
	if err != nil {
		t.Fatal(err)
	}
	t.Log(sp.Status())
}
