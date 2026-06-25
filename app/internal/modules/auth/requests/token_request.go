package requests

type RefreshTokenRequest struct {
	RefreshToken string `json:"refresh_token" binding:"required"`
}

type AccessTokenRequest struct {
	AccessToken string `json:"access_token" binding:"required"`
}
