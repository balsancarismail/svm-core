package db_models

import "gorm.io/gorm"

// User modeli
type User struct {
	gorm.Model   `swaggerignore:"true"`
	Name         string         `gorm:"size:100;not null"`
	Email        string         `gorm:"size:100;unique;not null"`
	PasswordHash string         `gorm:"not null"`
	HomeAddress  string         `gorm:"size:255"`
	ShareAddress bool           `gorm:"not null;default:false"`
	Friends      []*User        `gorm:"many2many:friends"`
	Locations    []UserLocation `gorm:"foreignKey:UserID"`
}
