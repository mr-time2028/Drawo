package migration

import (
	messageModel "drawo/internal/modules/message/models"
	roomModel "drawo/internal/modules/room/models"
	userModel "drawo/internal/modules/user/models"
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
	err := db.AutoMigrate(
		&userModel.User{},
		&roomModel.Room{},
		&messageModel.Message{},
	)
	if err != nil {
		log.Fatal("Migration failed: ", err.Error())
	}

	fmt.Println("Migration done...")
}
