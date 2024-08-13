package requests

type RegisterRequest struct {
	Username        string `json:"username" binding:"required,min=3,max=100"`
	Password        string `json:"password" binding:"required,min=8,max=30"`
	ConfirmPassword string `json:"confirm_password" binding:"required,min=8,max=30"`
}
