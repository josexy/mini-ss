package protocol

import "github.com/josexy/mini-ss/ssr/tools"

func newAuthAES128MD5(b *Base) Protocol {
	a := &authAES128{
		Base:               b,
		authData:           &authData{},
		authAES128Function: &authAES128Function{salt: "auth_aes128_md5", hmac: tools.HmacMD5, hashDigest: tools.MD5Sum},
		userData:           &userData{},
	}
	a.initUserData()
	return a
}
