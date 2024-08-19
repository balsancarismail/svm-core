package db_models

// Friend modeli
type Friend struct {
	UserID   uint `gorm:"primaryKey"`
	FriendID uint `gorm:"primaryKey"`
}
