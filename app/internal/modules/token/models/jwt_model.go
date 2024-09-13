package models

import "github.com/golang-jwt/jwt/v5"

type JwtUser struct {
	ID string
}

type JWTTokenPairs struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
}

type JWTClaims struct {
	jwt.RegisteredClaims
	TokenType string
}
