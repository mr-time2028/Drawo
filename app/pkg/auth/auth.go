package auth

import (
	"drawo/pkg/config"
	"errors"
	"fmt"
	"github.com/golang-jwt/jwt/v5"
	"strings"
	"time"
)

type JwtUser struct {
	ID string
}

type TokenPairs struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
}

type Claims struct {
	jwt.RegisteredClaims
	TokenType string
}

func GenerateTokenPair(ju *JwtUser) (*TokenPairs, error) {
	config.SetConfig()
	cfg := config.GetConfig()

	issuer := cfg.Auth.Issuer
	audience := cfg.Auth.Audience
	subject := ju.ID
	secretKey := cfg.App.SecretKey
	accessExpiry := cfg.Auth.AccessExpiry
	refreshExpiry := cfg.Auth.RefreshExpiry

	token := jwt.New(jwt.SigningMethodHS256)

	claims := token.Claims.(jwt.MapClaims)
	claims["sub"] = subject
	claims["aud"] = audience
	claims["iss"] = issuer
	claims["iat"] = time.Now().Unix()
	claims["exp"] = time.Now().Add(accessExpiry * time.Minute).Unix()
	claims["TokenType"] = "access"

	signedAccessToken, err := token.SignedString([]byte(secretKey))
	if err != nil {
		return &TokenPairs{}, err
	}

	refreshToken := jwt.New(jwt.SigningMethodHS256)
	refreshTokenClaims := refreshToken.Claims.(jwt.MapClaims)
	refreshTokenClaims["sub"] = subject
	refreshTokenClaims["aud"] = audience
	refreshTokenClaims["iss"] = issuer
	refreshTokenClaims["iat"] = claims["iat"]
	refreshTokenClaims["exp"] = time.Now().Add(refreshExpiry * time.Minute).Unix()
	refreshTokenClaims["TokenType"] = "refresh"

	signedRefreshToken, err := refreshToken.SignedString([]byte(secretKey))
	if err != nil {
		return &TokenPairs{}, err
	}

	var tokenPairs = &TokenPairs{
		AccessToken:  signedAccessToken,
		RefreshToken: signedRefreshToken,
	}

	return tokenPairs, nil
}

func VerifyAuthHeaderAccessToken(authHeader string) (string, *Claims, error) {
	if authHeader == "" {
		return "", nil, errors.New("there no authorization header")
	}

	headerParts := strings.Split(authHeader, " ")
	if len(headerParts) != 2 {
		return "", nil, errors.New("invalid auth header")
	}

	if headerParts[0] != "Bearer" {
		return "", nil, errors.New("invalid auth header")
	}

	token := headerParts[1]

	claims, err := ParseWithClaims(token)
	if err != nil {
		return "", nil, err
	}

	return token, claims, nil
}

func VerifyAccessToken(accessToken string) (*Claims, error) {
	claims, err := ParseWithClaims(accessToken)
	if err != nil {
		return nil, err
	}

	return claims, nil
}

func ParseWithClaims(token string) (*Claims, error) {
	config.SetConfig()
	cfg := config.GetConfig()

	secretKey := cfg.App.SecretKey
	issuer := cfg.Auth.Issuer

	claims := &Claims{}
	_, err := jwt.ParseWithClaims(token, claims, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return []byte(secretKey), nil
	})
	if err != nil {
		if strings.Contains(err.Error(), "token is expired") {
			return nil, errors.New("token has expired")
		}
		return nil, err
	}

	if claims.TokenType != "access" {
		return nil, errors.New("invalid token")
	}

	if claims.Issuer != issuer {
		return nil, errors.New("invalid issuer")
	}

	if claims.ExpiresAt.Before(time.Now()) {
		return nil, jwt.ErrTokenExpired
	}

	return claims, nil
}
