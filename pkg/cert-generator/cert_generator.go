package cert_generator

import (
	"bytes"
	cryptorand "crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"fmt"
	"github.com/sirupsen/logrus"
	"io/ioutil"
	"k8s-pod-mutator-webhook/internal/logger"
	"k8s-pod-mutator-webhook/pkg/webhook"
	"math/big"
	"os"
	"path/filepath"
	"time"
)

const organizationName = "k8s-pod-mutator.io"

const keyLength = 2048

type CertOutputFiles struct {
	CaCertOutputFile  string
	CaKeyOutputFile   string
	TlsCertOutputFile string
	TlsKeyOutputFile  string
}

type Certs struct {
	CaCert  []byte
	CaKey   []byte
	TlsCert []byte
	TlsKey  []byte
}

func Generate(serviceMetadata webhook.ServiceMetadata, outputFiles CertOutputFiles) (*Certs, error) {
	logger.Logger.WithFields(logrus.Fields{
		"serviceMetadata": fmt.Sprintf("%+v", serviceMetadata),
		"outputFiles":     fmt.Sprintf("%+v", outputFiles),
	}).Infoln("generating certs")

	logger.Logger.Debugln("generating ca key + cert...")
	caX509Cert := &x509.Certificate{
		SerialNumber: big.NewInt(2020),
		Subject: pkix.Name{
			Organization: []string{organizationName},
		},
		NotBefore:             time.Now(),
		NotAfter:              time.Now().AddDate(1, 0, 0),
		IsCA:                  true,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth, x509.ExtKeyUsageServerAuth},
		KeyUsage:              x509.KeyUsageDigitalSignature | x509.KeyUsageCertSign,
		BasicConstraintsValid: true,
	}

	caKey, err := rsa.GenerateKey(cryptorand.Reader, keyLength)
	if err != nil {
		return nil, err
	}
	caCert, err := createCert(caX509Cert, &caKey.PublicKey, caX509Cert, caKey)
	if err != nil {
		return nil, err
	}

	logger.Logger.Debugln("generating tls key + cert")
	tlsX509Cert := &x509.Certificate{
		DNSNames: []string{
			serviceMetadata.Name,
			fmt.Sprintf("%v.%v", serviceMetadata.Name, serviceMetadata.Namespace),
			fmt.Sprintf("%v.%v.svc", serviceMetadata.Name, serviceMetadata.Namespace),
		},
		SerialNumber: big.NewInt(1658),
		Subject: pkix.Name{
			CommonName:   fmt.Sprintf("%v.%v.svc", serviceMetadata.Name, serviceMetadata.Namespace),
			Organization: []string{organizationName},
		},
		NotBefore:    time.Now(),
		NotAfter:     time.Now().AddDate(1, 0, 0),
		SubjectKeyId: []byte{1, 2, 4, 8},
		ExtKeyUsage:  []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth, x509.ExtKeyUsageServerAuth},
		KeyUsage:     x509.KeyUsageDigitalSignature,
	}

	tlsKey, err := rsa.GenerateKey(cryptorand.Reader, keyLength)
	if err != nil {
		return nil, err
	}
	tlsCert, err := createCert(tlsX509Cert, &tlsKey.PublicKey, caX509Cert, caKey)
	if err != nil {
		return nil, err
	}

	encodedCaKey, err := encode(caKey)
	if err != nil {
		return nil, err
	}
	encodedTlsKey, err := encode(tlsKey)
	if err != nil {
		return nil, err
	}

	certs := &Certs{
		caCert.Bytes(),
		encodedCaKey.Bytes(),
		tlsCert.Bytes(),
		encodedTlsKey.Bytes(),
	}

	if err := write(certs, outputFiles); err != nil {
		return nil, err
	}

	logger.Logger.WithFields(logrus.Fields{
		"serviceMetadata": fmt.Sprintf("%+v", serviceMetadata),
		"outputFiles":     fmt.Sprintf("%+v", outputFiles),
	}).Infoln("successfully generated certs")

	return certs, nil
}

func createCert(x509Cert *x509.Certificate, publicKey *rsa.PublicKey, caX509Cert *x509.Certificate, caKey *rsa.PrivateKey) (*bytes.Buffer, error) {
	certBytes, err := x509.CreateCertificate(cryptorand.Reader, x509Cert, caX509Cert, publicKey, caKey)
	if err != nil {
		return nil, err
	}

	certPem := &bytes.Buffer{}
	err = pem.Encode(certPem, &pem.Block{Type: "CERTIFICATE", Bytes: certBytes})
	return certPem, err
}

func encode(key *rsa.PrivateKey) (*bytes.Buffer, error) {
	keyPem := &bytes.Buffer{}
	err := pem.Encode(keyPem, &pem.Block{
		Type:  "RSA PRIVATE KEY",
		Bytes: x509.MarshalPKCS1PrivateKey(key),
	})
	return keyPem, err
}

func write(certs *Certs, outputFiles CertOutputFiles) error {
	if err := writeFile(certs.CaCert, outputFiles.CaCertOutputFile); err != nil {
		return err
	}
	if err := writeFile(certs.CaKey, outputFiles.CaKeyOutputFile); err != nil {
		return err
	}
	if err := writeFile(certs.TlsCert, outputFiles.TlsCertOutputFile); err != nil {
		return err
	}
	if err := writeFile(certs.TlsKey, outputFiles.TlsKeyOutputFile); err != nil {
		return err
	}
	return nil
}

func writeFile(bytes []byte, path string) error {
	logger.Logger.WithFields(logrus.Fields{
		"path": path,
	}).Debugln("writing file...")
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0640); err != nil {
		return err
	}
	if err := ioutil.WriteFile(path, bytes, 0640); err != nil {
		return err
	}
	return nil
}
