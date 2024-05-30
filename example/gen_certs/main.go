package main

import (
	"crypto/x509/pkix"
	"flag"
	"log"
	"net"
	"os"
	"path/filepath"

	"github.com/josexy/mini-ss/util/cert"
)

var (
	caCommonName string
	commonName   string
	dns          string
	ip           string
	mtls         bool
	outputDir    string
)

func main() {
	flag.StringVar(&caCommonName, "ca", "example.ca.com", "ca common name")
	flag.StringVar(&commonName, "cn", "www.helloworld.com", "common name")
	flag.StringVar(&dns, "dns", "www.helloworld.com", "dns name")
	flag.StringVar(&ip, "ip", "127.0.0.1", "ip address")
	flag.BoolVar(&mtls, "mtls", false, "enable mTLS")
	flag.StringVar(&outputDir, "o", "out-certs", "output directory")
	flag.Parse()

	caPrivateKey, err := cert.GeneratePrivateKey()
	if err != nil {
		log.Fatalln(err)
	}
	caTemplate, _, caCertPem, caKeyPem, err := cert.GenerateCACertificate(
		pkix.Name{CommonName: caCommonName}, caPrivateKey)
	if err != nil {
		log.Fatalln(err)
	}

	if _, err = os.Stat(outputDir); os.IsNotExist(err) {
		if err := os.Mkdir(outputDir, 0755); err != nil {
			log.Fatalln(err)
		}
	}

	caCertFile := filepath.Join(outputDir, "ca.crt")
	caKeyFile := filepath.Join(outputDir, "ca.key")

	if err = os.WriteFile(caCertFile, caCertPem, 0644); err != nil {
		log.Fatalln(err)
	}
	if err = os.WriteFile(caKeyFile, caKeyPem, 0644); err != nil {
		log.Fatalln(err)
	}
	log.Println("Generated ca.crt and ca.key succeed!")

	serverPrivateKey, err := cert.GeneratePrivateKey()
	if err != nil {
		log.Fatalln(err)
	}
	_, serverCertPem, serverKeyPem, err := cert.GenerateCertificateWithPEM(
		pkix.Name{CommonName: commonName},
		[]string{dns}, []net.IP{net.ParseIP(ip)}, caTemplate, caPrivateKey, serverPrivateKey)
	if err != nil {
		log.Fatalln(err)
	}

	serverCertFile := filepath.Join(outputDir, "server.crt")
	serverKeyFile := filepath.Join(outputDir, "server.key")

	if err = os.WriteFile(serverCertFile, serverCertPem, 0644); err != nil {
		log.Fatalln(err)
	}
	if err = os.WriteFile(serverKeyFile, serverKeyPem, 0644); err != nil {
		log.Fatalln(err)
	}
	log.Println("Generated server.crt and server.key succeed!")

	if !mtls {
		return
	}

	clientPrivateKey, err := cert.GeneratePrivateKey()
	if err != nil {
		log.Fatalln(err)
	}
	_, clientCertPem, clientKeyPem, err := cert.GenerateCertificateWithPEM(
		pkix.Name{CommonName: commonName},
		[]string{dns}, []net.IP{net.ParseIP(ip)}, caTemplate, caPrivateKey, clientPrivateKey)
	if err != nil {
		log.Fatalln(err)
	}

	clientCertFile := filepath.Join(outputDir, "client.crt")
	clientKeyFile := filepath.Join(outputDir, "client.key")

	if err = os.WriteFile(clientCertFile, clientCertPem, 0644); err != nil {
		log.Fatalln(err)
	}
	if err = os.WriteFile(clientKeyFile, clientKeyPem, 0644); err != nil {
		log.Fatalln(err)
	}
	log.Println("Generated client.crt and client.key succeed!")
	log.Println("Generated done!")
}
