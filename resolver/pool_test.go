package resolver

import (
	"testing"
)

var (
	cidr = "10.1.0.1/16"
	pool = newFakeIPPool(cidr)
)

func TestDnsIPPool_Alloc(t *testing.T) {
	t.Log(pool.Alloc("www.google.com"))
	t.Log(pool.Alloc("www.example.com"))
	t.Log(pool.Alloc("www.baidu.com"))
}
