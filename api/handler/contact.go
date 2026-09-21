package handler

import (
	"context"
	"fmt"
	"net/http"
	"tibsphereai/pkg/errs"

	m "tibsphereai/pkg/models"

	"github.com/gin-gonic/gin"
	"golang.org/x/exp/slog"
)

// Create godoc
// @Summary Create a new contact
// @Description Create a new contact with the provided details
// @Tags contact
// @Accept  json
// @Produce  json
// @Param contact body models.ContactCreateReq true "Contact Details"
// @Success 200 {object} string
// @Failure 400 {object} string
// @Failure 500 {object} string
// @Security BearerAuth
// @Router /web/contact [post]
func (h *Handler) CreateContact(c *gin.Context) {
	req := m.ContactCreateReq{}
	err := c.BindJSON(&req)
	if err != nil {
		errs.Bad(c, err)
		slog.Error("Error binding request body: ", err)
		return
	}

	url_check := isValidURL(req.LinkUrl)
	if !url_check {
		errs.Bad(c, fmt.Errorf("invalid url"))
		slog.Error("Invalid url")
	}

	phone_check := isValidPhone(req.PhoneNumber)
	if !phone_check {
		errs.Bad(c, fmt.Errorf("invalid phone"))
		slog.Error("Invalid phone")
	}

	slog.Info("Create contact", "Contact", req)
	res, err := h.Clients.Contact.Create(context.Background(), &req)
	slog.Info("Create contact response", "Contact", res)
	if err != nil {
		errs.Bad(c, err)
		slog.Error("Error creating contact: ", err)
		return
	}

	slog.Info("Contact created successfully")
	c.JSON(200, gin.H{
		"message": "contact created successfully",
		"Id":      res.Id,
	})
}

// GetById godoc
// @Summary Get contact by ID
// @Description Get a contact by their ID
// @Tags contact
// @Accept  json
// @Produce  json
// @Param id path string false "Contact ID"
// @Success 200 {object} models.ContactRes
// @Failure 400 {object} string
// @Failure 500 {object} string
// @Security BearerAuth
// @Router /web/contact/{id} [get]
// @Router /mobile/contact/{id} [get]
func (h *Handler) GetByIdContact(c *gin.Context) {
	id := c.Param("id")

	slog.Info("Get contact with id", id)
	res, err := h.Clients.Contact.GetById(context.Background(), &m.ById{Id: id})
	slog.Info("Contact details", res)

	if err != nil {
		errs.Bad(c, err)
		slog.Error("Error getting contact by ID: ", err.Error())
		return
	}

	slog.Info("Contact retrieved successfully")
	c.JSON(200, res)
}

// GetAllContacts godoc
// @Summary Get all contact
// @Description Get all contact with optional filtering
// @Tags contact
// @Accept  json
// @Produce  json
// @Param name query string false "Name"
// @Param phone_number query string false "PhoneNumber"
// @Param limit query int false "Limit"
// @Param page query int false "Page"
// @Success 200 {object} models.ContactList
// @Failure 400 {object} string
// @Failure 500 {object} string
// @Security BearerAuth
// @Router /web/contact [get]
// @Router /mobile/contact [get]
func (h *Handler) GetAllContacts(c *gin.Context) {
	name := c.Query("name")
	phoneNumber := c.Query("phone_number")
	limit := c.Query("limit")
	page := c.Query("page")

	pagination := m.ParsePagination(limit, page)

	req := &m.ContactGetListReq{
		Name:        name,
		PhoneNumber: phoneNumber,
		Filter:      pagination,
	}

	slog.Info("Get all contact with filtes", req)
	data, _, err := h.Clients.Contact.GetAll(context.Background(), req)
	slog.Info("All contact", data)

	if err != nil {
		errs.Bad(c, err)
		slog.Error("Error getting contact: ", err.Error())
		return
	}

	slog.Info("Contacts retrieved successfully")
	c.JSON(200, data)
}

// Update godoc
// @Summary Update contact
// @Description Update contact
// @Tags contact
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path string false "Contact ID"
// @Param credentials body models.ContactUpdateBody true "Contact update credentials"
// @Success 200 {object} string "Contact updated"
// @Failure 400 {object} string "Invalid request payload"
// @Failure 500 {object} string "Server error"
// @Router /web/contact/{id}/update [put]
func (h *Handler) UpdateContact(c *gin.Context) {
	var req m.ContactUpdateReq
	var body m.ContactUpdateBody

	id := c.Param("id")

	if err := c.BindJSON(&body); err != nil {
		errs.Bad(c, err)
		slog.Error("Error binding request body", err.Error())
		return
	}

	req.Id = id
	req.LinkUrl = body.LinkUrl
	req.Name = body.Name
	req.PhoneNumber = body.PhoneNumber

	url_check := isValidURL(req.LinkUrl)
	if !url_check {
		errs.Bad(c, fmt.Errorf("invalid url"))
		slog.Error("Invalid url")
	}

	phone_check := isValidPhone(req.PhoneNumber)
	if !phone_check {
		errs.Bad(c, fmt.Errorf("invalid phone"))
		slog.Error("Invalid phone")
	}

	slog.Info("Update Contact", req)
	_, err := h.Clients.Contact.Update(context.Background(), &req)

	if err != nil {
		errs.Bad(c, err)
		slog.Error("Error updating Contact: ", err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "contact updated"})
}

// Delete godoc
// @Summary Delete contact
// @Description Delete contact
// @Tags contact
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path string true "Id"
// @Success 200 {object} string "Contact deleted"
// @Failure 400 {object} string "Invalid request payload"
// @Failure 500 {object} string "Server error"
// @Router /web/contact/{id}/delete [delete]
func (h *Handler) DeleteContact(c *gin.Context) {
	var req m.ById

	id := c.Param("id")

	req.Id = id

	slog.Info("Contact delete with id", req)
	_, err := h.Clients.Contact.Delete(context.Background(), &req)

	if err != nil {
		errs.Bad(c, err)
		slog.Error("Error deleting contact", err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "contact deleted successfully"})
}
