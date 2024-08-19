package db_models

import "gorm.io/gorm"

type UserLocation struct {
	gorm.Model `swaggerignore:"true"`
	UserID     uint    `json:"user_id"`
	Latitude   float64 `json:"latitude"`
	Longitude  float64 `json:"longitude"`
	Type       Type    `json:"type"`
	User       User    `json:"user"`
}

type Type string

const (
	Wish    Type = "Wish"
	NotGood Type = "NotGood"
	Good    Type = "Good"
)
