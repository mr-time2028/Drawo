package migration

import (
	"drawo/pkg/database"
	"fmt"
	"log"
)

func Migrate() {
	database.Connect()

	db := database.GetDB()

	err := db.AutoMigrate()
	if err != nil {
		log.Fatal("Migration failed: ", err.Error())
	}

	fmt.Println("Migration done...")
}
