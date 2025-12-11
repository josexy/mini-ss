package interceptor

import (
	"context"
	"crypto/tls"
	"encoding/hex"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/josexy/logx"
	"github.com/josexy/mini-ss/util/logger"
	"github.com/josexy/mitmpgo"
	"github.com/josexy/mitmpgo/buf"
	"github.com/josexy/mitmpgo/metadata"
)

type chunkBodyReader struct {
	io.ReadCloser
	N int64
}

func newChunkBodyReader(r io.ReadCloser, chunkBodySize int64) *chunkBodyReader {
	return &chunkBodyReader{
		N:          chunkBodySize,
		ReadCloser: r,
	}
}

func (r *chunkBodyReader) Read(p []byte) (n int, err error) {
	if r.N <= 0 {
		return 0, io.EOF
	}
	if int64(len(p)) > r.N {
		p = p[0:r.N]
	}
	n, err = r.ReadCloser.Read(p)
	if n > 0 {
		fmt.Printf("--> hex dump(data size/chunk size: %d/%d):\n%s\n", n, r.N, hex.Dump(p[:n]))
	}
	return
}

func ErrHandler(ec mitmpgo.ErrorContext) {
	logger.Logger.Error("mitm proxy error",
		logx.String("remote_addr", ec.RemoteAddr),
		logx.String("hostport", ec.Hostport),
		logx.Error("error", ec.Error),
	)
}

func HttpInterceptor(ctx context.Context, req *http.Request, invoker mitmpgo.HTTPDelegatedInvoker) (*http.Response, error) {
	_md, _ := metadata.FromContext(ctx)
	md := _md.MD()

	req.Body = newChunkBodyReader(req.Body, 512)
	// if md.StreamBody {
	// 	req.Body = newChunkBodyReader(req.Body, 512)
	// } else {
	// 	data, _ := httputil.DumpRequest(req, true)
	// 	fmt.Println("request:", string(data))
	// }

	rsp, err := invoker.Invoke(req)
	if err != nil {
		return rsp, err
	}

	logsFields := []logx.Field{
		logx.Object("request",
			logx.Bool("stream_body", md.StreamBody),
			logx.String("source", md.SourceAddr.String()),
			logx.String("destination", md.DestinationAddr.String()),
			logx.String("hostport", md.RequestHostport),
			logx.String("host", req.Host),
			logx.String("proto", req.Proto),
			logx.String("method", req.Method),
			logx.Bool("tls", req.TLS != nil),
			logx.String("url", req.URL.String()),
			logx.Any("headers", map[string][]string(req.Header)),
		),
		logx.Object("response",
			logx.Duration("connection_establishment", time.Since(md.ConnectionEstablishedTs)),
			logx.Duration("ssl_handshake_latency", md.SSLHandshakeCompletedTs.Sub(md.ConnectionEstablishedTs)),
			logx.Duration("request_latency", time.Since(md.RequestProcessedTs)),
			logx.String("status", rsp.Status),
			logx.String("protocol", rsp.Proto),
		),
	}
	if md.TLSState != nil && md.ServerCertificate != nil {
		logsFields = append(logsFields,
			logx.Object("tls state",
				logx.String("server_name", md.TLSState.ServerName),
				logx.String("alpn", strings.Join(md.TLSState.ALPN, ",")),
				logx.String("selected_ciphersuite", tls.CipherSuiteName(md.TLSState.SelectedCipherSuite)),
				logx.String("selected_version", tls.VersionName(md.TLSState.SelectedTLSVersion)),
				logx.String("selected_alpn", md.TLSState.SelectedALPN),
			),
			logx.Object("server certificate",
				logx.Int("version", md.ServerCertificate.Version),
				logx.String("not_after", md.ServerCertificate.NotAfter.String()),
				logx.String("not_before", md.ServerCertificate.NotBefore.String()),
				logx.String("subject", md.ServerCertificate.Subject.String()),
				logx.String("issuer", md.ServerCertificate.Issuer.String()),
				logx.String("serial_number", md.ServerCertificate.SerialNumberHex()),
				logx.String("signature_algorithm", md.ServerCertificate.SignatureAlgorithm.String()),
				logx.String("sha1_fingerprint", md.ServerCertificate.Sha1FingerprintHex()),
				logx.String("sha256_fingerprint", md.ServerCertificate.Sha256FingerprintHex()),
				logx.String("dns", strings.Join(md.ServerCertificate.DNSNames, ",")),
				logx.Any("ip", md.ServerCertificate.IPAddresses),
			),
		)
	}

	logger.Logger.Debug("http", logsFields...)

	rsp.Body = newChunkBodyReader(rsp.Body, 512)
	// if md.StreamBody {
	// 	rsp.Body = newChunkBodyReader(rsp.Body, 512)
	// } else {
	// 	data, _ := httputil.DumpResponse(rsp, true)
	// 	fmt.Println("response:", string(data))
	// }

	return rsp, err
}

func WebsocketInterceptor(ctx context.Context, dir mitmpgo.WSDirection, msgType int, b *buf.Buffer, req *http.Request, wdi mitmpgo.WebsocketDelegatedInvoker) error {
	_md, _ := metadata.FromContext(ctx)
	md := _md.MD()
	logsFields := []logx.Field{
		logx.Object("request",
			logx.String("source", md.SourceAddr.String()),
			logx.String("destination", md.DestinationAddr.String()),
			logx.String("hostport", md.RequestHostport),
			logx.String("host", req.Host),
			logx.String("proto", req.Proto),
			logx.String("method", req.Method),
			logx.Bool("tls", req.TLS != nil),
			logx.String("url", req.URL.String()),
			logx.Any("headers", map[string][]string(req.Header)),
		),
		logx.Object("ws",
			logx.Duration("connection_establishment", time.Since(md.ConnectionEstablishedTs)),
			logx.Duration("ssl_handshake_latency", md.SSLHandshakeCompletedTs.Sub(md.ConnectionEstablishedTs)),
			logx.Duration("request_latency", time.Since(md.RequestProcessedTs)),
			logx.Int("msg_type", msgType),
			logx.String("direction", dir.String()),
			logx.Int("data_len", b.Len()),
		),
	}
	if md.TLSState != nil && md.ServerCertificate != nil {
		logsFields = append(logsFields,
			logx.Object("tls state",
				logx.String("server_name", md.TLSState.ServerName),
				logx.String("alpn", strings.Join(md.TLSState.ALPN, ",")),
				logx.String("selected_ciphersuite", tls.CipherSuiteName(md.TLSState.SelectedCipherSuite)),
				logx.String("selected_version", tls.VersionName(md.TLSState.SelectedTLSVersion)),
				logx.String("selected_alpn", md.TLSState.SelectedALPN),
			),
			logx.Object("server certificate",
				logx.Int("version", md.ServerCertificate.Version),
				logx.String("not_after", md.ServerCertificate.NotAfter.String()),
				logx.String("not_before", md.ServerCertificate.NotBefore.String()),
				logx.String("subject", md.ServerCertificate.Subject.String()),
				logx.String("issuer", md.ServerCertificate.Issuer.String()),
				logx.String("serial_number", md.ServerCertificate.SerialNumberHex()),
				logx.String("signature_algorithm", md.ServerCertificate.SignatureAlgorithm.String()),
				logx.String("sha1_fingerprint", md.ServerCertificate.Sha1FingerprintHex()),
				logx.String("sha256_fingerprint", md.ServerCertificate.Sha256FingerprintHex()),
				logx.String("dns", strings.Join(md.ServerCertificate.DNSNames, ",")),
				logx.Any("ip", md.ServerCertificate.IPAddresses),
			),
		)
	}

	logger.Logger.Debug("ws", logsFields...)
	return wdi.Invoke(msgType, b)
}
