package migration

import (
	"drawo/internal/modules/user/models"
	"drawo/pkg/database"
	"fmt"
	"log"
)

func Migrate() {
	database.Connect()

	db := database.GetDB()

	err := db.AutoMigrate(
		&models.User{},
	)
	if err != nil {
		log.Fatal("Migration failed: ", err.Error())
	}

	fmt.Println("Migration done...")
}
