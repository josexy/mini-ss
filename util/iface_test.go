package util

import (
	"testing"
)

func TestResolveAllInterfaces(t *testing.T) {
	ResolveAllInterfaces()
	for _, iface := range res {
		t.Log(iface)
	}
}

func TestResolveDefaultRouteInterface(t *testing.T) {
	ifaceName, err := ResolveDefaultRouteInterface()
	if err != nil {
		t.Fatal(err)
	}
	t.Log(ifaceName)
}
