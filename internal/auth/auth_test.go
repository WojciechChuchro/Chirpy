package auth

import (
	"log"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

func TestMakeJWT(t *testing.T) {
	t.Run("Valid JWT Generation", func(t *testing.T) {
		userID := uuid.New()
		tokenSecret := "supersecretkey"
		expiresIn := time.Hour

		token, err := MakeJWT(userID, tokenSecret, expiresIn)
		log.Printf("Token: %s, err %v", token, err)

		assert.NoError(t, err)
		assert.NotEmpty(t, token)

	})
}

func TestValidateJWT(t *testing.T) {
	tokenSecret := "supersecretkey"
	userID := uuid.New()
	expiresIn := time.Hour

	// Create a valid token for testing
	token, err := MakeJWT(userID, tokenSecret, expiresIn)
	if err != nil {
		t.Fatalf("Error generating token: %v", err)
	}

	t.Run("Valid Token", func(t *testing.T) {
		// Test valid token
		parsedUserID, err := ValidateJWT(token, tokenSecret)
		if err != nil {
			t.Errorf("Expected no error, got %v", err)
		}

		if parsedUserID != userID {
			t.Errorf("Expected userID %v, got %v", userID, parsedUserID)
		}
	})

	t.Run("Invalid Token Signature", func(t *testing.T) {
		// Modify the token by changing the secret to simulate an invalid token signature
		invalidSecret := "wrongsecretkey"
		parsedUserID, err := ValidateJWT(token, invalidSecret)
		if err == nil {
			t.Errorf("Expected error, but got nil")
		}

		if parsedUserID != uuid.Nil {
			t.Errorf("Expected no user ID (uuid.Nil), got %v", parsedUserID)
		}
	})

	t.Run("Expired Token", func(t *testing.T) {
		// Create an expired token by using a negative expiration time
		expiredToken, err := MakeJWT(userID, tokenSecret, -time.Hour)
		if err != nil {
			t.Fatalf("Error generating expired token: %v", err)
		}

		parsedUserID, err := ValidateJWT(expiredToken, tokenSecret)
		if err == nil {
			t.Errorf("Expected error for expired token, but got nil")
		}

		if parsedUserID != uuid.Nil {
			t.Errorf("Expected no user ID (uuid.Nil), got %v", parsedUserID)
		}
	})

	t.Run("Invalid Subject", func(t *testing.T) {
		// Create a token with an invalid subject (non-UUID string)
		invalidToken := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.RegisteredClaims{
			Issuer:    "chirpy",
			Subject:   "invalid-uuid", // Invalid UUID format
			IssuedAt:  jwt.NewNumericDate(time.Now().UTC()),
			ExpiresAt: jwt.NewNumericDate(time.Now().UTC().Add(time.Hour)),
		})

		// Sign the invalid token with the correct secret
		tokenString, err := invalidToken.SignedString([]byte(tokenSecret))
		if err != nil {
			t.Fatalf("Error signing token: %v", err)
		}

		parsedUserID, err := ValidateJWT(tokenString, tokenSecret)
		if err == nil {
			t.Errorf("Expected error for invalid subject, but got nil")
		}

		if parsedUserID != uuid.Nil {
			t.Errorf("Expected no user ID (uuid.Nil), got %v", parsedUserID)
		}
	})
}
