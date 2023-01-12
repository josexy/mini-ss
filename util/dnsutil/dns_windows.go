package dnsutil

import (
	"fmt"
	"net"
	"syscall"
	"unsafe"

	"github.com/josexy/logx"
	"github.com/josexy/mini-ss/util"
)

var oldTunName string

func SetLocalDnsServer(tunName string) {
	oldTunName = tunName
	cmd := `netsh interface ipv4 add dnsserver "` + tunName + `" 127.0.0.1 index=1`
	if err := util.ExeCmd(cmd); err != nil {
		logx.ErrorBy(err)
	}
}

func UnsetLocalDnsServer() {
	cmd := `netsh interface ipv4 delete dnsservers "` + oldTunName + `" all`
	util.ExeCmd(cmd)
}

const MAX_HOSTNAME_LEN = 128
const MAX_DOMAIN_NAME_LEN = 128
const MAX_SCOPE_ID_LEN = 256
const ValueOverflow = 11

type DWORD uint32
type CHAR byte
type UINT uint32
type IP_ADDRESS_STRING struct{ String [4 * 4]CHAR }
type IP_MASK_STRING struct{ String [4 * 4]CHAR }
type PIP_ADDR_STRING *IP_ADDR_STRING
type IP_ADDR_STRING struct {
	Next      *IP_ADDR_STRING
	IpAddress IP_ADDRESS_STRING
	IpMask    IP_MASK_STRING
	Context   DWORD
}
type FIXED_INFO_W2KSP1 struct {
	HostName         [MAX_HOSTNAME_LEN + 4]CHAR
	DomainName       [MAX_DOMAIN_NAME_LEN + 4]CHAR
	CurrentDnsServer PIP_ADDR_STRING
	DnsServerList    IP_ADDR_STRING
	NodeType         UINT
	ScopeId          [MAX_SCOPE_ID_LEN + 4]CHAR
	EnableRouting    UINT
	EnableProxy      UINT
	EnableDns        UINT
}
type PFIXED_INFO *FIXED_INFO_W2KSP1

/*
IPHLPAPI_DLL_LINKAGE DWORD GetNetworkParams(

	[out] PFIXED_INFO pFixedInfo,
	[in]  PULONG      pOutBufLen

);
*/
func GetLocalDnsList() []string {
	var iphlpapi = syscall.NewLazyDLL("Iphlpapi.dll")
	var getNetworkParams = iphlpapi.NewProc("GetNetworkParams")

	info := FIXED_INFO_W2KSP1{}
	size := uint32(unsafe.Sizeof(info))
	r, _, _ := getNetworkParams.Call(uintptr(unsafe.Pointer(&info)), uintptr(unsafe.Pointer(&size)))
	var dns []string
	if r == 0 {
		for ai := &info.DnsServerList; ai != nil; ai = ai.Next {
			d := fmt.Sprintf("%v.%v.%v.%v", ai.Context&0xFF, (ai.Context>>8)&0xFF, (ai.Context>>16)&0xFF, (ai.Context>>24)&0xFF)
			dns = append(dns, net.JoinHostPort(d, "53"))
		}
	} else if r == ValueOverflow {
		newBuffers := make([]byte, size)
		netParams := (PFIXED_INFO)(unsafe.Pointer(&newBuffers[0]))
		getNetworkParams.Call(uintptr(unsafe.Pointer(&netParams)), uintptr(unsafe.Pointer(&size)))
		for ai := &netParams.DnsServerList; ai != nil; ai = ai.Next {
			d := fmt.Sprintf("%v.%v.%v.%v", ai.Context&0xFF, (ai.Context>>8)&0xFF, (ai.Context>>16)&0xFF, (ai.Context>>24)&0xFF)
			dns = append(dns, net.JoinHostPort(d, "53"))
		}
	}
	return dns
}
