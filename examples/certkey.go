package main

import (
	"context"
	"crypto"
	"crypto/ecdsa"
	"crypto/elliptic"
	cryptorand "crypto/rand"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"fmt"
	"io/ioutil"
	"math/big"
	"os"
	"time"

	"github.com/k3s-io/kine/pkg/client"
	"github.com/k3s-io/kine/pkg/endpoint"
)

var duration365d = time.Hour * 24 * 365

func genCert() {
	privateKey, err := ecdsa.GenerateKey(elliptic.P256(), cryptorand.Reader)
	if err != nil {
		panic("")
	}

	derBytes, err := x509.MarshalECPrivateKey(privateKey)
	if err != nil {
		panic("")
	}

	privateKeyPermBlock := &pem.Block{
		Type:  "EC PRIVATE KEY",
		Bytes: derBytes,
	}

	keyData := pem.EncodeToMemory(privateKeyPermBlock)
	ioutil.WriteFile("./test.key", keyData, os.FileMode(0600))

	// generate self sign cert
	var key interface{}
	var privateKeyPemBlock *pem.Block
	for {
		privateKeyPemBlock, keyData = pem.Decode(keyData)
		if privateKeyPemBlock == nil {
			break
		}

		switch privateKeyPemBlock.Type {
		case "EC PRIVATE KEY":
			if key, err = x509.ParseECPrivateKey(privateKeyPemBlock.Bytes); err != nil {
				panic("")
			}
		case "RSA PRIVATE KEY":
			if key, err = x509.ParsePKCS1PrivateKey(privateKeyPemBlock.Bytes); err != nil {
				panic("")
			}
		case "PRIVATE KEY":
			if key, err = x509.ParsePKCS8PrivateKey(privateKeyPemBlock.Bytes); err != nil {
				panic("")
			}
		}
	}

	now := time.Now()
	tmpl := x509.Certificate{
		SerialNumber: new(big.Int).SetInt64(0),
		Subject: pkix.Name{
			CommonName:   "common name",
			Organization: []string{"org"},
		},
		NotBefore:             now.UTC(),
		NotAfter:              now.Add(duration365d * 10).UTC(),
		KeyUsage:              x509.KeyUsageKeyEncipherment | x509.KeyUsageDigitalSignature | x509.KeyUsageCertSign,
		BasicConstraintsValid: true,
		IsCA:                  true,
	}

	certDERBytes, err := x509.CreateCertificate(cryptorand.Reader, &tmpl, &tmpl, key.(crypto.Signer).Public(), key.(crypto.Signer))
	if err != nil {
		panic("")
	}

	derCert, err := x509.ParseCertificate(certDERBytes)
	if err != nil {
		panic("")
	}

	block := pem.Block{
		Type:  "CERTIFICATE",
		Bytes: derCert.Raw,
	}
	cert := pem.EncodeToMemory(&block)

	ioutil.WriteFile("./test.pem", cert, os.FileMode(0644))
}

func main() {
	var config endpoint.Config
	etcdConfig, err := endpoint.Listen(context.Background(), config)
	if err != nil {
		return
	}
	fmt.Printf("### %+v ###\n", etcdConfig)

	c, err := client.New(etcdConfig)
	if err != nil {
		return
	}
	defer c.Close()

	genCert()

	select {}
}
