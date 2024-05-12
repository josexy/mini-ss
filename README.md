
# mini-ss

Mini shadowsocks server and client in Golang

[![Go Report Card](https://goreportcard.com/badge/github.com/josexy/mini-ss)](https://goreportcard.com/report/github.com/josexy/mini-ss)
[![License](https://img.shields.io/github/license/josexy/mini-ss)](https://github.com/josexy/mini-ss/blob/main/LICENSE)
[![Publish Go Releases](https://github.com/josexy/mini-ss/actions/workflows/go-release.yml/badge.svg)](https://github.com/josexy/mini-ss/actions/workflows/go-release.yml)

## Usage

### Clone

```shell
git clone https://github.com/josexy/mini-ss
# build
cd mini-ss && make build
# copy Country.mmdb
cp Country.mmdb bin/ && cd bin
# help
./mini-ss -h
./mini-ss client -h
./mini-ss server -h
```

### Client

```shell
# help
./mini-ss client -h

# simple
./mini-ss client -s 127.0.0.1:8388 -l :10086 -x :10087 -m aes-128-cfb -p 123456 -CV3

# udp relay if support
./mini-ss client -s 127.0.0.1:8388 -M 127.0.0.1:10088 -m aes-128-cfb -p 123456 -CV3 --udp-relay 

# enable tun mode
sudo ./mini-ss client -s 127.0.0.1:8388 -M :10088 -m aes-128-cfb -p 123456 -CV3 --tun-enable --auto-detect-iface

# ssr client
./mini-ss client -s server:port -M 127.0.0.1:10088 -m aes-256-cfb -p 123456 -t default -o tls1.2_ticket_auth -O auth_chain_a -T ssr -CV3 --system-proxy

# load from config file
./mini-ss client -c ../example-configs/simple-client-config.yaml
```

### Server

```shell
# help
./mini-ss server -h

# simple
./mini-ss server -s :8388 -m aes-128-cfb -p 123456 -CV3 --udp-relay --auto-detect-iface

# load from config file
./mini-ss server -c ../example-configs/simple-server-config.yaml
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

## Docker usage

Here is an example of how to use the Dockerfile to build the image and run the container for server and client.

```shell
docker build -t josexy/mini-ss:v1 -f docker/Dockerfile .

docker stop mini-ss-server ; docker rm mini-ss-server
docker stop mini-ss-client ; docker rm mini-ss-client

docker network create mini-ss-network
# run as server
docker run -itd --name mini-ss-server --network mini-ss-network -p 8388:8388 -v ./example-configs:/etc/configs josexy/mini-ss:v1 server -c /etc/configs/simple-server-config.yaml
# run as client, need to modify server address of config file to link to server
docker run -itd --name mini-ss-client --link mini-ss-server:mini-ss-server --network mini-ss-network -p 10088:10088 -v ./example-configs:/etc/configs josexy/mini-ss:v1 client -c /etc/configs/simple-client-config.yaml

# see the logs
docker logs mini-ss-client -f
docker logs mini-ss-server -f
```

## References

- https://github.com/shadowsocks/go-shadowsocks2
- https://github.com/Dreamacro/clash
- https://github.com/ginuerzh/gost
- https://github.com/xjasonlyu/tun2socks
- https://github.com/josexy/netstackgo
