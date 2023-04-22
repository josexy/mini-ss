#!/bin/bash

CA_DOMAIN=example.ca.com
DOMAIN=*.helloworld.com
OUTDIR=certs

# 制作CA证书
# 生成.key私钥文件
openssl genrsa -out ${OUTDIR}/ca.key 2048
# 生成.csr证书签名请求文件
openssl req -x509 -new -nodes -key ${OUTDIR}/ca.key -subj "/C=GB/L=China/CN=${CA_DOMAIN}" -days 5000 -out ${OUTDIR}/ca.csr
# 自签名生成.crt证书文件
openssl req -x509 -new -days 3650 -key ${OUTDIR}/ca.key -out ${OUTDIR}/ca.crt -subj "/C=GB/L=China/CN=${CA_DOMAIN}"

# 制作服务端证书
# 生成.key私钥文件
openssl genrsa -out ${OUTDIR}/server.key 2048
# 生成.csr证书签名请求文件
openssl req -new -key ${OUTDIR}/server.key \
    -subj "/C=GB/L=China/CN=${DOMAIN}" \
    -reqexts SAN \
    -config <(cat /System/Library/OpenSSL/openssl.cnf <(printf "[SAN]\nsubjectAltName=DNS:${DOMAIN}")) \
    -out ${OUTDIR}/server.csr
# 签名生成.crt 证书文件
openssl x509 -req -days 3650 -in ${OUTDIR}/server.csr \
    -CA ${OUTDIR}/ca.crt -CAkey ${OUTDIR}/ca.key -CAcreateserial \
    -extensions SAN \
    -extfile <(printf "subjectAltName=DNS:${DOMAIN}") \
    -extfile <(cat /System/Library/OpenSSL/openssl.cnf <(printf "[SAN]\nsubjectAltName=DNS:${DOMAIN}")) \
    -out ${OUTDIR}/server.crt

# 如果开启mTLS，需要制作客户端证书文件
# 生成.key  私钥文件
openssl genrsa -out ${OUTDIR}/client.key 2048
# 生成.csr证书签名请求文件
openssl req -new -key ${OUTDIR}/client.key \
    -subj "/C=GB/L=China/CN=${DOMAIN}" \
    -reqexts SAN \
    -config <(cat /System/Library/OpenSSL/openssl.cnf <(printf "[SAN]\nsubjectAltName=DNS:${DOMAIN}")) \
    -out ${OUTDIR}/client.csr
# 签名生成.crt 证书文件
openssl x509 -req -days 3650 -in ${OUTDIR}/client.csr \
    -CA ${OUTDIR}/ca.crt -CAkey ${OUTDIR}/ca.key -CAcreateserial \
    -extensions SAN \
    -extfile <(printf "subjectAltName=DNS:${DOMAIN}") \
    -extfile <(cat /System/Library/OpenSSL/openssl.cnf <(printf "[SAN]\nsubjectAltName=DNS:${DOMAIN}")) \
    -out ${OUTDIR}/client.crt
