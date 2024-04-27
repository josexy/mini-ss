package cert

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"errors"
	"math/big"
	"net"
	"os"
	"time"
)

const defaultKeySize = 2048

func GeneratePrivateKey() (*rsa.PrivateKey, error) {
	privateKey, err := rsa.GenerateKey(rand.Reader, defaultKeySize)
	if err != nil {
		return nil, err
	}
	err = privateKey.Validate()
	if err != nil {
		return nil, err
	}
	return privateKey, nil
}

func generateCertificateTemplate(subject pkix.Name, dnsNames []string, ipAddresses []net.IP, notBefore, notAfter time.Time, isCA bool) (*x509.Certificate, error) {
	serialNumberLimit := new(big.Int).Lsh(big.NewInt(1), 128)
	serialNumber, err := rand.Int(rand.Reader, serialNumberLimit)
	if err != nil {
		return nil, err
	}
	template := x509.Certificate{
		SerialNumber:          serialNumber,
		Subject:               subject,
		NotBefore:             notBefore,
		NotAfter:              notAfter,
		KeyUsage:              x509.KeyUsageKeyEncipherment | x509.KeyUsageDigitalSignature,
		ExtKeyUsage:           []x509.ExtKeyUsage{},
		BasicConstraintsValid: true,
		IsCA:                  isCA,
	}
	if isCA {
		template.KeyUsage |= x509.KeyUsageCertSign | x509.KeyUsageCRLSign
	} else {
		template.ExtKeyUsage = append(template.ExtKeyUsage, x509.ExtKeyUsageServerAuth)
		template.DNSNames = dnsNames
		template.IPAddresses = ipAddresses
	}
	return &template, nil
}

func GenerateCACertificate(subject pkix.Name, privateKey *rsa.PrivateKey) (template *x509.Certificate, cert tls.Certificate, certPem []byte, keyPem []byte, err error) {
	template, err = generateCertificateTemplate(
		subject,
		nil, nil,
		time.Now(),
		time.Now().AddDate(2, 0, 0),
		true,
	)
	if err != nil {
		return
	}
	certBytes, err := x509.CreateCertificate(rand.Reader, template, template, &privateKey.PublicKey, privateKey)
	if err != nil {
		return
	}
	keyPem = pem.EncodeToMemory(&pem.Block{Type: "RSA PRIVATE KEY", Bytes: x509.MarshalPKCS1PrivateKey(privateKey)})
	certPem = pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: certBytes})
	cert, err = tls.X509KeyPair(certPem, keyPem)
	return
}

func GenerateCertificateWithPEM(subject pkix.Name, dnsNames []string, ipAddresses []net.IP, caCertTemplate *x509.Certificate,
	caPrivateKey, privateKey *rsa.PrivateKey) (cert tls.Certificate, certPem []byte, keyPem []byte, err error) {
	template, err := generateCertificateTemplate(
		subject, dnsNames, ipAddresses,
		time.Now(),
		time.Now().AddDate(1, 0, 0),
		false,
	)
	if err != nil {
		return
	}
	certBytes, err := x509.CreateCertificate(rand.Reader, template, caCertTemplate, &privateKey.PublicKey, caPrivateKey)
	if err != nil {
		return
	}
	keyPem = pem.EncodeToMemory(&pem.Block{Type: "RSA PRIVATE KEY", Bytes: x509.MarshalPKCS1PrivateKey(privateKey)})
	certPem = pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: certBytes})
	cert, err = tls.X509KeyPair(certPem, keyPem)
	return
}

func GenerateCertificate(subject pkix.Name, dnsNames []string, ipAddresses []net.IP, caCertTemplate *x509.Certificate,
	caPrivateKey, privateKey *rsa.PrivateKey) (cert tls.Certificate, err error) {
	template, err := generateCertificateTemplate(
		subject, dnsNames, ipAddresses,
		time.Now(),
		time.Now().AddDate(1, 0, 0),
		false,
	)
	if err != nil {
		return
	}
	certBytes, err := x509.CreateCertificate(rand.Reader, template, caCertTemplate, &privateKey.PublicKey, caPrivateKey)
	if err != nil {
		return
	}
	cert = tls.Certificate{
		Certificate: [][]byte{certBytes},
		PrivateKey:  privateKey,
	}
	return
}

func LoadCACertificate(certPath, keyPath string) (caCert *x509.Certificate, caPriKey *rsa.PrivateKey, err error) {
	certPem, err := os.ReadFile(certPath)
	if err != nil {
		return
	}
	keyPem, err := os.ReadFile(keyPath)
	if err != nil {
		return
	}
	certBlock, _ := pem.Decode(certPem)
	keyBlock, _ := pem.Decode(keyPem)
	caCert, err = x509.ParseCertificate(certBlock.Bytes)
	if err != nil {
		return
	}
	caPriKey, err = x509.ParsePKCS1PrivateKey(keyBlock.Bytes)
	if err == nil {
		return
	}
	priKey, err := x509.ParsePKCS8PrivateKey(keyBlock.Bytes)
	if err == nil {
		var ok bool
		if caPriKey, ok = priKey.(*rsa.PrivateKey); !ok {
			err = errors.New("private key is not of RSA type")
		}
	}
	return
}
