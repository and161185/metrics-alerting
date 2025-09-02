package crypto

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
)

const (
	AlgRSAOAEP256 = "RSA-OAEP-256"
	EncAES256GCM  = "AES-256-GCM"
	VerV1         = 1
)

var (
	ErrNilKey       = errors.New("nil key")
	ErrBadParams    = errors.New("bad envelope params")
	ErrBadB64       = errors.New("bad base64 field")
	ErrWrongKeySize = errors.New("wrong AES key size")
	ErrWrongIV      = errors.New("wrong IV size")
	ErrEmptyCipher  = errors.New("empty ciphertext")
)

type Envelope struct {
	V   int    `json:"v"`
	Alg string `json:"alg"`
	Enc string `json:"enc"`
	EK  string `json:"ek"` // base64(rsa(aesKey))
	IV  string `json:"iv"` // base64(nonce 12b)
	CT  string `json:"ct"` // base64(ciphertext+tag)
}

// EncryptEnvelope: plain -> (gzip уже снаружи, если нужно) -> AES-GCM -> RSA-OAEP(key) -> JSON
func EncryptEnvelope(pub *rsa.PublicKey, plain []byte) ([]byte, error) {
	if pub == nil {
		return nil, ErrNilKey
	}

	// AES-256-GCM
	aesKey := make([]byte, 32)
	if _, err := rand.Read(aesKey); err != nil {
		return nil, err
	}
	blk, err := aes.NewCipher(aesKey)
	if err != nil {
		return nil, err
	}
	gcm, err := cipher.NewGCM(blk)
	if err != nil {
		return nil, err
	}
	iv := make([]byte, gcm.NonceSize()) // 12
	if _, err := rand.Read(iv); err != nil {
		return nil, err
	}
	ct := gcm.Seal(nil, iv, plain, nil)

	// RSA-OAEP(SHA-256) для ключа
	ek, err := rsa.EncryptOAEP(sha256.New(), rand.Reader, pub, aesKey, nil)
	if err != nil {
		return nil, err
	}

	env := Envelope{
		V:   VerV1,
		Alg: AlgRSAOAEP256,
		Enc: EncAES256GCM,
		EK:  base64.StdEncoding.EncodeToString(ek),
		IV:  base64.StdEncoding.EncodeToString(iv),
		CT:  base64.StdEncoding.EncodeToString(ct),
	}
	return json.Marshal(env)
}

// DecryptEnvelope: JSON-конверт -> RSA-OAEP(key) -> AES-GCM -> plain (gzipped JSON – если ты так отправлял)
func DecryptEnvelope(priv *rsa.PrivateKey, envBytes []byte) ([]byte, error) {
	if priv == nil {
		return nil, ErrNilKey
	}
	var env Envelope
	if err := json.Unmarshal(envBytes, &env); err != nil {
		return nil, err
	}
	if env.V != VerV1 || env.Alg != AlgRSAOAEP256 || env.Enc != EncAES256GCM {
		return nil, ErrBadParams
	}

	ek, err := base64.StdEncoding.DecodeString(env.EK)
	if err != nil {
		return nil, ErrBadB64
	}
	iv, err := base64.StdEncoding.DecodeString(env.IV)
	if err != nil {
		return nil, ErrBadB64
	}
	ct, err := base64.StdEncoding.DecodeString(env.CT)
	if err != nil {
		return nil, ErrBadB64
	}
	if len(iv) != 12 {
		return nil, ErrWrongIV
	}
	if len(ct) == 0 {
		return nil, ErrEmptyCipher
	}

	aesKey, err := rsa.DecryptOAEP(sha256.New(), rand.Reader, priv, ek, nil)
	if err != nil {
		return nil, err
	}
	if len(aesKey) != 32 {
		return nil, ErrWrongKeySize
	}

	blk, err := aes.NewCipher(aesKey)
	if err != nil {
		return nil, err
	}
	gcm, err := cipher.NewGCM(blk)
	if err != nil {
		return nil, err
	}
	return gcm.Open(nil, iv, ct, nil)
}
