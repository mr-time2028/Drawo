package database

import "gorm.io/gorm"

func GetDB() *gorm.DB {
	return DB
}
