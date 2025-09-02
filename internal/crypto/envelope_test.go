package crypto

import (
	"bytes"
	"compress/gzip"
	"crypto/rand"
	"crypto/rsa"
	"testing"
)

func genKey(t *testing.T) (*rsa.PrivateKey, *rsa.PublicKey) {
	t.Helper()
	priv, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("gen key: %v", err)
	}
	return priv, &priv.PublicKey
}

func gz(b []byte) []byte {
	var buf bytes.Buffer
	zw := gzip.NewWriter(&buf)
	_, _ = zw.Write(b)
	_ = zw.Close()
	return buf.Bytes()
}

func TestEncryptDecryptEnvelope_OK(t *testing.T) {
	priv, pub := genKey(t)
	plain := gz([]byte(`{"hello":"world","n":123}`))

	env, err := EncryptEnvelope(pub, plain)
	if err != nil {
		t.Fatalf("encrypt: %v", err)
	}

	got, err := DecryptEnvelope(priv, env)
	if err != nil {
		t.Fatalf("decrypt: %v", err)
	}

	if !bytes.Equal(got, plain) {
		t.Fatalf("mismatch: got %dB, want %dB", len(got), len(plain))
	}
}

func TestEncryptEnvelope_NilKey(t *testing.T) {
	_, err := EncryptEnvelope(nil, []byte("x"))
	if err == nil {
		t.Fatal("want error")
	}
}

func TestDecryptEnvelope_NilKey(t *testing.T) {
	_, err := DecryptEnvelope(nil, []byte("{}"))
	if err == nil {
		t.Fatal("want error")
	}
}

func TestDecryptEnvelope_BadJSON(t *testing.T) {
	priv, _ := genKey(t)
	_, err := DecryptEnvelope(priv, []byte("{"))
	if err == nil {
		t.Fatal("want error")
	}
}

func TestDecryptEnvelope_Tamper_CT(t *testing.T) {
	priv, pub := genKey(t)
	plain := gz([]byte(`{"x":1}`))
	env, err := EncryptEnvelope(pub, plain)
	if err != nil {
		t.Fatalf("encrypt: %v", err)
	}

	// ковыряем байт в base64(ct)
	idx := bytes.LastIndex(env, []byte(`"ct":"`))
	if idx < 0 {
		t.Fatal("no ct field?")
	}
	start := idx + len(`"ct":"`)
	for i := start; i < len(env); i++ {
		if env[i] != '"' { // меняем первый байт полезной части
			env[i] ^= 1
			break
		}
	}
	if _, err := DecryptEnvelope(priv, env); err == nil {
		t.Fatal("want auth failure on tamper")
	}
}

func TestEncryptEnvelope_RandomizedCipher_Differs(t *testing.T) {
	_, pub := genKey(t)
	plain := gz([]byte("same input"))
	e1, _ := EncryptEnvelope(pub, plain)
	e2, _ := EncryptEnvelope(pub, plain)
	if bytes.Equal(e1, e2) {
		t.Fatal("cipher envelopes should differ due to random IV/key")
	}
}
