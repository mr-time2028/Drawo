package helpers

import (
	"golang.org/x/crypto/bcrypt"
)

func HashPassword(password string) ([]byte, error) {
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return []byte(""), err
	}
	return hashedPassword, nil
}

func CompareRequestPasswords(password1, password2 string) bool {
	return password1 == password2
}

func CompareRequestAndHashPasswords(requestPassword, userPassword string) error {
	return bcrypt.CompareHashAndPassword([]byte(userPassword), []byte(requestPassword))
}
