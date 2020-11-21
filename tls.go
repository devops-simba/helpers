package helpers

import (
	"crypto"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"errors"
	"math/big"
	"time"
)

var (
	UnsupportedEncryptionType = errors.New("Unsupported encryption type")
	NoIssuerCertMustBeCA      = errors.New("When there is no issuer for the certificate it must be a CA")
)

type CryptoAlgorithm string

const (
	RSA2048  CryptoAlgorithm = "RSA2048"
	RSA4096  CryptoAlgorithm = "RSA4096"
	RSA8192  CryptoAlgorithm = "RSA8192"
	ECDSA224 CryptoAlgorithm = "ECDSA224"
	ECDSA256 CryptoAlgorithm = "ECDSA256"
	ECDSA384 CryptoAlgorithm = "ECDSA384"
	ECDSA521 CryptoAlgorithm = "ECDSAP521"
)

func CreatePrivateKey(algo CryptoAlgorithm) (crypto.PrivateKey, error) {
	switch algo {
	case RSA2048:
		return rsa.GenerateKey(rand.Reader, 2048)
	case RSA4096:
		return rsa.GenerateKey(rand.Reader, 4096)
	case RSA8192:
		return rsa.GenerateKey(rand.Reader, 8192)
	case ECDSA224:
		return ecdsa.GenerateKey(elliptic.P224(), rand.Reader)
	case ECDSA256:
		return ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	case ECDSA384:
		return ecdsa.GenerateKey(elliptic.P384(), rand.Reader)
	case ECDSA521:
		return ecdsa.GenerateKey(elliptic.P521(), rand.Reader)
	default:
		return nil, UnsupportedEncryptionType
	}
}
func GetPublicKey(privKey crypto.PrivateKey) (crypto.PublicKey, error) {
	switch k := privKey.(type) {
	case *rsa.PrivateKey:
		return k.Public(), nil

	case *ecdsa.PrivateKey:
		return k.Public(), nil

	default:
		return nil, UnsupportedEncryptionType
	}
}

func CreateX509Certificate(commonName string, isCA bool, expiryTime time.Time) (*x509.Certificate, error) {
	serialNumber, err := rand.Int(rand.Reader, big.NewInt(MaxInt64))
	if err != nil {
		return nil, err
	}

	result := &x509.Certificate{
		IsCA: isCA,
		Subject: pkix.Name{
			CommonName: commonName,
		},
		SerialNumber: serialNumber,
		NotBefore:    time.Now().Add(-5 * time.Minute),
		NotAfter:     expiryTime,
		KeyUsage:     x509.KeyUsageDigitalSignature,
		ExtKeyUsage:  []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth, x509.ExtKeyUsageServerAuth},
	}
	if isCA {
		result.BasicConstraintsValid = true
		result.KeyUsage |= x509.KeyUsageCertSign | x509.KeyUsageKeyEncipherment
	}

	return result, nil
}

type CertAndKey struct {
	Certificate  *x509.Certificate
	PrivateKey   crypto.PrivateKey
	certDERBytes []byte
}

func CreateCertificate(cert *x509.Certificate, privateKey crypto.PrivateKey, issuer *CertAndKey) (*CertAndKey, error) {
	var err error
	if issuer == nil && !cert.IsCA {
		return nil, NoIssuerCertMustBeCA
	}

	if privateKey == nil {
		privateKey, err = CreatePrivateKey(RSA4096)
		if err != nil {
			return nil, err
		}
	}

	publicKey, err := GetPublicKey(privateKey)
	if err != nil {
		return nil, err
	}

	parent := cert
	signKey := privateKey
	if issuer != nil {
		parent = issuer.Certificate
		signKey = issuer.PrivateKey
	}
	der, err := x509.CreateCertificate(rand.Reader, cert, parent, publicKey, signKey)
	if err != nil {
		return nil, err
	}

	return &CertAndKey{
		Certificate:  cert,
		PrivateKey:   privateKey,
		certDERBytes: der,
	}, nil
}
func (this *CertAndKey) CertificatePEMBlock() *pem.Block {
	return &pem.Block{Type: "CERTIFICATE", Bytes: this.certDERBytes}
}
func (this *CertAndKey) PrivateKeyPEMBlock() (*pem.Block, error) {
	switch k := this.PrivateKey.(type) {
	case *rsa.PrivateKey:
		return &pem.Block{Type: "RSA PRIVATE KEY", Bytes: x509.MarshalPKCS1PrivateKey(k)}, nil

	case *ecdsa.PrivateKey:
		b, err := x509.MarshalECPrivateKey(k)
		if err != nil {
			return nil, err
		}
		return &pem.Block{Type: "EC PRIVATE KEY", Bytes: b}, nil

	default:
		return nil, UnsupportedEncryptionType
	}
}
func (this *CertAndKey) CreateCertificate(cert *x509.Certificate, privateKey crypto.PrivateKey) (*CertAndKey, error) {
	return CreateCertificate(cert, privateKey, this)
}
