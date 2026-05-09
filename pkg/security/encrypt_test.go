package security

import (
	"bytes"
	"testing"
)

func TestEncryptDecrypt(t *testing.T) {
	key := []byte("0123456789abcdef")
	plain := []byte("hello skoll")
	cipherText, err := Encrypt(plain, key)
	if err != nil {
		t.Fatalf("encrypt: %v", err)
	}
	decrypted, err := Decrypt(cipherText, key)
	if err != nil {
		t.Fatalf("decrypt: %v", err)
	}
	if !bytes.Equal(plain, decrypted) {
		t.Fatalf("unexpected decrypt result")
	}
}
