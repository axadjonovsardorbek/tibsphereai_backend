package handler

import (
	"net/http"
	"strconv"
	"tibsphereai/pkg/errs"
	m "tibsphereai/pkg/models"

	"github.com/gin-gonic/gin"
)

// GetDueFlashcards godoc
// @Summary Bugungi takrorlashga tayyor fleshkartalar (BEPUL)
// @Tags flashcard
// @Produce json
// @Param limit query int false "Limit (default 20, max 100)"
// @Param deck_id query string false "Faqat shu to'plamdan"
// @Success 200 {object} models.FlashcardDueRes
// @Security BearerAuth
// @Router /mobile/flashcard/due [get]
func (h *Handler) GetDueFlashcards(c *gin.Context) {
	userId := claimData(c, "user_id")
	if userId == "" {
		return
	}
	limit, _ := strconv.Atoi(c.Query("limit"))
	res, err := h.Clients.Flashcard.Due(c.Request.Context(), userId, c.Query("deck_id"), limit)
	if err != nil {
		errs.Bad(c, err)
		return
	}
	c.JSON(http.StatusOK, res)
}

// ReviewFlashcard godoc
// @Summary Kartani baholash: again | hard | good | easy (BEPUL, streak'ni yangilaydi)
// @Tags flashcard
// @Accept json
// @Produce json
// @Param id path string true "Card ID"
// @Param body body models.FlashcardReviewReq true "rating"
// @Success 200 {object} models.FlashcardReviewRes
// @Security BearerAuth
// @Router /mobile/flashcard/cards/{id}/review [post]
func (h *Handler) ReviewFlashcard(c *gin.Context) {
	userId := claimData(c, "user_id")
	if userId == "" {
		return
	}
	var req m.FlashcardReviewReq
	if err := c.BindJSON(&req); err != nil {
		errs.Bad(c, err)
		return
	}
	req.UserId, req.CardId = userId, c.Param("id")
	res, err := h.Clients.Flashcard.Review(c.Request.Context(), &req)
	if err != nil {
		errs.Bad(c, err)
		return
	}
	c.JSON(http.StatusOK, res)
}

// GetFlashcardStats godoc
// @Summary Fleshkarta statistikasi (bugungi takrorlashlar, yangi, o'zlashtirilgan, streak)
// @Tags flashcard
// @Produce json
// @Success 200 {object} models.FlashcardStatsRes
// @Security BearerAuth
// @Router /mobile/flashcard/stats [get]
func (h *Handler) GetFlashcardStats(c *gin.Context) {
	userId := claimData(c, "user_id")
	if userId == "" {
		return
	}
	res, err := h.Clients.Flashcard.Stats(c.Request.Context(), userId)
	if err != nil {
		errs.Bad(c, err)
		return
	}
	c.JSON(http.StatusOK, res)
}

// ListFlashcardDecks godoc
// @Summary Fleshkarta to'plamlari ("Xatolarim" va mavzuli to'plamlar)
// @Tags flashcard
// @Produce json
// @Success 200 {array} models.FlashcardDeckRes
// @Security BearerAuth
// @Router /mobile/flashcard/decks [get]
func (h *Handler) ListFlashcardDecks(c *gin.Context) {
	userId := claimData(c, "user_id")
	if userId == "" {
		return
	}
	res, err := h.Clients.Flashcard.ListDecks(c.Request.Context(), userId)
	if err != nil {
		errs.Bad(c, err)
		return
	}
	c.JSON(http.StatusOK, res)
}

// CreateFlashcardDeck godoc
// @Summary Mavzu bo'yicha AI fleshkarta to'plami buyurtma qilish (AI token sarfi hisobidan tanga yechiladi)
// @Description category_id (+ixtiyoriy topic_id) yoki erkin mavzu topic_text; count 5..30 (default 10).
// @Tags flashcard
// @Accept json
// @Produce json
// @Param Accept-Language header string false "Language (uz|ru|en)" default(uz)
// @Param body body models.FlashcardDeckCreateReq true "deck"
// @Success 200 {object} service.DeckCreateRes
// @Security BearerAuth
// @Router /mobile/flashcard/decks [post]
func (h *Handler) CreateFlashcardDeck(c *gin.Context) {
	userId := claimData(c, "user_id")
	if userId == "" {
		return
	}
	var req m.FlashcardDeckCreateReq
	if err := c.BindJSON(&req); err != nil {
		errs.Bad(c, err)
		return
	}
	req.UserId, req.Language = userId, getLanguage(c)
	res, err := h.Clients.Flashcard.CreateDeck(c.Request.Context(), &req)
	if err != nil {
		errs.Bad(c, err)
		return
	}
	c.JSON(http.StatusOK, res)
}

// DeleteFlashcardDeck godoc
// @Summary To'plamni (va undagi kartalarni) o'chirish
// @Tags flashcard
// @Produce json
// @Param id path string true "Deck ID"
// @Success 200 {object} string
// @Security BearerAuth
// @Router /mobile/flashcard/decks/{id} [delete]
func (h *Handler) DeleteFlashcardDeck(c *gin.Context) {
	userId := claimData(c, "user_id")
	if userId == "" {
		return
	}
	if err := h.Clients.Flashcard.DeleteDeck(c.Request.Context(), userId, c.Param("id")); err != nil {
		errs.Bad(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "deck deleted"})
}

// ListFlashcards godoc
// @Summary Kartalar ro'yxati
// @Tags flashcard
// @Produce json
// @Param deck_id query string false "To'plam"
// @Param limit query int false "Limit"
// @Param page query int false "Page"
// @Success 200 {object} models.PaginationResponse
// @Security BearerAuth
// @Router /mobile/flashcard/cards [get]
func (h *Handler) ListFlashcards(c *gin.Context) {
	userId := claimData(c, "user_id")
	if userId == "" {
		return
	}
	p := m.ParsePagination(c.Query("limit"), c.Query("page"))
	cards, total, err := h.Clients.Flashcard.ListCards(c.Request.Context(), userId, c.Query("deck_id"), p.Limit, p.Page)
	if err != nil {
		errs.Bad(c, err)
		return
	}
	if p.Limit <= 0 {
		p.Limit = 20
	}
	c.JSON(http.StatusOK, m.NewPaginationResponse(p, total, cards))
}

// CreateFlashcard godoc
// @Summary Qo'lda karta qo'shish (bepul)
// @Tags flashcard
// @Accept json
// @Produce json
// @Param body body models.FlashcardCreateReq true "card"
// @Success 200 {object} models.FlashcardRes
// @Security BearerAuth
// @Router /mobile/flashcard/cards [post]
func (h *Handler) CreateFlashcard(c *gin.Context) {
	userId := claimData(c, "user_id")
	if userId == "" {
		return
	}
	var req m.FlashcardCreateReq
	if err := c.BindJSON(&req); err != nil {
		errs.Bad(c, err)
		return
	}
	req.UserId = userId
	res, err := h.Clients.Flashcard.CreateCard(c.Request.Context(), &req)
	if err != nil {
		errs.Bad(c, err)
		return
	}
	c.JSON(http.StatusOK, res)
}

// UpdateFlashcard godoc
// @Summary Kartani tahrirlash yoki to'xtatish (status: active|suspended)
// @Tags flashcard
// @Accept json
// @Produce json
// @Param id path string true "Card ID"
// @Param body body models.FlashcardUpdateReq true "front/back/status"
// @Success 200 {object} string
// @Security BearerAuth
// @Router /mobile/flashcard/cards/{id} [put]
func (h *Handler) UpdateFlashcard(c *gin.Context) {
	userId := claimData(c, "user_id")
	if userId == "" {
		return
	}
	var req m.FlashcardUpdateReq
	if err := c.BindJSON(&req); err != nil {
		errs.Bad(c, err)
		return
	}
	req.UserId, req.Id = userId, c.Param("id")
	if err := h.Clients.Flashcard.UpdateCard(c.Request.Context(), &req); err != nil {
		errs.Bad(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "card updated"})
}

// DeleteFlashcard godoc
// @Summary Kartani o'chirish
// @Tags flashcard
// @Produce json
// @Param id path string true "Card ID"
// @Success 200 {object} string
// @Security BearerAuth
// @Router /mobile/flashcard/cards/{id} [delete]
func (h *Handler) DeleteFlashcard(c *gin.Context) {
	userId := claimData(c, "user_id")
	if userId == "" {
		return
	}
	if err := h.Clients.Flashcard.DeleteCard(c.Request.Context(), userId, c.Param("id")); err != nil {
		errs.Bad(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "card deleted"})
}

// GetFlashcardAdminStats godoc
// @Summary Fleshkartalar monitoringi (kartalar, takrorlashlar, eslab qolish va o'zlashtirish foizi, eng ko'p xato mavzular)
// @Tags flashcard
// @Produce json
// @Success 200 {object} models.FlashcardAdminStats
// @Security BearerAuth
// @Router /web/flashcard/stats [get]
func (h *Handler) GetFlashcardAdminStats(c *gin.Context) {
	res, err := h.Clients.Flashcard.AdminStats(c.Request.Context())
	if err != nil {
		errs.Bad(c, err)
		return
	}
	c.JSON(http.StatusOK, res)
}
