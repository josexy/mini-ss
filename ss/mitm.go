package ss

import (
	"bufio"
	"context"
	"crypto/tls"
	"crypto/x509"
	"net"
	"net/http"
	"time"

	"github.com/josexy/logx"
	"github.com/josexy/mini-ss/transport"
	"github.com/josexy/mini-ss/util/cert"
	"github.com/josexy/mini-ss/util/logger"
)

func (r *httpReqHandler) initPrivateKeyAndCertPool() {
	r.priKeyPool = cert.NewPriKeyPool(10)
	r.certPool = cert.NewCertPool(r.owner.mitmOpt.fakeCertPool.capacity,
		time.Duration(r.owner.mitmOpt.fakeCertPool.interval)*time.Millisecond,
		time.Duration(r.owner.mitmOpt.fakeCertPool.expireSecond)*time.Millisecond,
	)
}

func (r *httpReqHandler) handleMIMT(ctx context.Context, conn net.Conn) error {
	reqCtx := ctx.Value(reqCtxKey).(reqContext)
	if reqCtx.isHttps {
		return r.handleHTTPSRequestAndResponse(ctx, conn)
	} else {
		dstConn, err := transport.DialTCP(reqCtx.Hostport())
		if err != nil {
			return err
		}
		defer dstConn.Close()
		return r.handleHTTPRequestAndResponse(ctx, conn, dstConn)
	}
}

func (r *httpReqHandler) getServerResponseCert(ctx context.Context, serverName string) (net.Conn, *tls.Config, error) {
	reqCtx := ctx.Value(reqCtxKey).(reqContext)
	dstConn, err := transport.DialTCP(reqCtx.Hostport())
	if err != nil {
		return nil, nil, err
	}
	tlsConfig := &tls.Config{}
	if serverName != "" {
		tlsConfig.ServerName = serverName
	} else {
		tlsConfig.InsecureSkipVerify = true
	}
	tlsClientConn := tls.Client(dstConn, tlsConfig)
	if err = tlsClientConn.Handshake(); err != nil {
		return nil, nil, err
	}
	// Get server certificate from local cache pool
	if serverCert, err := r.certPool.Get(reqCtx.host); err == nil {
		return tlsClientConn, &tls.Config{Certificates: []tls.Certificate{serverCert}}, nil
	}
	cs := tlsClientConn.ConnectionState()
	var foundCert *x509.Certificate
	for _, cert := range cs.PeerCertificates {
		logger.Logger.Trace("server cert info",
			logx.String("common_name", cert.Subject.CommonName),
			logx.Slice3("country", cert.Subject.Country),
			logx.Slice3("org", cert.Subject.Organization),
			logx.Slice3("org_unit", cert.Subject.OrganizationalUnit),
			logx.Slice3("locality", cert.Subject.Locality),
			logx.Slice3("province", cert.Subject.Province),
			logx.Slice3("dns", cert.DNSNames),
			logx.Slice3("ips", cert.IPAddresses),
			logx.Bool("isca", cert.IsCA),
		)
		if !cert.IsCA {
			foundCert = cert
		}
	}
	if foundCert == nil {
		return nil, nil, errServerCertUnavailable
	}
	// Get private key from local cache pool
	privateKey, err := r.priKeyPool.Get()
	if err != nil {
		return nil, nil, err
	}
	serverCert, err := cert.GenerateCertificate(
		foundCert.Subject, foundCert.DNSNames, foundCert.IPAddresses,
		r.owner.mitmOpt.caCert, r.owner.mitmOpt.caKey, privateKey,
	)
	if err != nil {
		return nil, nil, err
	}
	r.certPool.Add(reqCtx.host, serverCert)
	return tlsClientConn, &tls.Config{Certificates: []tls.Certificate{serverCert}}, nil
}

func (r *httpReqHandler) handleHTTPSRequestAndResponse(ctx context.Context, conn net.Conn) (err error) {
	// Load ca certificate and key failed
	if r.owner.mitmOpt.caErr != nil {
		return r.owner.mitmOpt.caErr
	}
	var tlsClientConn net.Conn
	tlsServerConn := tls.Server(conn, &tls.Config{
		GetConfigForClient: func(chi *tls.ClientHelloInfo) (tlsConfig *tls.Config, err error) {
			tlsClientConn, tlsConfig, err = r.getServerResponseCert(ctx, chi.ServerName)
			return
		},
	})
	if err = tlsServerConn.Handshake(); err != nil {
		logger.Logger.ErrorBy(err)
		return
	}

	defer tlsClientConn.Close()
	return r.handleHTTPRequestAndResponse(ctx, tlsServerConn, tlsClientConn)
}

func (r *httpReqHandler) handleHTTPRequestAndResponse(ctx context.Context, srcConn, dstConn net.Conn) (err error) {
	reqCtx := ctx.Value(reqCtxKey).(reqContext)
	request := reqCtx.request
	// Read the http request for https via tls tunnel
	if reqCtx.isHttps && request == nil {
		request, err = http.ReadRequest(bufio.NewReader(srcConn))
		if err != nil {
			return err
		}
		request.URL.Scheme = "https"
		request.URL.Host = request.Host
	}
	transport := &http.Transport{
		DialContext:    func(context.Context, string, string) (net.Conn, error) { return dstConn, nil },
		DialTLSContext: func(context.Context, string, string) (net.Conn, error) { return dstConn, nil },
	}
	response, err := transport.RoundTrip(request)
	if err != nil {
		return err
	}
	defer response.Body.Close()
	response.Write(srcConn)
	return nil
}
