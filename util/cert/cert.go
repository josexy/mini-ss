package cert

import (
	"crypto/tls"
	"crypto/x509"
	"errors"
	"os"
)

func GetClientMTlsConfig(certPath, keyPath, caPath, hostname string) (*tls.Config, error) {
	var rootCAs *x509.CertPool
	if caPath != "" {
		ca, err := os.ReadFile(caPath)
		if err != nil {
			return nil, err
		}
		// client ca file
		rootCAs = x509.NewCertPool()
		if ok := rootCAs.AppendCertsFromPEM(ca); !ok {
			return nil, errors.New("failed to append certs")
		}
	}
	// client mtls cert and key file
	cert, err := tls.LoadX509KeyPair(certPath, keyPath)
	if err != nil {
		return nil, err
	}
	return &tls.Config{
		Certificates: []tls.Certificate{cert}, // used for client tls
		ServerName:   hostname,                // ussed for client common tls
		RootCAs:      rootCAs,                 // used for client common tls
	}, nil
}

func GetServerMTlsConfig(certPath, keyPath, caPath string) (*tls.Config, error) {
	var clientCAs *x509.CertPool
	if caPath != "" {
		// server ca file
		ca, err := os.ReadFile(caPath)
		if err != nil {
			return nil, err
		}
		clientCAs = x509.NewCertPool()
		if ok := clientCAs.AppendCertsFromPEM(ca); !ok {
			return nil, errors.New("failed to append ca file to cert pool")
		}
	}
	// server tls cert and key file
	cert, err := tls.LoadX509KeyPair(certPath, keyPath)
	if err != nil {
		return nil, err
	}
	return &tls.Config{
		Certificates: []tls.Certificate{cert},        // used for server tls
		ClientAuth:   tls.RequireAndVerifyClientCert, // used for server mtls
		ClientCAs:    clientCAs,                      // used for server mtls
	}, nil
}

func GetInsecureTlsConfig() *tls.Config {
	return &tls.Config{
		InsecureSkipVerify: true,
	}
}

// client tls need ca file and Common name
func GetClientTlsConfig(caPath, hostname string) (*tls.Config, error) {
	var rootCAs *x509.CertPool
	if caPath != "" {
		ca, err := os.ReadFile(caPath)
		if err != nil {
			return nil, err
		}
		rootCAs = x509.NewCertPool()
		if ok := rootCAs.AppendCertsFromPEM(ca); !ok {
			return nil, errors.New("failed to append ca file to cert pool")
		}
	}
	return &tls.Config{
		ServerName: hostname,
		RootCAs:    rootCAs,
	}, nil
}

// server tls need cert file and key file
func GetServerTlsConfig(certPath, keyPath string) (*tls.Config, error) {
	cert, err := tls.LoadX509KeyPair(certPath, keyPath)
	if err != nil {
		return nil, err
	}
	return &tls.Config{
		Certificates: []tls.Certificate{cert},
	}, nil
}
