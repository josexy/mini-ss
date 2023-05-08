package main

var (
	event_traffic_speed    = "event-traffic-speed"
	event_traffic_snapshot = "event-traffic-snapshot"

	supportTransportList = []string{
		"default",
		"grpc",
		"kcp",
		"obfs",
		"quic",
		"ws",
	}

	supportMethodList = []string{
		"none",
		"aes-128-ctr",
		"aes-192-ctr",
		"aes-256-ctr",
		"aes-128-cfb",
		"aes-192-cfb",
		"aes-256-cfb",
		"bf-cfb",
		"salsa20",
		"rc4-md5",
		"chacha20",
		"chacha20-ietf",
		"aes-128-gcm",
		"aes-192-gcm",
		"aes-256-gcm",
		"chacha20-ietf-poly1305",
		"xchacha20-ietf-poly1305",
	}

	supportKcpModeList = []string{
		"normal",
		"fast",
		"fast2",
		"fast3",
	}

	supportKcpCryptList = []string{
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

	supportSSRObfsList = []string{
		"plain",
		"http_simple",
		"http_post",
		"random_head",
		"tls1.2_ticket_auth",
		"tls1.2_ticket_fastauth",
	}

	supportSSRProtocolList = []string{
		"origin",
		"auth_aes128_md5",
		"auth_aes128_sha1",
		"auth_chain_a",
		"auth_chain_b",
		"auth_sha1_v4",
	}
)
