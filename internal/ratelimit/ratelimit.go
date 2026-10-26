// Package ratelimit - Redis asosidagi oddiy fixed-window hisoblagich (HTTP middleware va service
// qatlami - masalan OTP urinishlari - ikkalasi ham ishlatadi).
package ratelimit

import (
	"time"

	"github.com/go-redis/redis"
)

// Allow - key uchun window ichida limit martadan oshmasa true. Redis ishlamasa (fail-open) true:
// himoya xizmatni to'xtatib qo'ymasligi kerak.
func Allow(rdb *redis.Client, key string, limit int, window time.Duration) (allowed bool, retryAfter time.Duration) {
	if rdb == nil || limit <= 0 {
		return true, 0
	}
	n, err := rdb.Incr(key).Result()
	if err != nil {
		return true, 0
	}
	if n == 1 {
		rdb.Expire(key, window)
	}
	if int(n) > limit {
		ttl, err := rdb.TTL(key).Result()
		if err != nil || ttl <= 0 {
			ttl = window
			rdb.Expire(key, window)
		}
		return false, ttl
	}
	return true, 0
}

// Hits - joriy hisoblagich qiymati (yo'q bo'lsa 0).
func Hits(rdb *redis.Client, key string) int {
	if rdb == nil {
		return 0
	}
	n, err := rdb.Get(key).Int()
	if err != nil {
		return 0
	}
	return n
}
