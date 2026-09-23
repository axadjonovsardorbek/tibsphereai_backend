package models

type AdminRes struct {
	Id          string  `json:"id"`
	Login       string  `json:"login"`
	RoleId      *string `json:"role_id"`
	RoleName    *string `json:"role_name"`
	PartnerId   *string `json:"partner_id"`
	PartnerName *string `json:"partner_name"`
	CreatedAt   string  `json:"created_at"`
	UpdatedAt   string  `json:"updated_at"`
}

type AdminList struct {
	Admins []*AdminRes `json:"admins"`
	Count  int         `json:"count"`
}

type AdminGetListReq struct {
	Login  string     `json:"login"`
	Filter Pagination `json:"filter"`
}

// AdminCreateReq - yangi admin/xodim qo'shish. RoleId bo'sh bo'lsa admin rolsiz yaratiladi
// (keyin RoleAccess/Update orqali biriktiriladi). PartnerId - faqat "Hamkor Admin"
// (partner_admin) roli uchun to'ldiriladi, shu hamkorning promokodlarigagina ruxsati bo'ladi.
type AdminCreateReq struct {
	Login     string `json:"login"`
	Password  string `json:"password"`
	RoleId    string `json:"role_id"`
	PartnerId string `json:"partner_id"`
}

// AdminUpdateReq - Password ixtiyoriy: bo'sh bo'lsa parol o'zgartirilmaydi (o'z parolini
// o'zi almashtirish uchun alohida /web/auth/password/change ishlatiladi).
type AdminUpdateReq struct {
	Id        string `json:"id"`
	Login     string `json:"login"`
	RoleId    string `json:"role_id"`
	Password  string `json:"password"`
	PartnerId string `json:"partner_id"`
}
