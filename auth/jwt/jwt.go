package auth

import (
	"github.com/golang-jwt/jwt/v4"
	"time"
)

var jwtSecret = []byte("secret-key") // Bu gizli anahtarı güvenli bir şekilde saklayın

type Claims struct {
	UserID  uint     `json:"userId"`
	Name    string   `json:"name"`
	Email   string   `json:"email"`
	Friends []string `json:"friends"`
	jwt.RegisteredClaims
}

// Access Token Oluşturma
func GenerateAccessToken(userID uint, email, name string, friends []string) (string, error) {
	claims := &Claims{
		UserID:  userID,
		Name:    name,
		Email:   email,
		Friends: friends,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Minute * 15)), // 15 dakika geçerli
			Issuer:    "svm",
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(jwtSecret)
}

// Refresh Token Oluşturma
func GenerateRefreshToken(userID uint) (string, error) {
	claims := &Claims{
		UserID: userID,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Hour * 24 * 7)), // 7 gün geçerli
			Issuer:    "svm",
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(jwtSecret)
}

// Token'ı Doğrulama
func ValidateToken(tokenStr string) (*Claims, error) {
	claims := &Claims{}
	token, err := jwt.ParseWithClaims(tokenStr, claims, func(token *jwt.Token) (interface{}, error) {
		return jwtSecret, nil
	})

	if err != nil || !token.Valid {
		return nil, err
	}

	return claims, nil
}

// ParseJWT gelen bir JWT'yi parse eder ve claim'leri döner
func ParseJWT(tokenStr string) (*Claims, error) {
	claims := &Claims{}
	token, err := jwt.ParseWithClaims(tokenStr, claims, func(token *jwt.Token) (interface{}, error) {
		return jwtSecret, nil
	})

	if err != nil || !token.Valid {
		return nil, err
	}

	return claims, nil
}
