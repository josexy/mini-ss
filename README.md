
# mini-ss

Mini shadowsocks server and client in Golang 

[![Go Report Card](https://goreportcard.com/badge/github.com/josexy/mini-ss)](https://goreportcard.com/report/github.com/josexy/mini-ss)
[![License](https://img.shields.io/github/license/josexy/mini-ss)](https://github.com/josexy/mini-ss/blob/main/LICENSE)


## Usage

compile

```shell
git clone https://github.com/josexy/mini-ss
cd mini-ss && make all
# copy Country.mmdb
cp build/Country.mmdb bin/ && cd bin
# help
./mini-ss-macos-arm64 -h
```

client

```shell
# help
./mini-ss-macos-arm64 client -h

# simple
./mini-ss-macos-arm64 client -s 127.0.0.1:8388 -l :10086 -x :10087 -m aes-128-cfb -p 123456 -CV

# tun mode
sudo ./mini-ss-macos-arm64 client -s 127.0.0.1:8388 -M :10088 -m aes-128-gcm -p password -CV --enable-tun --auto-detect-iface

# load from config file
./mini-ss-macos-arm64 client -c config.json
```

server

```shell
# help
./mini-ss-macos-arm64 server -h

# simple
./mini-ss-macos-arm64 server -s :8388 -m aes-128-cfb -p 123456 -CV
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

## Frontend

The frontend uses wails, see `frontend`

- https://github.com/wailsapp/wails

## References

- https://github.com/shadowsocks/go-shadowsocks2
- https://github.com/Dreamacro/clash
- https://github.com/ginuerzh/gost
- https://github.com/xjasonlyu/tun2socks