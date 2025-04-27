package auth

import (
	"errors"
	"log"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
)

func HashPassword(password string) (string, error) {
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		log.Printf("Error while hashing password %v", err)
		return "", err
	}

	return string(hashedPassword), nil
}

func CheckPasswordHash(hash, password string) error {
	err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))

	if err != nil {
		log.Printf("Error while hashing password %v", err)
		return err
	}

	return nil
}
func MakeJWT(userID uuid.UUID, tokenSecret string, expiresIn time.Duration) (string, error) {
	currentTime := time.Now().UTC()
	expiriationTime := time.Now().Add(expiresIn).UTC()
	token := jwt.NewWithClaims(jwt.SigningMethodHS256,
		jwt.RegisteredClaims{
			Issuer:    "chirpy",
			IssuedAt:  &jwt.NumericDate{currentTime},
			ExpiresAt: &jwt.NumericDate{expiriationTime},
			Subject:   userID.String(),
		},
	)

	signed, err := token.SignedString([]byte(tokenSecret))
	if err != nil {
		return "", err
	}

	return signed, nil

}

func ValidateJWT(tokenString, tokenSecret string) (uuid.UUID, error) {
	token, err := jwt.ParseWithClaims(tokenString, &jwt.RegisteredClaims{}, func(token *jwt.Token) (any, error) {
		return []byte(tokenSecret), nil
	})
	if err != nil {
		log.Printf("Invalid JWT: %v", err)
		return uuid.Nil, err
	}

	if claims, ok := token.Claims.(*jwt.RegisteredClaims); ok && token.Valid {
		subjectUUID, err := uuid.Parse(claims.Subject)
		if err != nil {
			return uuid.Nil, errors.New("invalid subject UUID")
		}
		return subjectUUID, nil
	} else {
		return uuid.Nil, errors.New("invalid token claims")
	}
}
