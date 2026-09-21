package helper

import (
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"math/big"
	"net/http"
	"net/smtp"
	"net/url"
	"strconv"
	"strings"
	"time"

	"golang.org/x/crypto/pbkdf2"

	"tibsphereai/internal/config"
	"tibsphereai/pkg/models"
)

func SendMessage(message string) error {
	cf := config.Load()
	apiURL := fmt.Sprintf("https://api.telegram.org/bot%s/sendMessage", cf.BOT_TOKEN)
	params := url.Values{}
	params.Add("chat_id", cf.GROUP_ID)
	params.Add("text", message)

	// So'rov yuborish
	resp, err := http.PostForm(apiURL, params)
	if err != nil {
		return fmt.Errorf("failed to send request: %v", err)
	}
	defer resp.Body.Close()

	// Javobni o'qish
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read response body: %v", err)
	}

	// Status kodni tekshirish
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("failed to send message, status code: %d, response: %s", resp.StatusCode, string(body))
	}

	return nil
}

func GenerateOtp() string {
	n, _ := rand.Int(rand.Reader, big.NewInt(1000000))
	return fmt.Sprintf("%06d", n.Int64())
}

func GeneratePassword() string {
	n, _ := rand.Int(rand.Reader, big.NewInt(10000000000))
	return fmt.Sprintf("%06d", n.Int64())
}

func GetStartEndDate(month int) (string, string) {
	start := time.Now().UTC().Add(time.Hour * 5).Format("2006-01-02 15:04:05")
	end := time.Now().UTC().AddDate(0, month, 0).Add(time.Hour * 5).Format("2006-01-02 15:04:05")
	return start, end
}

func StringToJSON(title, body string) (string, string) {
	a, _ := json.Marshal(map[string]string{
		"uz": title,
		"ru": title,
		"en": title,
	})

	b, _ := json.Marshal(map[string]string{
		"uz": body,
		"ru": body,
		"en": body,
	})

	title = string(a)
	body = string(b)

	return title, body
}

func CheckDjangoPBKDF2(password, stored string) (bool, error) {
	parts := strings.Split(stored, "$")
	if len(parts) != 4 {
		return false, errors.New("invalid django hash format")
	}

	if parts[0] != "pbkdf2_sha256" {
		return false, errors.New("not pbkdf2_sha256")
	}

	iter, err := strconv.Atoi(parts[1])
	if err != nil {
		return false, err
	}

	salt := parts[2]
	hashB64 := parts[3]

	want, err := base64.StdEncoding.DecodeString(hashB64)
	if err != nil {
		return false, err
	}

	got := pbkdf2.Key(
		[]byte(password),
		[]byte(salt),
		iter,
		len(want),
		sha256.New,
	)

	return subtle.ConstantTimeCompare(got, want) == 1, nil
}

func PickFromJSONStr(jsonStr string, lang string) (string, error) {
	jsonStr = strings.TrimSpace(jsonStr)
	if jsonStr == "" {
		return "", nil
	}

	var m map[string]string
	if err := json.Unmarshal([]byte(jsonStr), &m); err != nil {
		return "", fmt.Errorf("invalid json string: %w", err)
	}

	if v := strings.TrimSpace(m[lang]); v != "" {
		return v, nil
	}
	if v := strings.TrimSpace(m["uz"]); v != "" {
		return v, nil
	}
	if v := strings.TrimSpace(m["ru"]); v != "" {
		return v, nil
	}
	if v := strings.TrimSpace(m["en"]); v != "" {
		return v, nil
	}
	return "", nil
}

func SendVerificationCode(params models.Params, cfg config.Config) error {

	smtpHost := cfg.BREVO_SERVER
	smtpPort := cfg.BREVO_PORT

	message := fmt.Sprintf(
		"From: iMedTeam <%s>\r\n"+
			"To: %s\r\n"+
			"Subject: %s\r\n"+
			"MIME-Version: 1.0\r\n"+
			"Content-Type: text/html; charset=\"UTF-8\"\r\n\r\n"+
			"%s",
		params.From, params.To, params.Subject, params.HTMLBody,
	)

	auth := smtp.PlainAuth(
		"",
		cfg.BREVO_LOGIN,
		cfg.BREVO_KEY,
		smtpHost,
	)

	return smtp.SendMail(
		smtpHost+":"+smtpPort,
		auth,
		params.From,
		[]string{params.To},
		[]byte(message),
	)
}
