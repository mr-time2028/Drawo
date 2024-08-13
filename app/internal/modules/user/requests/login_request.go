package requests

type LoginRequest struct {
	Username string `binding:"required"`
	Password string `binding:"required"`
}
