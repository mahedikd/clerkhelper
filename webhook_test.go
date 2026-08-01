package clerkhelper

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"testing"

	"github.com/stretchr/testify/assert"
)

func signSvix(id, ts string, body, key []byte) string {
	mac := hmac.New(sha256.New, key)
	mac.Write([]byte(id))
	mac.Write([]byte("."))
	mac.Write([]byte(ts))
	mac.Write([]byte("."))
	mac.Write(body)
	return "v1," + base64.StdEncoding.EncodeToString(mac.Sum(nil))
}

func TestVerifySvixSignature(t *testing.T) {
	key := []byte("0123456789abcdef0123456789abcdef")
	secret := "whsec_" + base64.StdEncoding.EncodeToString(key)
	body := []byte(`{"type":"user.created","data":{}}`)

	t.Run("accepts a valid signature", func(t *testing.T) {
		sig := signSvix("id_1", "1700000000", body, key)
		assert.NoError(t, VerifySvixSignature(body, "id_1", "1700000000", sig, secret))
	})

	t.Run("accepts a signature among multiple", func(t *testing.T) {
		sig := signSvix("id_1", "1700000000", body, key)
		multi := "v1," + base64.StdEncoding.EncodeToString(make([]byte, 32)) + " " + sig
		assert.NoError(t, VerifySvixSignature(body, "id_1", "1700000000", multi, secret))
	})

	t.Run("rejects a tampered body", func(t *testing.T) {
		sig := signSvix("id_1", "1700000000", body, key)
		assert.Error(t, VerifySvixSignature([]byte(`{"type":"user.deleted"}`), "id_1", "1700000000", sig, secret))
	})

	t.Run("rejects a wrong secret", func(t *testing.T) {
		sig := signSvix("id_1", "1700000000", body, []byte("ffffffffffffffffffffffffffffffff"))
		assert.Error(t, VerifySvixSignature(body, "id_1", "1700000000", sig, secret))
	})

	t.Run("rejects missing svix headers", func(t *testing.T) {
		sig := signSvix("id_1", "1700000000", body, key)
		assert.Error(t, VerifySvixSignature(body, "", "", sig, secret))
	})

	t.Run("rejects invalid secret format", func(t *testing.T) {
		sig := signSvix("id_1", "1700000000", body, key)
		assert.Error(t, VerifySvixSignature(body, "id_1", "1700000000", sig, "not-a-secret"))
	})
}
