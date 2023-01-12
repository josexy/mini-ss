package main

var (
	supportCipherMethods = []string{
		"none",
		"aes-128-ctr",
		"aes-192-ctr",
		"aes-256-ctr",
		"aes-128-cfb",
		"aes-192-cfb",
		"aes-256-cfb",
		"bf-cfb",
		"rc4-md5",
		"salsa20",
		"chacha20",
		"chacha20-ietf",
		"aes-128-gcm",
		"aes-192-gcm",
		"aes-256-gcm",
		"chacha20-ietf-poly1305",
		"xchacha20-ietf-poly1305",
	}

	supportTransportTypes = []string{
		"default",
		"ws",
		"kcp",
		"quic",
		"obfs",
	}

	supportKcpCrypts = []string{
		"none",
		"sm4",
		"tea",
		"xor",
		"aes-128",
		"aes-192",
		"aes-256",
		"blowfish",
		"twofish",
		"cast5",
		"3des",
		"xtea",
		"salsa20",
	}

	supportKcpModes = []string{
		"normal",
		"fast1",
		"fast2",
		"fast3",
	}
)
