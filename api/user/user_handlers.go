package user

import (
	"encoding/json"
	"github.com/gorilla/mux"
	"gorm.io/gorm"
	"net/http"
	"strconv"
	"svm/auth/hashing"
	"svm/models/api_models"
	"svm/models/db_models"
)

// CreateUser godoc
// @Summary      Create a new user
// @Description  Create a new user with the given details
// @Security     BearerAuth
// @Tags         users
// @Accept       json
// @Produce      json
// @Param        user body api_models.CreateUserRequest true "User data"
// @Success      201  {object}  api_models.UserResponse
// @Failure      400  {string}  string "Invalid request payload"
// @Failure      500  {string}  string "Failed to create user"
// @Router       /api/users [post]
func CreateUser(db *gorm.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req api_models.CreateUserRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "Invalid request payload", http.StatusBadRequest)
			return
		}

		// Şifreyi hashleyelim
		hashedPassword, err := hashing.HashPassword(req.Password)
		if err != nil {
			http.Error(w, "Failed to hash password", http.StatusInternalServerError)
			return
		}

		// User nesnesine dönüştürme
		user := db_models.User{
			Email:        req.Email,
			HomeAddress:  req.HomeAddress,
			Name:         req.Name,
			PasswordHash: hashedPassword,
			ShareAddress: req.ShareAddress,
		}

		// Kullanıcıyı veritabanına kaydet
		if err := db.Create(&user).Error; err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		// Response için user verisini struct'a dönüştürme
		userResponse := api_models.UserResponse{
			ID:      user.ID,
			Name:    user.Name,
			Email:   user.Email,
			Friends: []api_models.FriendResponse{}, // Boş bir friends listesi ile başlıyoruz
		}

		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(userResponse)
	}
}

// UpdateUser godoc
// @Summary      Update an existing user
// @Description  Update user details by ID, excluding email and password
// @Security     BearerAuth
// @Tags         users
// @Accept       json
// @Produce      json
// @Param        id   path      string                true  "User ID"
// @Param        user body      api_models.UpdateUserRequest true  "Updated user data"
// @Success      200  {object}  api_models.UserResponse
// @Failure      400  {string}  string "Invalid request payload"
// @Failure      404  {string}  string "User not found"
// @Failure      500  {string}  string "Failed to update user"
// @Router       /api/users/{id} [put]
func UpdateUser(db *gorm.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		vars := mux.Vars(r)
		id := vars["id"]

		var user db_models.User
		if err := db.First(&user, id).Error; err != nil {
			http.Error(w, "User not found", http.StatusNotFound)
			return
		}

		var req api_models.UpdateUserRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "Invalid request payload", http.StatusBadRequest)
			return
		}

		// Sadece izin verilen alanları güncelleyelim
		user.HomeAddress = req.HomeAddress
		user.Name = req.Name
		user.ShareAddress = req.ShareAddress

		if err := db.Save(&user).Error; err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		// Response için user verisini struct'a dönüştürme
		userResponse := api_models.UserResponse{
			ID:      user.ID,
			Name:    user.Name,
			Email:   user.Email,
			Friends: []api_models.FriendResponse{}, // Friends listesi burada kullanılabilir
		}

		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(userResponse)
	}
}

// ListUsers godoc
// @Summary      List users
// @Description  Get a list of users with their friends
// @Security     BearerAuth
// @Tags         users
// @Produce      json
// @Param        page     query     int     false  "Page number"
// @Param        pageSize query     int     false  "Number of users per page"
// @Success      200  {array}   api_models.UserResponse
// @Failure      500  {string}  string "Failed to fetch users"
// @Router       /api/users [get]
func ListUsers(db *gorm.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var users []db_models.User
		page, _ := strconv.Atoi(r.URL.Query().Get("page"))
		if page < 1 {
			page = 1
		}
		pageSize, _ := strconv.Atoi(r.URL.Query().Get("pageSize"))
		if pageSize < 1 {
			pageSize = 10
		}

		offset := (page - 1) * pageSize
		if err := db.Preload("Friends").Offset(offset).Limit(pageSize).Find(&users).Error; err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		// Kullanıcıları UserResponse modeline dönüştürme
		var userResponses []api_models.UserResponse
		for _, user := range users {
			// Friends'i map ediyoruz
			var friendResponses []api_models.FriendResponse
			for _, friend := range user.Friends {
				friendResponses = append(friendResponses, api_models.FriendResponse{
					ID:    friend.ID,
					Name:  friend.Name,
					Email: friend.Email,
				})
			}

			// UserResponse struct'ına verileri atıyoruz
			userResponse := api_models.UserResponse{
				ID:      user.ID,
				Name:    user.Name,
				Email:   user.Email,
				Friends: friendResponses,
			}
			userResponses = append(userResponses, userResponse)
		}

		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(userResponses)
	}
}

// DeleteUser godoc
// @Summary      Delete a user
// @Description  Delete a user by ID
// @Security     BearerAuth
// @Tags         users
// @Param        id   path      string  true  "User ID"
// @Success      204  "No Content"
// @Failure      404  {string}  string "User not found"
// @Failure      500  {string}  string "Failed to delete user"
// @Router       /api/users/{id} [delete]
func DeleteUser(db *gorm.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		vars := mux.Vars(r)
		id := vars["id"]

		if err := db.Delete(&db_models.User{}, id).Error; err != nil {
			http.Error(w, "User not found", http.StatusNotFound)
			return
		}

		w.WriteHeader(http.StatusNoContent)
	}
}

// GetUserByID godoc
// @Summary      Get a user by ID
// @Description  Get details of a specific user by ID
// @Security     BearerAuth
// @Tags         users
// @Produce      json
// @Param        id   path      string  true  "User ID"
// @Success      200  {object}  api_models.UserResponse
// @Failure      404  {string}  string "User not found"
// @Failure      500  {string}  string "Failed to fetch user"
// @Router       /api/users/{id} [get]
func GetUserByID(db *gorm.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		vars := mux.Vars(r)
		id := vars["id"]

		var user db_models.User
		if err := db.Preload("Friends").Preload("Locations").First(&user, id).Error; err != nil {
			if err == gorm.ErrRecordNotFound {
				http.Error(w, "User not found", http.StatusNotFound)
			} else {
				http.Error(w, err.Error(), http.StatusInternalServerError)
			}
			return
		}

		// Friends verisini UserResponse struct'ına dönüştürme
		var friendResponses []api_models.FriendResponse
		for _, friend := range user.Friends {
			friendResponses = append(friendResponses, api_models.FriendResponse{
				ID:    friend.ID,
				Name:  friend.Name,
				Email: friend.Email,
			})
		}

		// Locations verisini UserResponse struct'ına dönüştürme
		var locationResponses []api_models.UserLocationResponse
		for _, location := range user.Locations {
			locationResponses = append(locationResponses, api_models.UserLocationResponse{
				ID:        location.ID,
				Latitude:  location.Latitude,
				Longitude: location.Longitude,
			})
		}

		// UserResponse struct'ına verileri atıyoruz
		userResponse := api_models.UserResponse{
			ID:        user.ID,
			Name:      user.Name,
			Email:     user.Email,
			Friends:   friendResponses,
			Locations: locationResponses,
		}

		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(userResponse)
	}
}

// AddFriend godoc
// @Summary      Add a friend
// @Description  Create a friendship between two users
// @Security     BearerAuth
// @Tags         users
// @Accept       json
// @Produce      json
// @Param        friend body api_models.FriendRequest true "Friend request data"
// @Success      201  {string}  string "Friend added successfully"
// @Failure      400  {string}  string "Invalid request payload"
// @Failure      500  {string}  string "Failed to add friend"
// @Router       /api/users/friends [post]
func AddFriend(db *gorm.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var request api_models.FriendRequest
		if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
			http.Error(w, "Invalid request payload", http.StatusBadRequest)
			return
		}

		friend := db_models.Friend{
			UserID:   request.UserID,
			FriendID: request.FriendID,
		}

		// Arkadaşlık ilişkisini her iki yönde de ekleyelim
		if err := db.Create(&friend).Error; err != nil {
			http.Error(w, "Failed to add friend", http.StatusInternalServerError)
			return
		}

		reverseFriend := db_models.Friend{
			UserID:   request.FriendID,
			FriendID: request.UserID,
		}

		if err := db.Create(&reverseFriend).Error; err != nil {
			http.Error(w, "Failed to add reverse friend", http.StatusInternalServerError)
			return
		}

		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode("Friend added successfully")
	}
}

// AddUserLocation godoc
// @Summary      Add a user location
// @Description  Add a location for a user
// @Security     BearerAuth
// @Tags         users
// @Accept       json
// @Produce      json
// @Param        location body api_models.UserLocationRequest true "Location data"
// @Success      201  {object}  db_models.UserLocation
// @Failure      400  {string}  string "Invalid request payload"
// @Failure      500  {string}  string "Failed to add location"
// @Router       /api/users/locations [post]
func AddUserLocation(db *gorm.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req api_models.UserLocationRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "Invalid request payload", http.StatusBadRequest)
			return
		}

		userLocation := db_models.UserLocation{
			UserID:    req.UserID,
			Latitude:  req.Latitude,
			Longitude: req.Longitude,
			Type:      db_models.Wish,
		}

		if err := db.Create(&userLocation).Error; err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		resp := api_models.UserLocationResponse{
			ID:        userLocation.ID,
			Latitude:  userLocation.Latitude,
			Longitude: userLocation.Longitude,
		}

		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(resp)
	}
}
