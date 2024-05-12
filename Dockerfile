FROM golang:alpine as builder

WORKDIR /work/src
COPY . /work/src

RUN apk add --update --no-cache make git && make build

FROM alpine:latest
WORKDIR /work

COPY --from=builder /work/src/example-configs /etc/configs 
COPY --from=builder /work/src/Country.mmdb . 
COPY --from=builder /work/src/bin/mini-ss .

RUN apk add --update --no-cache iproute2

ENTRYPOINT [ "./mini-ss" ]
