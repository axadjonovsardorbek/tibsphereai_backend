package middleware

import (
	"encoding/json"
	"net/http"
	"net/url"
	"strings"
	"time"

	"tibsphereai/pkg/errs"

	"github.com/gin-gonic/gin"
)

var captchaEndpoints = map[string]string{
	"turnstile": "https://challenges.cloudflare.com/turnstile/v0/siteverify",
	"hcaptcha":  "https://hcaptcha.com/siteverify",
	"recaptcha": "https://www.google.com/recaptcha/api/siteverify",
}

// CaptchaVerifier - captcha token'ini provayderda tekshiradi (Cloudflare Turnstile / hCaptcha / reCAPTCHA).
type CaptchaVerifier struct {
	Provider string
	Secret   string
	Client   *http.Client
	// Endpoint - testlar uchun provayder manzilini almashtirish.
	Endpoint string
}

func NewCaptchaVerifier(provider, secret string) *CaptchaVerifier {
	return &CaptchaVerifier{Provider: strings.ToLower(provider), Secret: secret, Client: &http.Client{Timeout: 8 * time.Second}}
}

// Enabled - secret sozlangan bo'lsa yoqiladi (aks holda middleware hech narsa qilmaydi).
func (v *CaptchaVerifier) Enabled() bool { return v != nil && v.Secret != "" }

func (v *CaptchaVerifier) endpoint() string {
	if v.Endpoint != "" {
		return v.Endpoint
	}
	if e, ok := captchaEndpoints[v.Provider]; ok {
		return e
	}
	return captchaEndpoints["turnstile"]
}

// Verify - token yaroqlimi.
func (v *CaptchaVerifier) Verify(token, ip string) bool {
	if token == "" {
		return false
	}
	form := url.Values{"secret": {v.Secret}, "response": {token}}
	if ip != "" {
		form.Set("remoteip", ip)
	}
	resp, err := v.Client.PostForm(v.endpoint(), form)
	if err != nil {
		return false // provayder ishlamasa - xavfsiz tomon (rad etish)
	}
	defer resp.Body.Close()
	var out struct {
		Success bool `json:"success"`
	}
	if json.NewDecoder(resp.Body).Decode(&out) != nil {
		return false
	}
	return out.Success
}

// Captcha - himoyalangan endpointlar uchun: token `X-Captcha-Token` header'ida (yoki ?captcha_token=).
// Verifier yoqilmagan bo'lsa (CAPTCHA_SECRET bo'sh) o'tkazib yuboradi.
func Captcha(v *CaptchaVerifier) gin.HandlerFunc {
	return func(c *gin.Context) {
		if !v.Enabled() {
			c.Next()
			return
		}
		token := c.GetHeader("X-Captcha-Token")
		if token == "" {
			token = c.Query("captcha_token")
		}
		if !v.Verify(token, c.ClientIP()) {
			errs.Send(c, errs.ErrCaptchaFailed)
			c.Abort()
			return
		}
		c.Next()
	}
}
