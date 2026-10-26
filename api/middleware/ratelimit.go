package middleware

import (
	"fmt"
	"strconv"
	"time"

	"tibsphereai/internal/ratelimit"
	"tibsphereai/pkg/errs"

	"github.com/gin-gonic/gin"
	"github.com/go-redis/redis"
	"github.com/golang-jwt/jwt"
)

// KeyFunc - rate limit kalitini (IP, foydalanuvchi, token va h.k.) aniqlaydi; bo'sh qaytarsa limit qo'llanmaydi.
type KeyFunc func(c *gin.Context) string

// ByIP - mijoz IP manzili bo'yicha.
func ByIP(c *gin.Context) string { return c.ClientIP() }

// ByUser - JWT'dagi user_id bo'yicha (auth middleware'dan KEYIN ishlatiladi); topilmasa IP.
func ByUser(c *gin.Context) string {
	if claims, ok := c.Get("claims"); ok {
		if mc, ok := claims.(jwt.MapClaims); ok {
			if id, _ := mc["user_id"].(string); id != "" {
				return "u:" + id
			}
		}
	}
	return c.ClientIP()
}

// Allow - ratelimit.Allow'ga yo'naltiradi (testlar va eski chaqiruvlar uchun).
func Allow(rdb *redis.Client, key string, limit int, window time.Duration) (bool, time.Duration) {
	return ratelimit.Allow(rdb, key, limit, window)
}

// RateLimit - name bo'yicha ajratilgan limit: har bir (name, key) uchun window ichida limit ta so'rov.
// Oshsa 429 too_many_requests + Retry-After header.
func RateLimit(rdb *redis.Client, disabled bool, name string, limit int, window time.Duration, keyFn KeyFunc) gin.HandlerFunc {
	return func(c *gin.Context) {
		if disabled {
			c.Next()
			return
		}
		k := keyFn(c)
		if k == "" {
			c.Next()
			return
		}
		ok, retry := Allow(rdb, fmt.Sprintf("rl:%s:%s", name, k), limit, window)
		if !ok {
			secs := int(retry.Seconds()) + 1
			c.Header("Retry-After", strconv.Itoa(secs))
			errs.Send(c, errs.ErrTooManyRequests)
			c.Abort()
			return
		}
		c.Next()
	}
}
