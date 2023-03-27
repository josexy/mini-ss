package util

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"math/big"
	"time"
)

func GenCertificate() (tls.Certificate, error) {
	priKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return tls.Certificate{}, err
	}
	template := x509.Certificate{
		SerialNumber:          big.NewInt(int64(time.Now().UnixNano())),
		Subject:               pkix.Name{Organization: []string{"mini-ss"}},
		NotBefore:             time.Now(),
		NotAfter:              time.Now().Add(time.Hour * 24 * 365),
		KeyUsage:              x509.KeyUsageDigitalSignature | x509.KeyUsageKeyEncipherment,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		SignatureAlgorithm:    x509.SHA256WithRSA,
		BasicConstraintsValid: true,
	}
	certBytes, err := x509.CreateCertificate(rand.Reader, &template, &template, &priKey.PublicKey, priKey)
	if err != nil {
		return tls.Certificate{}, err
	}
	rawCert := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: certBytes})
	rawKey := pem.EncodeToMemory(&pem.Block{Type: "RSA PRIVATE KEY", Bytes: x509.MarshalPKCS1PrivateKey(priKey)})
	return tls.X509KeyPair(rawCert, rawKey)
}
