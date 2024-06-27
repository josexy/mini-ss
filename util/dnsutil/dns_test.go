package dnsutil

import (
	"testing"
)

func TestGetLocalDnsList(t *testing.T) {
	t.Log(GetLocalDnsList())
}
