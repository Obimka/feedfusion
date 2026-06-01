package websub

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"strings"
	"testing"
)

func TestVerifySignature(t *testing.T) {
	secret := "test-secret-key"
	body := []byte(`{"hub":"test","self":"http://example.com"}`)

	// Create valid signature
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write(body)
	expectedMAC := mac.Sum(nil)
	validSignature := base64.URLEncoding.EncodeToString(expectedMAC)

	t.Run("valid signature", func(t *testing.T) {
		result := VerifySignature(body, secret, validSignature)
		if !result {
			t.Error("Expected valid signature to verify successfully")
		}
	})

	t.Run("invalid signature", func(t *testing.T) {
		result := VerifySignature(body, secret, "invalid-signature")
		if result {
			t.Error("Expected invalid signature to fail verification")
		}
	})

	t.Run("wrong secret", func(t *testing.T) {
		wrongSecret := "wrong-secret"
		mac2 := hmac.New(sha256.New, []byte(wrongSecret))
		mac2.Write(body)
		wrongSignature := base64.URLEncoding.EncodeToString(mac2.Sum(nil))

		result := VerifySignature(body, secret, wrongSignature)
		if result {
			t.Error("Expected signature with wrong secret to fail verification")
		}
	})

	t.Run("empty body", func(t *testing.T) {
		emptyBody := []byte("")
		mac3 := hmac.New(sha256.New, []byte(secret))
		mac3.Write(emptyBody)
		emptySignature := base64.URLEncoding.EncodeToString(mac3.Sum(nil))

		result := VerifySignature(emptyBody, secret, emptySignature)
		if !result {
			t.Error("Expected empty body with valid signature to verify successfully")
		}
	})

	t.Run("empty signature", func(t *testing.T) {
		result := VerifySignature(body, secret, "")
		if result {
			t.Error("Expected empty signature to fail verification")
		}
	})

	t.Run("signature with sha256 prefix stripped", func(t *testing.T) {
		// The caller should strip sha256= prefix before calling VerifySignature
		// This test verifies that the function works when prefix is already stripped
		prefixedSignature := "sha256=" + validSignature
		// Strip prefix as the real code does
		strippedSignature := strings.TrimPrefix(prefixedSignature, "sha256=")
		result := VerifySignature(body, secret, strippedSignature)
		if !result {
			t.Error("Expected signature with stripped sha256= prefix to verify successfully")
		}
	})
}


