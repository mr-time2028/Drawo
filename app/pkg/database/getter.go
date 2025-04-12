package database

import "gorm.io/gorm"

func Get() *gorm.DB {
	return DB
}
