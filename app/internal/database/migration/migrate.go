package migration

import (
	"drawo/internal/modules/user/models"
	"drawo/pkg/database"
	"fmt"
	"log"
)

func Migrate() {
	// connect to the database
	database.Connect()
	db := database.GetDB()

	// create ossp extension
	result := db.Exec("CREATE EXTENSION IF NOT EXISTS \"uuid-ossp\"")
	if result.Error != nil {
		log.Fatal("Failed to install uuid-ossp extension")
	}
	fmt.Println("Extension uuid-ossp installed successfully")

	// migrations
	err := db.AutoMigrate(&models.User{})
	if err != nil {
		log.Fatal("Migration failed: ", err.Error())
	}

	fmt.Println("Migration done...")
}
