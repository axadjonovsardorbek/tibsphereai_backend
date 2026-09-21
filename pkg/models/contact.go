package models

type ContactCreateReq struct {
	Name        string `json:"name"`
	PhoneNumber string `json:"phone_number"`
	LinkUrl     string `json:"link_url"`
}

type ContactRes struct {
	Id          *string `json:"id"`
	Name        *string `json:"name"`
	PhoneNumber *string `json:"phone_number"`
	LinkUrl     *string `json:"link_url"`
	CreatedAt   *string `json:"created_at"`
	UpdatedAt   *string `json:"updated_at"`
}

type ContactList struct {
	Contacts []*ContactRes `json:"contacts"`
	Count    int           `json:"count"`
}

type ContactGetListReq struct {
	Name        string     `json:"name"`
	PhoneNumber string     `json:"phone_number"`
	Filter      Pagination `json:"filter"`
}

type ContactUpdateBody struct {
	Name        string `json:"name"`
	PhoneNumber string `json:"phone_number"`
	LinkUrl     string `json:"link_url"`
}

type ContactUpdateReq struct {
	Id          string `json:"id"`
	Name        string `json:"name"`
	PhoneNumber string `json:"phone_number"`
	LinkUrl     string `json:"link_url"`
}
