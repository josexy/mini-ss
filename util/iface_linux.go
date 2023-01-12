package util

func ResolveDefaultRouteInterface() (string, error) {
	return ExeShell("route -n | grep 'UG[ \t]' | awk 'NR==1{print $8}'")
}
