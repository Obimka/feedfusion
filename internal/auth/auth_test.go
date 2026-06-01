package auth

import (
	"testing"
	"time"
)

func TestHashPassword(t *testing.T) {
	password := "my-secret-password"
	hashed, err := HashPassword(password)
	if err != nil {
		t.Fatalf("HashPassword failed: %v", err)
	}

	if hashed == password {
		t.Error("Hashed password should not equal original password")
	}
}

func TestCheckPassword(t *testing.T) {
	password := "my-secret-password"
	hashed, err := HashPassword(password)
	if err != nil {
		t.Fatalf("HashPassword failed: %v", err)
	}

	if !CheckPassword(password, hashed) {
		t.Error("CheckPassword should return true for correct password")
	}

	if CheckPassword("wrong-password", hashed) {
		t.Error("CheckPassword should return false for wrong password")
	}
}

func TestGenerateJWT(t *testing.T) {
	config := JWTConfig{
		SecretKey:  "test-secret-key-12345",
		Expiration: time.Hour,
	}

	user := &User{
		ID:       1,
		Username: "testuser",
		Password: "hashed-password",
		IsAdmin:  true,
	}

	token, err := GenerateJWT(user, config)
	if err != nil {
		t.Fatalf("GenerateJWT failed: %v", err)
	}

	if token == "" {
		t.Error("Generated token should not be empty")
	}

	// Validate the token
	claims, err := ValidateJWT(token, config)
	if err != nil {
		t.Fatalf("ValidateJWT failed: %v", err)
	}

	if claims.UserID != user.ID {
		t.Errorf("Expected UserID %d, got %d", user.ID, claims.UserID)
	}
	if claims.Username != user.Username {
		t.Errorf("Expected Username %s, got %s", user.Username, claims.Username)
	}
	if !claims.IsAdmin {
		t.Error("Expected IsAdmin to be true")
	}
}

func TestGenerateRefreshToken(t *testing.T) {
	config := JWTConfig{
		SecretKey:   "test-secret-key-12345",
		Expiration:  time.Hour,
		RefreshExpiry: 7 * 24 * time.Hour,
	}

	user := &User{
		ID:       1,
		Username: "testuser",
		IsAdmin:  false,
	}

	token, err := GenerateRefreshToken(user, config)
	if err != nil {
		t.Fatalf("GenerateRefreshToken failed: %v", err)
	}

	if token == "" {
		t.Error("Generated refresh token should not be empty")
	}

	// Validate the token
	claims, err := ValidateJWT(token, config)
	if err != nil {
		t.Fatalf("ValidateJWT failed: %v", err)
	}

	if claims.UserID != user.ID {
		t.Errorf("Expected UserID %d, got %d", user.ID, claims.UserID)
	}
}

func TestValidateJWT_InvalidToken(t *testing.T) {
	config := JWTConfig{
		SecretKey: "test-secret-key-12345",
	}

	_, err := ValidateJWT("invalid-token", config)
	if err == nil {
		t.Error("Expected error for invalid token")
	}

	if err != ErrTokenInvalid {
		t.Errorf("Expected ErrTokenInvalid, got: %v", err)
	}
}

func TestValidateJWT_WrongSecret(t *testing.T) {
	config := JWTConfig{
		SecretKey:  "test-secret-key-12345",
		Expiration: time.Hour,
	}

	user := &User{
		ID:       1,
		Username: "testuser",
	}

	token, _ := GenerateJWT(user, config)

	// Try to validate with wrong secret
	wrongConfig := JWTConfig{
		SecretKey: "wrong-secret-key",
	}

	_, err := ValidateJWT(token, wrongConfig)
	if err == nil {
		t.Error("Expected error for wrong secret")
	}
}

func TestValidateJWT_ExpiredToken(t *testing.T) {
	config := JWTConfig{
		SecretKey:  "test-secret-key-12345",
		Expiration: time.Nanosecond, // Token expires immediately
	}

	user := &User{
		ID:       1,
		Username: "testuser",
	}

	token, _ := GenerateJWT(user, config)

	// Small delay to ensure token is expired
	time.Sleep(10 * time.Millisecond)

	_, err := ValidateJWT(token, config)
	if err == nil {
		t.Error("Expected error for expired token")
	}

	if err != ErrTokenExpired {
		t.Errorf("Expected ErrTokenExpired, got: %v", err)
	}
}

func TestDefaultJWTConfig(t *testing.T) {
	config := DefaultJWTConfig()

	if config.SecretKey == "" {
		t.Error("Default secret key should not be empty")
	}

	if config.Expiration == 0 {
		t.Error("Default expiration should not be zero")
	}

	if config.RefreshExpiry == 0 {
		t.Error("Default refresh expiry should not be zero")
	}
}
