package cert

import (
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"io"
	"net"
	"net/http"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestGenerateCertificate(t *testing.T) {
	caPrivateKey, err := GeneratePrivateKey()
	assert.NoError(t, err)
	caTemplate, cert, certPem, keyPem, err := GenerateCACertificate(
		pkix.Name{CommonName: "example.ca.com"}, caPrivateKey)
	assert.NoError(t, err)
	_ = cert

	defer os.RemoveAll("/tmp/cert")
	os.Mkdir("/tmp/cert", 0755)
	os.WriteFile("/tmp/cert/ca.crt", certPem, 0644)
	os.WriteFile("/tmp/cert/ca.key", keyPem, 0644)

	serverPrivateKey, err := GeneratePrivateKey()
	assert.NoError(t, err)
	serverCert, serverCertPem, serverKeyPem, err := GenerateCertificateWithPEM(
		pkix.Name{CommonName: "www.helloworld.com"},
		[]string{"www.helloworld.com"},
		[]net.IP{net.ParseIP("127.0.0.1")}, caTemplate, caPrivateKey, serverPrivateKey)
	assert.NoError(t, err)

	os.WriteFile("/tmp/cert/server.crt", serverCertPem, 0644)
	os.WriteFile("/tmp/cert/server.key", serverKeyPem, 0644)

	go func() {
		time.Sleep(time.Second * 1)
		certPool := x509.NewCertPool()
		certPool.AppendCertsFromPEM(certPem)
		client := &http.Client{
			Transport: &http.Transport{
				TLSClientConfig: &tls.Config{
					RootCAs:    certPool,
					ServerName: "www.helloworld.com",
				},
			},
		}
		rsp, err := client.Get("https://127.0.0.1:10086")
		if err != nil {
			t.Log(err)
			return
		}
		defer rsp.Body.Close()
		data, _ := io.ReadAll(rsp.Body)
		t.Log(rsp.StatusCode, string(data))
	}()
	server := &http.Server{
		Addr:      ":10086",
		TLSConfig: &tls.Config{Certificates: []tls.Certificate{serverCert}},
		Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Write([]byte("hello world"))
		}),
	}
	go func() {
		if err := server.ListenAndServeTLS("", ""); err != nil && err != http.ErrServerClosed {
			t.Log(err)
		}
	}()
	time.Sleep(time.Second * 2)
	server.Close()
}
