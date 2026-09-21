package token

import (
	"errors"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/go-redis/redis"
	"github.com/golang-jwt/jwt"
)

func signingKey() []byte {
	if key := os.Getenv("JWT_SECRET"); key != "" {
		return []byte(key)
	}
	return []byte("tibsphereai-dev-secret")
}

type Tokens struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
}

func GenerateJWTToken(userID string, role string, rdb *redis.Client) *Tokens {
	accessToken := jwt.New(jwt.SigningMethodHS256)
	refreshToken := jwt.New(jwt.SigningMethodHS256)
	sessionID := fmt.Sprintf("%d", time.Now().UnixNano())

	claims := accessToken.Claims.(jwt.MapClaims)
	claims["user_id"] = userID
	claims["role"] = role
	claims["session_id"] = sessionID
	claims["iat"] = time.Now().Unix()
	claims["exp"] = time.Now().Add(4320 * time.Hour).Unix() // Token expires in 180 days
	access, err := accessToken.SignedString(signingKey())
	if err != nil {
		log.Fatal("error while generating access token : ", err)
	}

	rftClaims := refreshToken.Claims.(jwt.MapClaims)
	rftClaims["user_id"] = userID
	rftClaims["role"] = role
	rftClaims["session_id"] = sessionID
	rftClaims["iat"] = time.Now().Unix()
	rftClaims["exp"] = time.Now().Add(4320 * time.Hour).Unix() // Refresh token expires in 180 days
	refresh, err := refreshToken.SignedString(signingKey())
	if err != nil {
		log.Fatal("error while generating refresh token : ", err)
	}

	redisKey := "user_session:" + userID
	_ = rdb.Set(redisKey, sessionID, 4320*time.Hour).Err() // Session expires in 180 days

	return &Tokens{
		AccessToken:  access,
		RefreshToken: refresh,
	}
}

func ValidateToken(tokenStr string) (bool, error) {
	_, err := ExtractClaim(tokenStr)
	if err != nil {
		return false, err
	}
	return true, nil
}

func ExtractClaim(tokenStr string) (jwt.MapClaims, error) {
	token, err := jwt.Parse(tokenStr, func(token *jwt.Token) (interface{}, error) {
		return signingKey(), nil
	})
	if err != nil {
		return nil, fmt.Errorf("parsing token: %w", err)
	}
	if !token.Valid {
		return nil, errors.New("invalid token")
	}

	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		return nil, errors.New("invalid token claims")
	}

	return claims, nil
}
