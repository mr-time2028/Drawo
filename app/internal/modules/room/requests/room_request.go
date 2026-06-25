package requests

type RoomRequest struct {
	Name            string `json:"name" binding:"required"`
	Password        string `json:"password"`
	ConfirmPassword string `json:"confirm_password"`
}
