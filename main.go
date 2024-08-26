package main

import (
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/olahol/melody"
	"github.com/rs/cors"
	httpSwagger "github.com/swaggo/http-swagger"
	authhandlers "svm/api/auth"
	"svm/api/user"
	authToken "svm/auth/token"
	_ "svm/docs" // Swagger documentation
	smvmmidlleware "svm/middleware"
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
		log.Fatalf("Failed to create database connection: %v", err)
	}

	tokenStore := authToken.NewTokenStore("localhost:6379")
	m := melody.New()

	// Melody WebSocket handlers
	m.HandleConnect(handleWsCon())
	m.HandleDisconnect(handleWsDisc())
	m.HandleMessage(func(s *melody.Session, msg []byte) {
		userID := s.Request.URL.Query().Get("user_id")

		var user db_models.User
		db.Preload("Friends").First(&user, userID)

		for _, friend := range user.Friends {
			if session, ok := authhandlers.UserSessions[fmt.Sprintf("%d", friend.ID)]; ok {
				session.Write(msg)
			}
		}
	})

	r := chi.NewRouter()

	// Middlewares
	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(middleware.Timeout(60 * time.Second))

	// CORS middleware using rs/cors package
	corsMiddleware := cors.New(cors.Options{
		AllowedOrigins:   []string{"https://example.com"}, // Update with your allowed origins
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Authorization", "Content-Type"},
		ExposedHeaders:   []string{"Link"},
		AllowCredentials: true,
		MaxAge:           300, // Maximum value not to be preflighted
	})

	// Applying the CORS middleware to the router
	r.Use(corsMiddleware.Handler)

	// Public routes
	r.Get("/swagger/*", httpSwagger.WrapHandler)
	r.Post("/api/login", authhandlers.Login(db, tokenStore, m))
	r.Post("/api/refresh-token", authhandlers.RefreshToken(db, tokenStore))
	r.Post("/api/logout", authhandlers.Logout(tokenStore))
	r.Get("/api/online-users", authhandlers.GetOnlineUsers(tokenStore))
	r.Post("/api/register", user.CreateUser(db))

	// Protected routes
	r.Group(func(r chi.Router) {
		r.Use(smvmmidlleware.JWTAuthentication)
		r.Route("/api/users", func(r chi.Router) {
			r.Put("/{id}", user.UpdateUser(db))
			r.Get("/", user.ListUsers(db))
			r.Delete("/{id}", user.DeleteUser(db))
			r.Get("/{id}", user.GetUserByID(db))
			r.Post("/friends", user.AddFriend(db))
			r.Post("/location", user.AddUserLocation(db))
		})
	})

	// WebSocket endpoint
	r.Get("/ws", handleWSConnection(m))

	log.Fatal(http.ListenAndServe(":8080", r))
}

func handleWSConnection(m *melody.Melody) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		if err := m.HandleRequest(w, r); err != nil {
			log.Println(err)
		}
	}
}

func handleWsCon() func(s *melody.Session) {
	return func(s *melody.Session) {
		userID := s.Request.URL.Query().Get("user_id")
		if userID != "" {
			authhandlers.UserSessions[userID] = s
		}
	}
}

func handleWsDisc() func(s *melody.Session) {
	return func(s *melody.Session) {
		for userID, session := range authhandlers.UserSessions {
			if session == s {
				delete(authhandlers.UserSessions, userID)
				break
			}
		}
	}
}
