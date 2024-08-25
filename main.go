package main

import (
	"fmt"
	"github.com/gorilla/mux"
	"github.com/olahol/melody"
	"github.com/rs/cors"
	httpSwagger "github.com/swaggo/http-swagger"
	"log"
	"net/http"
	authhandlers "svm/api/auth"
	"svm/api/user"
	authToken "svm/auth/token"
	_ "svm/docs" // Swagger dokümantasyonunu içe aktar
	"svm/middleware"
	"svm/migrations"
	"svm/models/db_models"
)

// @title MyApp API
// @version 1.0
// @description This is a sample server for MyApp.
// @termsOfService http://myapp.com/terms/

// @contact.name API Support
// @contact.url http://www.swagger.io/support
// @contact.email support@swagger.io

// @license.name Apache 2.0
// @license.url http://www.apache.org/licenses/LICENSE-2.0.html

// @host localhost:8080
// @BasePath /

// @securityDefinitions.apikey BearerAuth
// @in header
// @name Authorization
func main() {

	db, err := migrations.CreateDb()
	if err != nil {
		return
	}

	tokenStore := authToken.NewTokenStore("localhost:6379")

	// Melody instance'ını oluşturun
	m := melody.New()

	// WebSocket olaylarını ele alacak işlevleri ayarlayın
	m.HandleConnect(handleWsCon())
	m.HandleDisconnect(handleWsDisc())
	m.HandleMessage(func(s *melody.Session, msg []byte) {
		// Tüm kullanıcılara mesajı yayınlama
		user_id := s.Request.URL.Query().Get("user_id")

		//Fecth user from db and preload friends
		var user db_models.User
		db.Preload("Friends").First(&user, user_id)

		//Send message to all friends
		for _, friend := range user.Friends {
			if session, ok := authhandlers.UserSessions[fmt.Sprintf("%d", friend.ID)]; ok {
				session.Write(msg)
			}
		}

	})

	router := mux.NewRouter()

	// WebSocket endpoint
	router.HandleFunc("/ws", handleWs(m))

	// Swagger endpoint
	router.PathPrefix("/swagger/").Handler(httpSwagger.WrapHandler)

	// authToken middleware
	api := router.PathPrefix("/api").Subrouter()

	// Public endpoints
	api.HandleFunc("/login", authhandlers.Login(db, tokenStore, m)).Methods("POST")
	api.HandleFunc("/refresh-token", authhandlers.RefreshToken(db, tokenStore)).Methods("POST")
	api.HandleFunc("/logout", authhandlers.Logout(tokenStore)).Methods("POST")
	api.HandleFunc("/online-users", authhandlers.GetOnlineUsers(tokenStore)).Methods("GET")
	api.HandleFunc("/register", user.CreateUser(db)).Methods("POST")

	users := api.PathPrefix("/users").Subrouter()
	users.Use(middleware.JWTAuthentication)

	// Protected user operations under /api
	users.HandleFunc("/{id}", user.UpdateUser(db)).Methods("PUT")
	users.HandleFunc("", user.ListUsers(db)).Methods("GET")
	users.HandleFunc("/{id}", user.DeleteUser(db)).Methods("DELETE")
	users.HandleFunc("/{id}", user.GetUserByID(db)).Methods("GET")
	users.HandleFunc("/friends", user.AddFriend(db)).Methods("POST")
	users.HandleFunc("/location", user.AddUserLocation(db)).Methods("POST")

	//Online users endpoint

	// CORS middleware'i ekleyin
	corsHandler := cors.New(cors.Options{
		AllowedOrigins:   []string{"www.ismailsancar.com", "ismailsancar.com"},
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE"},
		AllowedHeaders:   []string{"Authorization", "Content-Type"},
		AllowCredentials: true,
	}).Handler(router)

	err = http.ListenAndServe(":8080", corsHandler)

	if err != nil {
		log.Fatalf("ListenAndServeTLS: %v", err)
	}
}

func handleWsCon() func(s *melody.Session) {
	return func(s *melody.Session) {
		// Oturum açan kullanıcının ID'sini oturumun sorgu parametrelerinden alıyoruz
		userID := s.Request.URL.Query().Get("user_id")
		if userID != "" {
			authhandlers.UserSessions[userID] = s
		}
	}
}

func handleWsDisc() func(s *melody.Session) {
	return func(s *melody.Session) {
		// Bağlantıyı kesen kullanıcıyı userSessions haritasından kaldırıyoruz
		for userID, session := range authhandlers.UserSessions {
			if session == s {
				delete(authhandlers.UserSessions, userID)
				break
			}
		}
	}
}

func handleWs(m *melody.Melody) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		err := m.HandleRequest(w, r)
		if err != nil {
			log.Println(err)
		}
	}
}
