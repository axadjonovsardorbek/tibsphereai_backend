package handler

import (
	"net/http"
	"tibsphereai/pkg/errs"

	"github.com/gin-gonic/gin"
)

// TelegramLoginStart godoc
// @Summary Telegram bot orqali OTP'siz kirishni boshlash (token + bot havolasi)
// @Description Ilova bot_url'ni ochadi; foydalanuvchi botda (bir marta raqamini yuborib) tasdiqlaydi.
// @Description Ilova /mobile/auth/telegram/check ni 2 soniyada bir poll qiladi. Captcha yoqilgan bo'lsa X-Captcha-Token header kerak.
// @Tags auth
// @Description client=web - ilova oddiy brauzerda ochiq (Telegram Mini App emas): bot "Ilovaga qaytish" tugmasi oddiy havola bo'ladi (brauzerda sessiya saqlanadi). Bo'sh = Mini App.
// @Produce json
// @Param client query string false "web | miniapp (default miniapp)"
// @Success 200 {object} service.TelegramLinkStartRes
// @Router /mobile/auth/telegram/start [post]
func (h *Handler) TelegramLoginStart(c *gin.Context) {
	res, err := h.Clients.User.TelegramLoginStart(c.Request.Context(), c.Query("client") == "web")
	if err != nil {
		errs.Bad(c, err)
		return
	}
	c.JSON(http.StatusOK, res)
}

// TelegramLoginCheck godoc
// @Summary Telegram orqali kirish natijasini tekshirish (poll)
// @Description status=pending - kutish; status=confirmed - auth ichida access/refresh tokenlar (token bir martalik).
// @Description Token yo'q/muddati o'tgan bo'lsa 400 otp_expired.
// @Tags auth
// @Accept json
// @Produce json
// @Param body body object true "{token, referral_code?}"
// @Success 200 {object} service.TelegramLinkCheckRes
// @Router /mobile/auth/telegram/check [post]
func (h *Handler) TelegramLoginCheck(c *gin.Context) {
	var body struct {
		Token        string `json:"token"`
		ReferralCode string `json:"referral_code"`
	}
	if err := c.BindJSON(&body); err != nil || body.Token == "" {
		errs.Bad(c, errOr(err, "token is required"))
		return
	}
	res, err := h.Clients.User.TelegramLoginCheck(c.Request.Context(), body.Token, body.ReferralCode)
	if err != nil {
		errs.Bad(c, err)
		return
	}
	c.JSON(http.StatusOK, res)
}

// PhoneLinkStart godoc
// @Summary Hisobga telefon raqam ulash/almashtirishni boshlash (Telegram orqali tasdiqlanadi)
// @Tags user
// @Produce json
// @Param client query string false "web | miniapp (default miniapp)"
// @Success 200 {object} service.TelegramLinkStartRes
// @Security BearerAuth
// @Router /mobile/user/phone/start [post]
func (h *Handler) PhoneLinkStart(c *gin.Context) {
	userId := claimData(c, "user_id")
	if userId == "" {
		return
	}
	res, err := h.Clients.User.PhoneLinkStart(c.Request.Context(), userId, c.Query("client") == "web")
	if err != nil {
		errs.Bad(c, err)
		return
	}
	c.JSON(http.StatusOK, res)
}

// PhoneLinkConfirm godoc
// @Summary Telegramda tasdiqlangan raqamni hisobga yozish (poll)
// @Description status=pending - kutish; confirmed - raqam ulandi. Raqam boshqa hisobda bo'lsa 409 identifier_taken.
// @Tags user
// @Accept json
// @Produce json
// @Param body body object true "{token}"
// @Success 200 {object} service.TelegramLinkCheckRes
// @Security BearerAuth
// @Router /mobile/user/phone/confirm [post]
func (h *Handler) PhoneLinkConfirm(c *gin.Context) {
	userId := claimData(c, "user_id")
	if userId == "" {
		return
	}
	var body struct {
		Token string `json:"token"`
	}
	if err := c.BindJSON(&body); err != nil || body.Token == "" {
		errs.Bad(c, errOr(err, "token is required"))
		return
	}
	res, err := h.Clients.User.PhoneLinkConfirm(c.Request.Context(), userId, body.Token)
	if err != nil {
		errs.Bad(c, err)
		return
	}
	c.JSON(http.StatusOK, res)
}

// UnlinkPhone godoc
// @Summary Telefon raqamni hisobdan uzish (kamida bitta boshqa kirish usuli qolishi shart)
// @Tags user
// @Produce json
// @Success 200 {object} string
// @Security BearerAuth
// @Router /mobile/user/phone [delete]
func (h *Handler) UnlinkPhone(c *gin.Context) {
	userId := claimData(c, "user_id")
	if userId == "" {
		return
	}
	if err := h.Clients.User.UnlinkPhone(c.Request.Context(), userId); err != nil {
		errs.Bad(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "phone unlinked"})
}

// EmailLinkSend godoc
// @Summary Hisobga email ulash: tasdiqlash kodi yuboriladi
// @Tags user
// @Accept json
// @Produce json
// @Param body body object true "{email}"
// @Success 200 {object} string
// @Security BearerAuth
// @Router /mobile/user/email/send [post]
func (h *Handler) EmailLinkSend(c *gin.Context) {
	userId := claimData(c, "user_id")
	if userId == "" {
		return
	}
	var body struct {
		Email string `json:"email"`
	}
	if err := c.BindJSON(&body); err != nil {
		errs.Bad(c, err)
		return
	}
	if _, err := h.Clients.User.EmailLinkSend(c.Request.Context(), userId, body.Email); err != nil {
		errs.Bad(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "code sent"})
}

// EmailLinkConfirm godoc
// @Summary Email ulashni kod bilan tasdiqlash
// @Tags user
// @Accept json
// @Produce json
// @Param body body object true "{email, code}"
// @Success 200 {object} string
// @Security BearerAuth
// @Router /mobile/user/email/confirm [post]
func (h *Handler) EmailLinkConfirm(c *gin.Context) {
	userId := claimData(c, "user_id")
	if userId == "" {
		return
	}
	var body struct {
		Email string `json:"email"`
		Code  string `json:"code"`
	}
	if err := c.BindJSON(&body); err != nil {
		errs.Bad(c, err)
		return
	}
	if _, err := h.Clients.User.EmailLinkConfirm(c.Request.Context(), userId, body.Email, body.Code); err != nil {
		errs.Bad(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "email linked"})
}

// UnlinkEmail godoc
// @Summary Emailni hisobdan uzish (kamida bitta boshqa kirish usuli qolishi shart)
// @Tags user
// @Produce json
// @Success 200 {object} string
// @Security BearerAuth
// @Router /mobile/user/email [delete]
func (h *Handler) UnlinkEmail(c *gin.Context) {
	userId := claimData(c, "user_id")
	if userId == "" {
		return
	}
	if err := h.Clients.User.UnlinkEmail(c.Request.Context(), userId); err != nil {
		errs.Bad(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "email unlinked"})
}

func errOr(err error, msg string) error {
	if err != nil {
		return err
	}
	return errString(msg)
}

type errString string

func (e errString) Error() string { return string(e) }
