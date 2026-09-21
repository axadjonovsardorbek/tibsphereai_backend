package middleware

import (
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"strings"

	"tibsphereai/api/token"

	"github.com/casbin/casbin/v2"
	"github.com/go-redis/redis"
	"github.com/golang-jwt/jwt"

	"github.com/gin-gonic/gin"
)

const (
	unauthorized = "unauthorized"
)

var ErrTokenExpired = errors.New("token expired")

func NewAuth(enforcer *casbin.Enforcer, rdb *redis.Client) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		allow, err := CheckPermission(ctx.FullPath(), ctx.Request, enforcer)

		if err != nil {
			slog.Error("Error checking permission", "err", err)

			if errors.Is(err, ErrTokenExpired) {
				RequireRefresh(ctx)
				return
			}

			InvalidToken(ctx)
			return
		}

		if !allow {
			RequirePermission(ctx)
			return
		}

		claims, err := ExtractToken(ctx.Request)
		if err != nil {
			if errors.Is(err, ErrTokenExpired) {
				RequireRefresh(ctx)
				return
			}
			InvalidToken(ctx)
			return
		}

		tokenSessionID, ok := claims["session_id"].(string)
		if !ok || tokenSessionID == "" {
			tokenSessionID = ""
		}

		redisKey := "user_session:" + claims["user_id"].(string)
		storedSessionID, err := rdb.Get(redisKey).Result()

		if (err == redis.Nil || storedSessionID == "") && tokenSessionID == "" {
			ctx.Set("claims", claims)
			ctx.Set("user_id", claims["user_id"])
			ctx.Next()
		}
		if err != nil {
			ctx.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"error": "session expired or not found",
			})
			return
		}

		if storedSessionID != tokenSessionID {
			ctx.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"error": "you are logged in from another device",
			})
			return
		}

		ctx.Set("claims", claims)
		ctx.Set("user_id", claims["user_id"])
		ctx.Next()
	}
}
func ExtractToken(r *http.Request) (jwt.MapClaims, error) {
	jwtToken := r.Header.Get("Authorization")
	if jwtToken == "" {
		// WebSocket handshake'da browser custom header qo'sha olmaydi (masalan
		// /mobile/simulation/{id}/ws real-vaqt vitallar monitori) - shu sabab
		// ?token=... query orqali ham qabul qilamiz (faqat header bo'sh bo'lsa).
		if q := r.URL.Query().Get("token"); q != "" {
			jwtToken = "Bearer " + q
		} else {
			return nil, fmt.Errorf("access token missing")
		}
	}
	if strings.Contains(jwtToken, "Basic") {
		return nil, fmt.Errorf("invalid token format")
	}

	tokenString := strings.TrimPrefix(jwtToken, "Bearer ")

	claims, err := token.ExtractClaim(tokenString)
	if err != nil {
		msg := strings.ToLower(err.Error())
		if strings.Contains(msg, "token is expired") || strings.Contains(msg, "expired") {
			return nil, ErrTokenExpired
		}
		return nil, err
	}

	return claims, nil
}

func GetRole(r *http.Request) (string, error) {
	claims, err := ExtractToken(r)
	if err != nil {
		return unauthorized, err
	}

	role, ok := claims["role"].(string)
	if !ok {
		return unauthorized, errors.New("role claim not found")
	}
	return role, nil
}

func CheckPermission(path string, r *http.Request, enforcer *casbin.Enforcer) (bool, error) {
	role, err := GetRole(r)
	if err != nil {
		slog.Error("Error getting role from token", "err", err)
		return false, err
	}

	allowed, err := enforcer.Enforce(role, path, r.Method)
	if err != nil {
		slog.Error("Error during Casbin enforce", "err", err)
		return false, err
	}

	return allowed, nil
}

func InvalidToken(c *gin.Context) {
	c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
		"message": "invalid token",
	})
}

func RequirePermission(c *gin.Context) {
	c.AbortWithStatusJSON(http.StatusForbidden, gin.H{
		"message": "permission denied",
	})
}

func RequireRefresh(c *gin.Context) {
	c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
		"message": "access token expired",
	})
}
