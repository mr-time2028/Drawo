package controllers

import "drawo/internal/modules/user/services"

type Controller struct {
	UserService services.UserServiceInterface
}

func New() *Controller {
	return &Controller{
		UserService: services.New(),
	}
}
