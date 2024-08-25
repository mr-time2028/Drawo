package auth

import (
	"drawo/pkg/config"
	"github.com/golang-jwt/jwt/v5"
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
