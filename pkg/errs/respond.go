package errs

import (
	"errors"
	"fmt"

	sms "tibsphereai/internal/helper"

	"github.com/gin-gonic/gin"
	"golang.org/x/exp/slog"
)

// LangFromCtx - so'rovdagi Accept-Language header (yoki ?lang=) bo'yicha tilni aniqlaydi,
// noma'lum/bo'sh bo'lsa "uz"ga tushadi.
func LangFromCtx(c *gin.Context) Lang {
	l := c.GetHeader("Accept-Language")
	if l == "" {
		l = c.Query("lang")
	}
	switch l {
	case "ru":
		return LangRU
	case "en":
		return LangEN
	default:
		return LangUZ
	}
}

type body struct {
	Code    int    `json:"code"`
	Slug    string `json:"slug"`
	Message string `json:"message"`
	Details string `json:"details,omitempty"`
}

// Respond - yagona xato-javob strukturasi: {error: {code, slug, message, details}}.
// err *AppError bo'lsa (masalan service qatlamidan errs.ErrInvalidOTP qaytgan) aynan
// o'shani ishlatadi; oddiy error bo'lsa defaultErr'ga o'raydi (details = err.Error()).
func Respond(c *gin.Context, defaultErr *AppError, err error) {
	var ae *AppError
	if errors.As(err, &ae) {
		respond(c, ae)
		return
	}
	respond(c, defaultErr.Wrap(err))
}

func respond(c *gin.Context, ae *AppError) {
	lang := LangFromCtx(c)
	resp := body{Code: ae.HTTPCode, Slug: ae.Slug, Message: ae.Text.Pick(lang)}
	if ae.err != nil {
		resp.Details = ae.err.Error()
		notify(c, ae)
	}
	c.JSON(ae.HTTPCode, gin.H{"error": resp})
}

// notify - imed patternidagi smssender o'rnini bosadi: 5xx/4xx xatoni telegram guruhga yuboradi (fon rejimida).
func notify(c *gin.Context, ae *AppError) {
	go func() {
		msg := fmt.Sprintf("Status code: %d\nSlug: %s\nEndpoint: %s\nError: %s", ae.HTTPCode, ae.Slug, c.Request.URL, ae.err)
		if sendErr := sms.SendMessage(msg); sendErr != nil {
			slog.Error("error isnt sending to group by bot: ", sendErr)
		}
	}()
}

// Quyidagilar eski `c.JSON(<code>, gin.H{"message": err.Error()})` chaqiruvlari o'rniga ishlatiladi.

func Bad(c *gin.Context, err error)          { Respond(c, ErrValidation, err) }
func Conflict(c *gin.Context, err error)     { Respond(c, ErrConflict, err) }
func Unauthorized(c *gin.Context, err error) { Respond(c, ErrUnauthorized, err) }
func Forbidden(c *gin.Context, err error)    { Respond(c, ErrForbidden, err) }
func NotFound(c *gin.Context, err error)     { Respond(c, ErrNotFound, err) }
func Internal(c *gin.Context, err error)     { Respond(c, ErrInternal, err) }

// Send - AppError'ni to'g'ridan-to'g'ri (default'ga o'ramasdan) qaytarish kerak bo'lganda.
func Send(c *gin.Context, ae *AppError) { respond(c, ae) }
