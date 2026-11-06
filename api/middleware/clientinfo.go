package middleware

import (
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/go-redis/redis"
)

// ClientInfo - mobil/web mijoz yuborgan X-Platform (ios|android|web) va X-App-Version
// sarlavhalarini foydalanuvchiga yozadi (admin statistikasi: platforma/versiya bo'yicha).
// Handler'dan KEYIN ishlaydi (user_id auth middleware'da qo'yiladi); DB'ga faqat qiymat
// o'zgarganda yoki soatiga bir marta yoziladi.
func ClientInfo(rdb *redis.Client, save func(userId, platform, version string)) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Next()
		platform := strings.ToLower(strings.TrimSpace(c.GetHeader("X-Platform")))
		version := strings.TrimSpace(c.GetHeader("X-App-Version"))
		if platform != "ios" && platform != "android" && platform != "web" {
			platform = ""
		}
		if len(version) > 40 {
			version = version[:40]
		}
		if platform == "" && version == "" {
			return
		}
		userId, _ := c.Get("user_id")
		uid, _ := userId.(string)
		if uid == "" {
			return
		}
		val := platform + "|" + version
		key := "client_info:" + uid
		if rdb != nil {
			if cur, err := rdb.Get(key).Result(); err == nil && cur == val {
				return
			}
			rdb.Set(key, val, time.Hour)
		}
		save(uid, platform, version)
	}
}
