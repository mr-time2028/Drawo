package responses

import (
	"drawo/internal/modules/user/models"
)

type User struct {
	ID       string `json:"id"`
	Username string `json:"name"`
}

func ToUser(user *models.User) *User {
	return &User{
		ID:       user.ID,
		Username: user.Username,
	}
}
