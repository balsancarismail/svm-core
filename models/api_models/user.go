package api_models

// UserLocationResponse represents the structure of the user location data in the response
type UserLocationResponse struct {
	ID        uint    `json:"id"`
	Latitude  float64 `json:"latitude"`
	Longitude float64 `json:"longitude"`
}

// UserResponse represents the structure of the user data in the response
type UserResponse struct {
	ID        uint                   `json:"id"`
	Name      string                 `json:"name"`
	Email     string                 `json:"email"`
	Friends   []FriendResponse       `json:"friends"`
	Locations []UserLocationResponse `json:"locations"`
}

// FriendResponse represents the structure of the friend data in the response
type FriendResponse struct {
	ID    uint   `json:"id"`
	Name  string `json:"name"`
	Email string `json:"email"`
}

// FriendRequest represents the structure for the friend request payload
type FriendRequest struct {
	UserID   uint `json:"user_id"`
	FriendID uint `json:"friend_id"`
}

// CreateUserRequest represents the expected payload for creating a user
type CreateUserRequest struct {
	Email        string `json:"email"`
	HomeAddress  string `json:"homeAddress"`
	Name         string `json:"name"`
	Password     string `json:"password"`
	ShareAddress bool   `json:"shareAddress"`
}

// UpdateUserRequest represents the expected payload for updating a user
type UpdateUserRequest struct {
	HomeAddress  string `json:"homeAddress"`
	Name         string `json:"name"`
	ShareAddress bool   `json:"shareAddress"`
}

type UserLocationRequest struct {
	UserID    uint    `json:"user_id"`
	Latitude  float64 `json:"latitude"`
	Longitude float64 `json:"longitude"`
}
