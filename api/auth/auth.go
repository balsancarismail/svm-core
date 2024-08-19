package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/olahol/melody"
	"gorm.io/gorm"
	"net/http"
	"svm/auth/hashing"
	authJWT "svm/auth/jwt"
	authToken "svm/auth/token"
	"svm/models/db_models"
	"time"
)

var UserSessions = make(map[string]*melody.Session)

type LoginRequest struct {
	Email    string  `json:"email"`
	Password string  `json:"password"`
	Lat      float64 `json:"lat"` // Kullanıcının enlem bilgisi
	Lng      float64 `json:"lng"` // Kullanıcının boylam bilgisi
}

// LoginResponse represents the structure for the login response
type LoginResponse struct {
	AccessToken  string       `json:"access_token"`
	RefreshToken string       `json:"refresh_token"`
	UserResponse UserResponse `json:"user"`
}

type UserResponse struct {
	ID          uint          `json:"id"`
	Name        string        `json:"name"`
	Email       string        `json:"email"`
	HomeAddress string        `json:"home_address"`
	Friends     []string      `json:"friends"`
	Locations   []interface{} `json:"locations"`
}

// Login godoc
// @Summary      User login
// @Description  Authenticate user and return access and refresh tokens
// @Tags         auth
// @Accept       json
// @Produce      json
// @Param        credentials body LoginRequest true "Login credentials"
// @Success      200  {object}  LoginResponse
// @Failure      400  {string}  string "Invalid request payload"
// @Failure      401  {string}  string "Invalid email or password"
// @Router       /api/login [post]
func Login(db *gorm.DB, tokenStore *authToken.TokenStore, m *melody.Melody) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var credentials LoginRequest
		if err := json.NewDecoder(r.Body).Decode(&credentials); err != nil {
			http.Error(w, "Invalid request payload", http.StatusBadRequest)
			return
		}

		var user db_models.User

		// Preload Friends
		if err := db.Preload("Friends").Preload("Locations").Where("email = ?", credentials.Email).First(&user).Error; err != nil {
			http.Error(w, "Invalid email or password", http.StatusUnauthorized)
			return
		}

		if !hashing.CheckPassword(&user, credentials.Password) {
			http.Error(w, "Invalid email or password", http.StatusUnauthorized)
			return
		}

		accessToken, err := authJWT.GenerateAccessToken(user.ID, user.Email, user.Name, getUserFriendsAsEmails(user))
		if err != nil {
			http.Error(w, "Failed to generate access token", http.StatusInternalServerError)
			return
		}

		refreshToken, err := authJWT.GenerateRefreshToken(user.ID)
		if err != nil {
			http.Error(w, "Failed to generate refresh token", http.StatusInternalServerError)
			return
		}

		if err := tokenStore.StoreRefreshToken(user.ID, refreshToken, time.Hour*24*7); err != nil {
			http.Error(w, "Failed to store refresh token", http.StatusInternalServerError)
			return
		}

		response := LoginResponse{
			AccessToken:  accessToken,
			RefreshToken: refreshToken,
			UserResponse: UserResponse{
				ID:          user.ID,
				Name:        user.Name,
				Email:       user.Email,
				HomeAddress: user.HomeAddress,
				Friends:     getUserFriendsAsEmails(user),
				Locations:   getUserLocationsAsResponse(user),
			},
		}

		// Kullanıcıyı online olarak işaretleme
		if err := markUserOnline(context.Background(), fmt.Sprintf("%d", user.Email), tokenStore); err != nil {
			http.Error(w, "Failed to mark user online", http.StatusInternalServerError)
			return
		}

		// Kullanıcının arkadaşlarına WebSocket mesajı gönderme (konum bilgisi ile birlikte)
		sendLoginNotificationToFriends(user, credentials.Lat, credentials.Lng, m)

		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(response)
	}
}

func sendLoginNotificationToFriends(user db_models.User, lat, lng float64, m *melody.Melody) {

	type LocationData struct {
		UserID string  `json:"user_id"`
		Name   string  `json:"name"`
		Lat    float64 `json:"lat"`
		Lng    float64 `json:"lng"`
	}

	locationData := LocationData{
		UserID: fmt.Sprintf("%d", user.ID),
		Name:   user.Name,
		Lat:    lat,
		Lng:    lng,
	}

	// Kullanıcının arkadaşlarının WebSocket bağlantılarına mesaj gönderme
	for _, friend := range user.Friends {
		if session, ok := UserSessions[fmt.Sprintf("%d", friend.ID)]; ok {

			//set location data to []byte
			locationData, err := json.Marshal(locationData)
			if err != nil {
				fmt.Println("Failed to marshal location data:", err)
				return
			}

			err = session.Write(locationData)
			if err != nil {
				fmt.Println("Failed to send location data to friend:", err)
			}

		}
	}
}

func markUserOnline(ctx context.Context, userID string, tokenStore *authToken.TokenStore) error {
	// Kullanıcıyı online olarak işaretleme
	return tokenStore.RedisClient.Set(ctx, userID, "online", time.Minute*15).Err() // 15 dakika boyunca aktif olarak işaretle
}

func markUserOffline(ctx context.Context, userID string, tokenStore *authToken.TokenStore) error {
	return tokenStore.RedisClient.Del(ctx, userID).Err() // Kullanıcıyı online listesinden çıkart
}

func getUserFriendsAsEmails(user db_models.User) []string {
	friends := []string{}
	for _, friend := range user.Friends {
		friends = append(friends, friend.Email)
	}
	return friends
}

func getUserLocationsAsResponse(user db_models.User) []interface{} {
	type loc struct {
		ID        uint    `json:"id"`
		Latitude  float64 `json:"latitude"`
		Longitude float64 `json:"longitude"`
	}
	locations := make([]interface{}, 0)

	for _, location := range user.Locations {
		locations = append(locations, loc{
			ID:        location.ID,
			Latitude:  location.Latitude,
			Longitude: location.Longitude,
		})
	}
	return locations
}

// RefreshTokenRequest represents the structure for the refresh token request
type RefreshTokenRequest struct {
	UserID       uint   `json:"user_id"`
	RefreshToken string `json:"refresh_token"`
}

// RefreshTokenResponse represents the structure for the refresh token response
type RefreshTokenResponse struct {
	AccessToken string `json:"access_token"`
}

// RefreshToken godoc
// @Summary      Refresh access token
// @Description  Generate a new access token using a valid refresh token
// @Tags         auth
// @Accept       json
// @Produce      json
// @Param        request body RefreshTokenRequest true "Refresh token request data"
// @Success      200  {object}  RefreshTokenResponse
// @Failure      400  {string}  string "Invalid request payload"
// @Failure      401  {string}  string "Invalid or expired refresh token"
// @Router       /api/refresh-token [post]
func RefreshToken(db *gorm.DB, tokenStore *authToken.TokenStore) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var request RefreshTokenRequest
		if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
			http.Error(w, "Invalid request payload", http.StatusBadRequest)
			return
		}

		// Token'ı doğrulama
		_, err := authJWT.ValidateToken(request.RefreshToken)
		if err != nil {
			http.Error(w, "Invalid or expired refresh token", http.StatusUnauthorized)
			return
		}

		// Redis'ten refresh token'ı çekme
		storedUserID, err := tokenStore.FetchRefreshToken(request.UserID, request.RefreshToken) // Sadece token ile arama yapıyoruz
		if err != nil {
			http.Error(w, "Invalid refresh token", http.StatusUnauthorized)
			return
		}

		// Kullanıcı ID'si uyuşmuyor mu? Token'ı sil
		if storedUserID != request.UserID {
			_ = tokenStore.DeleteRefreshToken(storedUserID, request.RefreshToken) // Token'ı siliyoruz
			http.Error(w, "Invalid refresh token", http.StatusUnauthorized)
			return
		}

		//User i çekme
		var user db_models.User
		if err := db.Where("id = ?", request.UserID).First(&user).Error; err != nil {
			http.Error(w, "User not found", http.StatusNotFound)
			return
		}

		// Yeni access token oluşturma
		accessToken, err := authJWT.GenerateAccessToken(request.UserID, user.Email, user.Name, getUserFriendsAsEmails(user))
		if err != nil {
			http.Error(w, "Failed to generate access token", http.StatusInternalServerError)
			return
		}

		response := RefreshTokenResponse{
			AccessToken: accessToken,
		}

		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(response)
	}
}

type LogoutRequest struct {
	UserID       uint   `json:"user_id"`
	RefreshToken string `json:"refresh_token"`
}

type LogoutResponse struct {
	Message string `json:"message"`
}

// Logout godoc
// @Summary      User logout
// @Description  Invalidate user tokens and close WebSocket session
// @Tags         auth
// @Accept       json
// @Produce      json
// @Param        request body LogoutRequest true "Logout request data"
// @Success      200  {object}  LogoutResponse
// @Failure      400  {string}  string "Invalid request payload"
// @Failure      401  {string}  string "Invalid or expired refresh token"
// @Router       /api/logout [post]
func Logout(tokenStore *authToken.TokenStore) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var request LogoutRequest
		if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
			http.Error(w, "Invalid request payload", http.StatusBadRequest)
			return
		}

		// Refresh token'ı doğrulama
		storedUserID, err := tokenStore.FetchRefreshToken(request.UserID, request.RefreshToken)
		if err != nil || storedUserID != request.UserID {
			http.Error(w, "Invalid or expired refresh token", http.StatusUnauthorized)
			return
		}

		// Refresh token'ı silme
		if err := tokenStore.DeleteRefreshToken(request.UserID, request.RefreshToken); err != nil {
			http.Error(w, "Failed to delete refresh token", http.StatusInternalServerError)
			return
		}

		// WebSocket oturumunu kapatma
		if session, ok := UserSessions[fmt.Sprintf("%d", request.UserID)]; ok {
			session.Close()
			delete(UserSessions, fmt.Sprintf("%d", request.UserID))
		}

		response := LogoutResponse{
			Message: "Successfully logged out",
		}

		// Kullanıcıyı offline olarak işaretleme
		if err := markUserOffline(context.Background(), fmt.Sprintf("%d", request.UserID), tokenStore); err != nil {
			http.Error(w, "Failed to mark user offline", http.StatusInternalServerError)
			return
		}

		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(response)
	}
}

// GetOnlineUsers godoc
// @Summary      Get online users
// @Description  Retrieve a list of currently online users
// @Tags         auth
// @Produce      json
// @Success      200  {array}  string
// @Failure      500  {string}  string "Internal server error"
// @Router       /api/online-users [get]
func GetOnlineUsers(tokenStore *authToken.TokenStore) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		onlineUsers, err := getOnlineUsers(tokenStore)
		if err != nil {
			http.Error(w, "Failed to retrieve online users", http.StatusInternalServerError)
			return
		}

		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(onlineUsers)
	}
}

func getOnlineUsers(tokenStore *authToken.TokenStore) ([]string, error) {
	ctx := context.Background()
	keys, err := tokenStore.RedisClient.Keys(ctx, "*").Result() // Tüm anahtarları al
	if err != nil {
		return nil, err
	}

	var onlineUsers []string
	for _, key := range keys {
		value, err := tokenStore.RedisClient.Get(ctx, key).Result()
		if err == nil && value == "online" {
			onlineUsers = append(onlineUsers, key)
		}
	}

	return onlineUsers, nil
}
