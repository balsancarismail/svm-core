package middleware

import (
	"context"
	"net/http"
	"strings"
	auth "svm/auth/jwt"
)

func JWTAuthentication(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		authHeader := r.Header.Get("Authorization")
		if authHeader == "" {
			http.Error(w, "Missing Authorization header", http.StatusUnauthorized)
			return
		}

		parts := strings.Split(authHeader, " ")
		if len(parts) != 2 || parts[0] != "Bearer" {
			http.Error(w, "Invalid Authorization header format", http.StatusUnauthorized)
			return
		}

		tokenString := parts[1]
		claims, err := auth.ValidateToken(tokenString)
		if err != nil {
			http.Error(w, "Invalid or expired token", http.StatusUnauthorized)
			return
		}

		// Token geçerli, kullanıcı ID'sini isteğin context'ine ekleyebiliriz
		ctx := r.Context()
		ctx = context.WithValue(ctx, "userID", claims.UserID)
		r = r.WithContext(ctx)

		// Sonraki middleware veya handler'a geç
		next.ServeHTTP(w, r)
	})
}
