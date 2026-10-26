package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/gin-gonic/gin"
	"github.com/go-redis/redis"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newRedis(t *testing.T) (*redis.Client, *miniredis.Miniredis) {
	mr := miniredis.RunT(t)
	return redis.NewClient(&redis.Options{Addr: mr.Addr()}), mr
}

func do(r http.Handler, method, path string, hdr map[string]string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(method, path, nil)
	for k, v := range hdr {
		req.Header.Set(k, v)
	}
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	return w
}

func TestRateLimit_BlocksAfterLimit_PerKey(t *testing.T) {
	gin.SetMode(gin.TestMode)
	rdb, mr := newRedis(t)
	r := gin.New()
	r.GET("/x", RateLimit(rdb, false, "t", 3, time.Minute, func(c *gin.Context) string { return c.Query("k") }), func(c *gin.Context) { c.String(200, "ok") })

	for i := 0; i < 3; i++ {
		assert.Equal(t, 200, do(r, "GET", "/x?k=a", nil).Code)
	}
	w := do(r, "GET", "/x?k=a", nil)
	assert.Equal(t, http.StatusTooManyRequests, w.Code)
	assert.Contains(t, w.Body.String(), "too_many_requests")
	assert.NotEmpty(t, w.Header().Get("Retry-After"))
	// boshqa kalit ta'sirlanmaydi
	assert.Equal(t, 200, do(r, "GET", "/x?k=b", nil).Code)
	// oyna tugagach qayta ochiladi
	mr.FastForward(61 * time.Second)
	assert.Equal(t, 200, do(r, "GET", "/x?k=a", nil).Code)
}

func TestRateLimit_DisabledAndEmptyKey(t *testing.T) {
	gin.SetMode(gin.TestMode)
	rdb, _ := newRedis(t)
	r := gin.New()
	r.GET("/off", RateLimit(rdb, true, "t", 1, time.Minute, ByIP), func(c *gin.Context) { c.String(200, "ok") })
	r.GET("/nokey", RateLimit(rdb, false, "t2", 1, time.Minute, func(*gin.Context) string { return "" }), func(c *gin.Context) { c.String(200, "ok") })
	for i := 0; i < 5; i++ {
		assert.Equal(t, 200, do(r, "GET", "/off", nil).Code)
		assert.Equal(t, 200, do(r, "GET", "/nokey", nil).Code)
	}
}

func TestAllow_FailOpenWhenRedisDown(t *testing.T) {
	rdb, mr := newRedis(t)
	mr.Close()
	ok, _ := Allow(rdb, "k", 1, time.Minute)
	assert.True(t, ok)
	ok, _ = Allow(nil, "k", 1, time.Minute)
	assert.True(t, ok)
}

func TestCaptcha(t *testing.T) {
	gin.SetMode(gin.TestMode)
	good := "good-token"
	prov := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = r.ParseForm()
		w.Header().Set("Content-Type", "application/json")
		if r.Form.Get("secret") == "sek" && r.Form.Get("response") == good {
			w.Write([]byte(`{"success":true}`))
			return
		}
		w.Write([]byte(`{"success":false}`))
	}))
	defer prov.Close()

	v := NewCaptchaVerifier("turnstile", "sek")
	v.Endpoint = prov.URL
	r := gin.New()
	r.POST("/otp", Captcha(v), func(c *gin.Context) { c.String(200, "ok") })

	req := func(tok string) int {
		rq := httptest.NewRequest("POST", "/otp", nil)
		if tok != "" {
			rq.Header.Set("X-Captcha-Token", tok)
		}
		w := httptest.NewRecorder()
		r.ServeHTTP(w, rq)
		return w.Code
	}
	assert.Equal(t, 200, req(good))
	assert.Equal(t, 400, req("bad"))
	assert.Equal(t, 400, req(""))

	// secret yo'q - himoya o'chiq
	off := gin.New()
	off.POST("/otp", Captcha(NewCaptchaVerifier("turnstile", "")), func(c *gin.Context) { c.String(200, "ok") })
	w := httptest.NewRecorder()
	off.ServeHTTP(w, httptest.NewRequest("POST", "/otp", nil))
	assert.Equal(t, 200, w.Code)

	// provayder ishlamasa - rad etiladi
	v2 := NewCaptchaVerifier("turnstile", "sek")
	v2.Endpoint = "http://127.0.0.1:1"
	assert.False(t, v2.Verify("x", ""))
	require.True(t, v.Verify(good, "1.2.3.4"))
}
