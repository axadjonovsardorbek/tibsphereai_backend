package models

import (
	"strconv"
)

// Pagination represents pagination parameters
type Pagination struct {
	Page  int `form:"page" json:"page"`
	Limit int `form:"limit" json:"limit"`
}

// GetOffset returns the offset for database queries
func (p *Pagination) GetOffset() int {
	if p.Page <= 0 {
		p.Page = 1
	}
	return (p.Page - 1) * p.GetLimit()
}

// GetLimit returns the limit for database queries
func (p *Pagination) GetLimit() int {
	if p.Limit <= 0 {
		p.Limit = 10
	}
	if p.Limit > 100 {
		p.Limit = 100
	}
	return p.Limit
}

// PaginationResponse represents a paginated response
type PaginationResponse struct {
	Total       int         `json:"total"`
	Page        int         `json:"page"`
	Limit       int         `json:"limit"`
	TotalPages  int         `json:"total_pages"`
	HasPrevious bool        `json:"has_previous"`
	HasNext     bool        `json:"has_next"`
	Data        interface{} `json:"data"`
}

// NewPaginationResponse creates a new pagination response
func NewPaginationResponse(pagination Pagination, total int, data interface{}) PaginationResponse {
	if total == 0 {
		data = []interface{}{}
		return PaginationResponse{
			Data: data,
		}
	}

	totalPages := total / pagination.Limit
	if total%pagination.Limit != 0 {
		totalPages++
	}
	page := (pagination.Page / pagination.Limit) + 1

	return PaginationResponse{
		Total:       total,
		Page:        page,
		Limit:       pagination.Limit,
		TotalPages:  totalPages,
		HasPrevious: page > 1,
		HasNext:     page < totalPages,
		Data:        data,
	}
}

func ParsePagination(limit, offset string) Pagination {
	limitValue := 10
	offsetValue := 0

	if limit != "" {
		parsedLimit, err := strconv.Atoi(limit)
		if err != nil {
			parsedLimit = 10
		}
		limitValue = parsedLimit
	}

	if offset != "" {
		parsedOffset, err := strconv.Atoi(offset)
		if err != nil {
			parsedOffset = 1
		}
		offsetValue = (parsedOffset - 1) * limitValue
	}

	return Pagination{
		Page:  offsetValue,
		Limit: limitValue,
	}
}
