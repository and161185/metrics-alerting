package crypto

import (
	"bytes"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"os"
	"path/filepath"
	"testing"
)

// --- helpers ---

func genRSA(t *testing.T) (*rsa.PrivateKey, *rsa.PublicKey) {
	t.Helper()
	priv, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("gen key: %v", err)
	}
	return priv, &priv.PublicKey
}

func pemPrivPKCS1(t *testing.T, priv *rsa.PrivateKey) []byte {
	t.Helper()
	return pem.EncodeToMemory(&pem.Block{
		Type:  "RSA PRIVATE KEY",
		Bytes: x509.MarshalPKCS1PrivateKey(priv),
	})
}

func pemPrivPKCS8(t *testing.T, priv *rsa.PrivateKey) []byte {
	t.Helper()
	b, err := x509.MarshalPKCS8PrivateKey(priv)
	if err != nil {
		t.Fatalf("marshal pkcs8: %v", err)
	}
	return pem.EncodeToMemory(&pem.Block{
		Type:  "PRIVATE KEY",
		Bytes: b,
	})
}

func pemPubPKIX(t *testing.T, pub *rsa.PublicKey) []byte {
	t.Helper()
	b, err := x509.MarshalPKIXPublicKey(pub)
	if err != nil {
		t.Fatalf("marshal pub pkix: %v", err)
	}
	return pem.EncodeToMemory(&pem.Block{
		Type:  "PUBLIC KEY",
		Bytes: b,
	})
}

func pemPubPKCS1(t *testing.T, pub *rsa.PublicKey) []byte {
	t.Helper()
	b := x509.MarshalPKCS1PublicKey(pub)
	return pem.EncodeToMemory(&pem.Block{
		Type:  "RSA PUBLIC KEY",
		Bytes: b,
	})
}

func pemCertificateDummy() []byte {

	return pem.EncodeToMemory(&pem.Block{
		Type:  "CERTIFICATE",
		Bytes: []byte{0x01, 0x02, 0x03},
	})
}

// --- tests ---

func TestLoadPrivateKeyFromBytes_PKCS1_OK(t *testing.T) {
	priv, _ := genRSA(t)
	p := pemPrivPKCS1(t, priv)

	got, err := LoadPrivateKeyFromBytes(p)
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	if got.N.Cmp(priv.N) != 0 {
		t.Fatalf("mismatch PKCS1")
	}
}

func TestLoadPrivateKeyFromBytes_PKCS8_OK(t *testing.T) {
	priv, _ := genRSA(t)
	p := pemPrivPKCS8(t, priv)

	got, err := LoadPrivateKeyFromBytes(p)
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	if got.N.Cmp(priv.N) != 0 {
		t.Fatalf("mismatch PKCS8")
	}
}

func TestLoadPublicKeyFromBytes_PKIX_OK(t *testing.T) {
	priv, pub := genRSA(t)
	_ = priv
	p := pemPubPKIX(t, pub)

	got, err := LoadPublicKeyFromBytes(p)
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	if got.N.Cmp(pub.N) != 0 {
		t.Fatalf("mismatch PKIX")
	}
}

func TestLoadPublicKeyFromBytes_PKCS1_OK(t *testing.T) {
	_, pub := genRSA(t)
	p := pemPubPKCS1(t, pub)

	got, err := LoadPublicKeyFromBytes(p)
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	if got.N.Cmp(pub.N) != 0 {
		t.Fatalf("mismatch PKCS1")
	}
}

func TestLoadPublicKeyFromBytes_NotRSA(t *testing.T) {

	priv, _ := genRSA(t)
	p := pemPrivPKCS1(t, priv)

	if _, err := LoadPublicKeyFromBytes(p); err == nil {
		t.Fatalf("expected error for non-RSA public")
	}
}

func TestLoadPrivateKeyFromBytes_NotRSA(t *testing.T) {

	_, pub := genRSA(t)
	p := pemPubPKIX(t, pub)

	if _, err := LoadPrivateKeyFromBytes(p); err == nil {
		t.Fatalf("expected error for non-RSA private")
	}
}

func TestLoadPublicKeyFromBytes_NoPEM(t *testing.T) {
	_, err := LoadPublicKeyFromBytes([]byte("   \n"))
	if err == nil || err != ErrNoPEMBlocks {
		t.Fatalf("want ErrNoPEMBlocks, got %v", err)
	}
}

func TestLoadPrivateKeyFromBytes_NoPEM(t *testing.T) {
	_, err := LoadPrivateKeyFromBytes([]byte{})
	if err == nil || err != ErrNoPEMBlocks {
		t.Fatalf("want ErrNoPEMBlocks, got %v", err)
	}
}

func TestLoadPublicKeyFromBytes_UnsupportedPEM(t *testing.T) {

	b := pemCertificateDummy()
	_, err := LoadPublicKeyFromBytes(b)
	if err == nil || err != ErrUnsupportedPEM {
		t.Fatalf("want ErrUnsupportedPEM, got %v", err)
	}
}

func TestLoadPrivateKeyFromBytes_UnsupportedPEM(t *testing.T) {
	b := pemCertificateDummy()
	_, err := LoadPrivateKeyFromBytes(b)
	if err == nil || err != ErrUnsupportedPEM {
		t.Fatalf("want ErrUnsupportedPEM, got %v", err)
	}
}

func TestLoadPublicKeyFromBytes_MultiPEM_IgnoresForeign_TakesValid(t *testing.T) {
	_, pub := genRSA(t)
	buf := bytes.Join([][]byte{pemCertificateDummy(), pemPubPKIX(t, pub)}, []byte{})
	got, err := LoadPublicKeyFromBytes(buf)
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	if got.N.Cmp(pub.N) != 0 {
		t.Fatalf("mismatch multi-PEM")
	}
}

func TestLoadPrivateKeyFromBytes_MultiPEM_IgnoresForeign_TakesValid(t *testing.T) {
	priv, _ := genRSA(t)
	buf := bytes.Join([][]byte{pemCertificateDummy(), pemPrivPKCS1(t, priv)}, []byte{})
	got, err := LoadPrivateKeyFromBytes(buf)
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	if got.N.Cmp(priv.N) != 0 {
		t.Fatalf("mismatch multi-PEM")
	}
}

func TestLoadPublicKeyFromBytes_BadDER(t *testing.T) {
	// correct TYPE, but wrong bytes -> parser error
	block := &pem.Block{Type: "PUBLIC KEY", Bytes: []byte{0xde, 0xad, 0xbe, 0xef}}
	p := pem.EncodeToMemory(block)
	if _, err := LoadPublicKeyFromBytes(p); err == nil {
		t.Fatalf("expected parse error")
	}
}

func TestLoadPrivateKeyFromBytes_BadDER(t *testing.T) {
	block := &pem.Block{Type: "RSA PRIVATE KEY", Bytes: []byte{0xca, 0xfe}}
	p := pem.EncodeToMemory(block)
	if _, err := LoadPrivateKeyFromBytes(p); err == nil {
		t.Fatalf("expected parse error")
	}
}

func TestLoadPrivateKeyFromBytes_EncryptedPEM_Unsupported(t *testing.T) {
	// имитируем зашифрованный PKCS#8: тип "ENCRYPTED PRIVATE KEY"
	block := &pem.Block{Type: "ENCRYPTED PRIVATE KEY", Bytes: []byte{0x01, 0x02}}
	p := pem.EncodeToMemory(block)

	_, err := LoadPrivateKeyFromBytes(p)
	if err == nil || err != ErrUnsupportedPEM {
		t.Fatalf("want ErrUnsupportedPEM, got %v", err)
	}
}

func TestLoadPublicKey_File_OK(t *testing.T) {
	_, pub := genRSA(t)
	dir := t.TempDir()
	path := filepath.Join(dir, "pub.pem")
	if err := os.WriteFile(path, pemPubPKIX(t, pub), 0o600); err != nil {
		t.Fatalf("write: %v", err)
	}
	got, err := LoadPublicKey(path)
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	if got.N.Cmp(pub.N) != 0 {
		t.Fatalf("mismatch")
	}
}

func TestLoadPrivateKey_File_OK(t *testing.T) {
	priv, _ := genRSA(t)
	dir := t.TempDir()
	path := filepath.Join(dir, "priv.pem")
	if err := os.WriteFile(path, pemPrivPKCS1(t, priv), 0o600); err != nil {
		t.Fatalf("write: %v", err)
	}
	got, err := LoadPrivateKey(path)
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	if got.N.Cmp(priv.N) != 0 {
		t.Fatalf("mismatch")
	}
}

func TestLoadPublicKey_File_ReadErr(t *testing.T) {
	_, err := LoadPublicKey(filepath.Join(t.TempDir(), "nope.pem"))
	if err == nil {
		t.Fatalf("expected read error")
	}
}

func TestLoadPrivateKey_File_ReadErr(t *testing.T) {
	_, err := LoadPrivateKey(filepath.Join(t.TempDir(), "nope.pem"))
	if err == nil {
		t.Fatalf("expected read error")
	}
}
