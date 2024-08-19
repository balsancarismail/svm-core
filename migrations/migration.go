package migrations

import (
	"fmt"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
	"svm/auth/hashing"
	"svm/models/db_models"
)

func CreateDb() (*gorm.DB, error) {
	dsn := "host=localhost user=postgres password=123456 dbname=swm port=5432 sslmode=disable TimeZone=Asia/Shanghai"
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Info), // Sorguları loglama
	})
	err = db.AutoMigrate(&db_models.User{}, &db_models.Friend{}, &db_models.UserLocation{})
	if err != nil {
		return nil, err
	}
	seedData(db)
	return db, err
}

// Örnek verilerin eklenmesi
func seedData(db *gorm.DB) {
	john := db_models.User{
		Name:         "John Doe",
		Email:        "john@mail.com",
		HomeAddress:  "123 Main St",
		ShareAddress: true,
	}
	err := hashing.SetPassword(&john, "12")
	if err != nil {
		fmt.Println("Failed to set password	hash:", err)
	}

	jane := db_models.User{
		Name:         "Jane Smith",
		Email:        "jane@mail.com",
		HomeAddress:  "456 Elm St",
		ShareAddress: true,
	}
	err = hashing.SetPassword(&jane, "21")
	if err != nil {
		fmt.Println("Failed to set password	hash:", err)
	}

	db.Create(&john)
	db.Create(&jane)

	// Kullanıcıları birbirine arkadaş olarak ekleme
	db.Model(&john).Association("Friends").Append(&jane)
	db.Model(&jane).Association("Friends").Append(&john)
}
