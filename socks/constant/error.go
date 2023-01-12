package constant

import "errors"

var (
	ErrSerializeFailure    = errors.New("socks serialized data packet format invalid")
	ErrVersion5Invalid     = errors.New("socks version not 0x05")
	ErrVersion1Invalid     = errors.New("socks version not 0x01")
	ErrUnsupportedMethod   = errors.New("socks unsupported method")
	ErrUnsupportedReqCmd   = errors.New("socks unsupported request cmd")
	ErrUnsupportedReqAType = errors.New("socks unsupported request address type")
	ErrAuthFailure         = errors.New("socks authentication failure")
	ErrRequestFailure      = errors.New("socks request failure")
)

var (
	ErrRuleMatchDropped = errors.New("rule match: dropped")
)
