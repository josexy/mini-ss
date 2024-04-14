package ss

import (
	"bufio"
	"crypto/tls"
	"crypto/x509"
	"errors"
	"net"
	"net/http"

	"github.com/josexy/logx"
	"github.com/josexy/mini-ss/transport"
	"github.com/josexy/mini-ss/util/cert"
	"github.com/josexy/mini-ss/util/logger"
)

func (r *httpReqHandler) handleMIMT(conn net.Conn) error {
	logger.Logger.Info("mitm proxy", logx.String("target", r.reqCtx.target), logx.Bool("https", r.reqCtx.isHttps))
	dstConn, err := transport.DialTCP(r.reqCtx.target)
	if err != nil {
		return err
	}
	defer dstConn.Close()
	if r.reqCtx.isHttps {
		r.relayHTTPSRequestAndResponse(conn, dstConn)
	} else {
		r.relayHTTPRequestAndResponse(conn, dstConn)
	}
	return nil
}

func (r *httpReqHandler) relayHTTPSRequestAndResponse(conn, dstConn net.Conn) (err error) {
	tlsConfig := &tls.Config{}
	host, _, _ := net.SplitHostPort(r.reqCtx.target)
	if net.ParseIP(host) == nil {
		tlsConfig.ServerName = host
	} else {
		tlsConfig.InsecureSkipVerify = true
	}
	tlsClientConn := tls.Client(dstConn, tlsConfig)
	if err = tlsClientConn.Handshake(); err != nil {
		return
	}

	cs := tlsClientConn.ConnectionState()
	var foundCert *x509.Certificate
	for _, cert := range cs.PeerCertificates {
		logger.Logger.Trace("cert info",
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
		return errors.New("cannot found a available server tls certificate")
	}
	// TODO: using cert cache pool
	privateKey, err := cert.GeneratePrivateKey()
	if err != nil {
		return err
	}

	caCert, caPriKey, err := cert.LoadCACertificate(r.owner.mitmOpt.caPath, r.owner.mitmOpt.keyPath)
	if err != nil {
		return err
	}

	serverCert, _, _, err := cert.GenerateCertificate(
		foundCert.Subject,
		foundCert.DNSNames, foundCert.IPAddresses,
		caCert, caPriKey, privateKey,
	)
	if err != nil {
		return err
	}
	tlsServerConn := tls.Server(conn, &tls.Config{
		Certificates: []tls.Certificate{serverCert},
	})
	if err = tlsServerConn.Handshake(); err != nil {
		return err
	}

	return r.relayHTTPRequestAndResponse(tlsServerConn, tlsClientConn)
}

func (r *httpReqHandler) relayHTTPRequestAndResponse(src, dst net.Conn) error {
	readRequest := func(src, dst net.Conn) (*http.Request, error) {
		req, err := http.ReadRequest(bufio.NewReader(src))
		if err != nil {
			return nil, err
		}
		logger.Logger.Trace("parse http request",
			logx.String("method", req.Method),
			logx.String("url", req.URL.String()),
			logx.String("host", req.Host),
			logx.Any("header", req.Header),
		)
		err = req.Write(dst)
		if err != nil {
			return nil, err
		}
		return req, nil
	}
	readResponse := func(req *http.Request, src, dst net.Conn) error {
		rsp, err := http.ReadResponse(bufio.NewReader(src), req)
		if err != nil {
			return err
		}
		logger.Logger.Trace("parse http response",
			logx.Int("code", rsp.StatusCode),
			logx.Any("header", rsp.Header),
		)
		rsp.Write(dst)
		return nil
	}
	var req *http.Request
	var err error
	if req, err = readRequest(src, dst); err != nil {
		return err
	}
	return readResponse(req, dst, src)
}
