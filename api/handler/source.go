package handler

import (
	"context"
	"io"
	"net/http"
	"strings"
	"tibsphereai/internal/ingest"
	"tibsphereai/pkg/errs"
	m "tibsphereai/pkg/models"

	"github.com/gin-gonic/gin"
)

// splitIds - ["a,b", "c"] yoki ["a","b"] shaklidagi form qiymatlarini tekis ro'yxatga aylantiradi.
func splitIds(in []string) []string {
	var out []string
	for _, v := range in {
		for _, p := range strings.Split(v, ",") {
			if p = strings.TrimSpace(p); p != "" {
				out = append(out, p)
			}
		}
	}
	return out
}

// CreateSource godoc
// @Summary Manba yuklash (kitob/maqola: fayl, sayt: URL, matn) - AI practice bilimlar bazasi
// @Description multipart/form-data (fayl `file` maydonida; PDF, EPUB, DOCX, TXT, max 50MB) yoki JSON. Fon rejimida qayta ishlanadi: status uploaded→processing→ready|failed.
// @Tags source
// @Accept json,mpfd
// @Produce json
// @Param title formData string true "Nomi"
// @Param type formData string true "book|article|website|text"
// @Param language formData string false "uz|ru|en"
// @Param author formData string false "Muallif"
// @Param url formData string false "website uchun URL"
// @Param content formData string false "text/article uchun matn"
// @Param category_ids formData []string false "Bo'limlar (bo'sh=umumiy)"
// @Param topic_ids formData []string false "Mavzular"
// @Param file formData file false "Fayl"
// @Success 200 {object} models.ById
// @Security BearerAuth
// @Router /web/source [post]
func (h *Handler) CreateSource(c *gin.Context) {
	var req m.SourceCreateReq
	if strings.HasPrefix(c.ContentType(), "multipart/") {
		c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, ingest.MaxFileBytes+(1<<20))
		if err := c.Request.ParseMultipartForm(32 << 20); err != nil {
			errs.Bad(c, err)
			return
		}
		req.Title = c.PostForm("title")
		req.Type = c.PostForm("type")
		req.Language = c.PostForm("language")
		req.Author = c.PostForm("author")
		req.Url = c.PostForm("url")
		req.Content = c.PostForm("content")
		req.CategoryIds = splitIds(c.PostFormArray("category_ids"))
		req.TopicIds = splitIds(c.PostFormArray("topic_ids"))
		if fh, err := c.FormFile("file"); err == nil {
			f, err := fh.Open()
			if err != nil {
				errs.Bad(c, err)
				return
			}
			defer f.Close()
			data, err := io.ReadAll(io.LimitReader(f, ingest.MaxFileBytes+1))
			if err != nil {
				errs.Bad(c, err)
				return
			}
			req.FileName, req.FileData = fh.Filename, data
		}
	} else if err := c.BindJSON(&req); err != nil {
		errs.Bad(c, err)
		return
	}

	res, err := h.Clients.Source.Create(context.Background(), &req)
	if err != nil {
		errs.Bad(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "source uploaded, processing started", "id": res.Id})
}

// GetAllSources godoc
// @Tags source
// @Produce json
// @Param type query string false "book|article|website|text"
// @Param status query string false "uploaded|processing|ready|failed"
// @Param category_id query string false "Bo'lim"
// @Param topic_id query string false "Mavzu"
// @Param name query string false "Nomi bo'yicha qidiruv"
// @Param limit query int false "Limit"
// @Param page query int false "Page"
// @Success 200 {object} models.PaginationResponse
// @Security BearerAuth
// @Router /web/source [get]
func (h *Handler) GetAllSources(c *gin.Context) {
	pagination := m.ParsePagination(c.Query("limit"), c.Query("page"))
	data, _, err := h.Clients.Source.GetAll(c.Request.Context(), &m.SourceGetListReq{
		Type: c.Query("type"), Status: c.Query("status"), CategoryId: c.Query("category_id"),
		TopicId: c.Query("topic_id"), Name: c.Query("name"), Filter: pagination,
	})
	if err != nil {
		errs.Bad(c, err)
		return
	}
	c.JSON(http.StatusOK, data)
}

// GetByIdSource godoc
// @Tags source
// @Produce json
// @Param id path string true "Source ID"
// @Success 200 {object} models.SourceRes
// @Security BearerAuth
// @Router /web/source/{id} [get]
func (h *Handler) GetByIdSource(c *gin.Context) {
	res, err := h.Clients.Source.GetById(c.Request.Context(), &m.ById{Id: c.Param("id")})
	if err != nil {
		errs.Bad(c, err)
		return
	}
	c.JSON(http.StatusOK, res)
}

// UpdateSource godoc
// @Tags source
// @Accept json
// @Produce json
// @Param id path string true "Source ID"
// @Param body body models.SourceUpdateReq true "Source update"
// @Success 200 {object} string
// @Security BearerAuth
// @Router /web/source/{id}/update [put]
func (h *Handler) UpdateSource(c *gin.Context) {
	var req m.SourceUpdateReq
	if err := c.BindJSON(&req); err != nil {
		errs.Bad(c, err)
		return
	}
	req.Id = c.Param("id")
	if _, err := h.Clients.Source.Update(c.Request.Context(), &req); err != nil {
		errs.Bad(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "source updated"})
}

// ReindexSource godoc
// @Summary Manbani qayta indekslash (website qayta yuklanadi)
// @Tags source
// @Produce json
// @Param id path string true "Source ID"
// @Success 200 {object} string
// @Security BearerAuth
// @Router /web/source/{id}/reindex [post]
func (h *Handler) ReindexSource(c *gin.Context) {
	if _, err := h.Clients.Source.Reindex(c.Request.Context(), &m.ById{Id: c.Param("id")}); err != nil {
		errs.Bad(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "reindex started"})
}

// DeleteSource godoc
// @Tags source
// @Produce json
// @Param id path string true "Source ID"
// @Success 200 {object} string
// @Security BearerAuth
// @Router /web/source/{id}/delete [delete]
func (h *Handler) DeleteSource(c *gin.Context) {
	if _, err := h.Clients.Source.Delete(c.Request.Context(), &m.ById{Id: c.Param("id")}); err != nil {
		errs.Bad(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "source deleted"})
}

// GetPracticeStats godoc
// @Summary AI practice statistikasi (sessiyalar, bo'lim/mavzu bo'yicha, token, shikoyatlar)
// @Tags practice
// @Produce json
// @Success 200 {object} models.PracticeStats
// @Security BearerAuth
// @Router /web/practice/stats [get]
func (h *Handler) GetPracticeStats(c *gin.Context) {
	res, err := h.Clients.Practice.Stats(context.Background())
	if err != nil {
		errs.Bad(c, err)
		return
	}
	c.JSON(http.StatusOK, res)
}
