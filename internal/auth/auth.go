package auth

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/alexedwards/argon2id"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

type TokenType string

type jwtClaims struct {
	IsAdmin bool `json:"is_admin"`
	jwt.RegisteredClaims
}

const (
	// TokenTypeAccess -
	TokenTypeAccess TokenType = "movie-reserve-access"
	// JwtExpiresIn -
	JwtExpiresIn time.Duration = time.Hour
	// RefreshTokenExpiresIn -
	RefreshTokenExpiresIn time.Duration = 60 * 24 * time.Hour
)

func HashPassword(password string) (string, error) {
	return argon2id.CreateHash(password, argon2id.DefaultParams)
}

func CheckPasswordHash(password, hash string) (bool, error) {
	return argon2id.ComparePasswordAndHash(password, hash)
}

func MakeJWT(userID uuid.UUID, isAdmin bool, tokenSecret string, expiresIn time.Duration) (string, error) {
	claims := jwtClaims{
		IsAdmin: isAdmin,
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    string(TokenTypeAccess),
			IssuedAt:  jwt.NewNumericDate(time.Now().UTC()),
			ExpiresAt: jwt.NewNumericDate(time.Now().UTC().Add(expiresIn)),
			Subject:   userID.String(),
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)

	return token.SignedString([]byte(tokenSecret))
}

func ValidateJWT(tokenString, tokenSecret string) (uuid.UUID, bool, error) {
	claims := &jwtClaims{}
	tkn, err := jwt.ParseWithClaims(tokenString, claims, func(token *jwt.Token) (any, error) {
		return []byte(tokenSecret), nil
	})
	if err != nil {
		return uuid.Nil, false, err
	}
	if !tkn.Valid {
		return uuid.Nil, false, errors.New("invalid token")
	}

	if claims.Issuer != string(TokenTypeAccess) {
		return uuid.Nil, false, errors.New("invalid issuer")
	}

	userID, err := uuid.Parse(claims.Subject)
	if err != nil {
		return uuid.Nil, false, fmt.Errorf("invalid user ID: %w", err)
	}
	return userID, claims.IsAdmin, nil
}

func GetBearerToken(headers http.Header) (string, error) {
	auth := headers.Get("Authorization")
	if auth == "" {
		return "", errors.New("no authorization found in header")
	}

	splitAuth := strings.Split(auth, " ")
	if len(splitAuth) < 2 || splitAuth[0] != "Bearer" {
		return "", errors.New("malformed authorization header")
	}

	return splitAuth[1], nil
}

func MakeRefreshToken() string {
	data := make([]byte, 32)
	rand.Read(data)
	return hex.EncodeToString(data)
}
