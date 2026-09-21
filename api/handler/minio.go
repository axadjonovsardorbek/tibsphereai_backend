package handler

import (
	"fmt"
	"mime/multipart"
	"net/http"
	"tibsphereai/pkg/errs"

	"github.com/gin-gonic/gin"
	"golang.org/x/exp/slog"
)

type File struct {
	File *multipart.FileHeader `form:"file" binding:"required"`
}

// File upload
// @Security    BearerAuth
// @Summary File upload
// @Description File upload
// @Tags file-upload
// @Accept multipart/form-data
// @Produce json
// @Param file formData file true "File"
// @Security BearerAuth
// @Router /web/file-upload [post]
// @Success 200 {object} string
func (h *Handler) UploadFile(c *gin.Context) {

	fileName, tempFilePath, err := formFile(c, "file")
	if err != nil {
		errs.Bad(c, err)
		slog.Error("Error processing image: ", err)
		return
	} else if fileName == "" && tempFilePath == "" {
		errs.Bad(c, fmt.Errorf("file not provided"))
		slog.Error("File not provided")
		return
	}

	// MinIO ga yuklash (avvalgi MinIO.Upload funksiyangizdan foydalanamiz)
	minioURL, err := h.MinIO.Upload(fileName, tempFilePath)
	if err != nil {
		errs.Bad(c, fmt.Errorf("failed to upload to MinIO"))
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "successfully upload",
		"Url":     minioURL,
	})
}
