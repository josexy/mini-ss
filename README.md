
# mini-ss

Mini shadowsocks server and client in Golang

[![Go Report Card](https://goreportcard.com/badge/github.com/josexy/mini-ss)](https://goreportcard.com/report/github.com/josexy/mini-ss)
[![License](https://img.shields.io/github/license/josexy/mini-ss)](https://github.com/josexy/mini-ss/blob/main/LICENSE)
[![Publish Go Releases](https://github.com/josexy/mini-ss/actions/workflows/go-release.yml/badge.svg)](https://github.com/josexy/mini-ss/actions/workflows/go-release.yml)

## Usage

### Clone

```shell
git clone https://github.com/josexy/mini-ss
cd mini-ss && make all
# copy Country.mmdb
cp Country.mmdb bin/ && cd bin
# help
./mini-ss-darwin-arm64 -h
./mini-ss-darwin-arm64 client -h
./mini-ss-darwin-arm64 server -h
```

### Client

```shell
# help
./mini-ss-darwin-arm64 client -h

# simple
./mini-ss-darwin-arm64 client -s 127.0.0.1:8388 -l :10086 -x :10087 -m aes-128-cfb -p 123456 -CV3

# udp relay if support
./mini-ss-darwin-arm64 client -s 127.0.0.1:8388 -M 127.0.0.1:10088 -m aes-128-cfb -p 123456 -CV3 --udp-relay 

# enable tun mode
sudo ./mini-ss-darwin-arm64 client -s 127.0.0.1:8388 -M :10088 -m aes-128-cfb -p 123456 -CV3 --tun-enable --auto-detect-iface

# ssr client
./mini-ss-darwin-arm64 client -s server:port -M 127.0.0.1:10088 -m aes-256-cfb -p 123456 -t default -o tls1.2_ticket_auth -O auth_chain_a -T ssr -CV3 --system-proxy

# load from config file
./mini-ss-darwin-arm64 client -c ../example-configs/simple-client-config.yaml
```

### Server

```shell
# help
./mini-ss-darwin-arm64 server -h

# simple
./mini-ss-darwin-arm64 server -s :8388 -m aes-128-cfb -p 123456 -CV3 --udp-relay --auto-detect-iface

# load from config file
./mini-ss-darwin-arm64 server -c ../example-configs/simple-server-config.yaml
```

You can find the test configuration from `example-configs`

## Rules

- GLOBAL
- DIRECT
- MATCH
  - DOMAIN
  - DOMAIN-KEYWORD
  - DOMAIN-SUFFIX
  - GEOIP
  - IP-CIDR
  - OTHERS

## References

- https://github.com/shadowsocks/go-shadowsocks2
- https://github.com/Dreamacro/clash
- https://github.com/ginuerzh/gost
- https://github.com/xjasonlyu/tun2socks
- https://github.com/josexy/netstackgo
