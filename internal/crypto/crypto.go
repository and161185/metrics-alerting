package crypto

import (
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"errors"
	"fmt"
	"os"
)

var (
	ErrNoPEMBlocks    = errors.New("no PEM blocks found")
	ErrNotRSAPublic   = errors.New("PEM is not an RSA public key")
	ErrNotRSAPrivate  = errors.New("PEM is not an RSA private key")
	ErrUnsupportedPEM = errors.New("unsupported PEM block type")
)

// LoadPublicKey read RSA public key from PEM-file.
func LoadPublicKey(path string) (*rsa.PublicKey, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read public key: %w", err)
	}
	return LoadPublicKeyFromBytes(b)
}

// LoadPrivateKey read RSA private key from PEM-file.
func LoadPrivateKey(path string) (*rsa.PrivateKey, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read private key: %w", err)
	}
	return LoadPrivateKeyFromBytes(b)
}

// LoadPublicKeyFromBytes parse RSA public key from PEM-bytes.
// Support: "PUBLIC KEY" (PKIX, SubjectPublicKeyInfo) and "RSA PUBLIC KEY" (PKCS#1).
func LoadPublicKeyFromBytes(pemBytes []byte) (*rsa.PublicKey, error) {
	var found bool
	for {
		var block *pem.Block
		block, pemBytes = pem.Decode(pemBytes)
		if block == nil {
			break
		}
		found = true

		switch block.Type {
		case "PUBLIC KEY":
			pub, err := x509.ParsePKIXPublicKey(block.Bytes)
			if err != nil {
				return nil, fmt.Errorf("parse PKIX public key: %w", err)
			}
			rsaPub, ok := pub.(*rsa.PublicKey)
			if !ok {
				return nil, ErrNotRSAPublic
			}
			return rsaPub, nil

		case "RSA PUBLIC KEY":
			rsaPub, err := x509.ParsePKCS1PublicKey(block.Bytes)
			if err != nil {
				return nil, fmt.Errorf("parse PKCS1 public key: %w", err)
			}
			return rsaPub, nil

		default:

			continue
		}
	}
	if !found {
		return nil, ErrNoPEMBlocks
	}
	return nil, ErrUnsupportedPEM
}

// LoadPrivateKeyFromBytes parse RSA private key from PEM-bytes.
// Support: "RSA PRIVATE KEY" (PKCS#1) and "PRIVATE KEY" (PKCS#8).
func LoadPrivateKeyFromBytes(pemBytes []byte) (*rsa.PrivateKey, error) {
	var found bool
	for {
		var block *pem.Block
		block, pemBytes = pem.Decode(pemBytes)
		if block == nil {
			break
		}
		found = true

		switch block.Type {
		case "RSA PRIVATE KEY":
			priv, err := x509.ParsePKCS1PrivateKey(block.Bytes)
			if err != nil {
				return nil, fmt.Errorf("parse PKCS1 private key: %w", err)
			}
			return priv, nil

		case "PRIVATE KEY":
			key, err := x509.ParsePKCS8PrivateKey(block.Bytes)
			if err != nil {
				return nil, fmt.Errorf("parse PKCS8 private key: %w", err)
			}
			rsaPriv, ok := key.(*rsa.PrivateKey)
			if !ok {
				return nil, ErrNotRSAPrivate
			}
			return rsaPriv, nil

		default:
			continue
		}
	}
	if !found {
		return nil, ErrNoPEMBlocks
	}
	return nil, ErrUnsupportedPEM
}
