package models

// RoleCreateReq - admin panel roli (masalan "Kontent menejer"). "superadmin" tizim
// roli - u migratsiyada seed qilinadi, alohida yaratilmaydi.
type RoleCreateReq struct {
	Name string `json:"name"`
	Slug string `json:"slug"`
}

type RoleRes struct {
	Id        *string `json:"id"`
	Name      *string `json:"name"`
	Slug      *string `json:"slug"`
	IsSystem  *bool   `json:"is_system"`
	CreatedAt *string `json:"created_at"`
	UpdatedAt *string `json:"updated_at"`
}

type RoleList struct {
	Roles []*RoleRes `json:"roles"`
	Count int        `json:"count"`
}

type RoleGetListReq struct {
	Name   string     `json:"name"`
	Filter Pagination `json:"filter"`
}

type RoleUpdateReq struct {
	Id   string `json:"id"`
	Name string `json:"name"`
	Slug string `json:"slug"`
}
