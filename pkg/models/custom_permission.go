package models

// CustomPermissionCreateReq - admin panel moduli (U-Code custom-permission patterni,
// API_DOCUMENTATION_EN.md). ParentId bo'sh bo'lsa - root modul, bo'lsa - shu modulning
// bo'lim-moduli (bir daraja chuqurlik yetarli, U-Code'dagi kabi).
type CustomPermissionCreateReq struct {
	Title      string         `json:"title"`
	ParentId   string         `json:"parent_id"`
	Attributes map[string]any `json:"attributes"`
	OrderNum   int            `json:"order_num"`
}

type CustomPermissionRes struct {
	Id         *string        `json:"id"`
	Title      *string        `json:"title"`
	ParentId   *string        `json:"parent_id"`
	Attributes map[string]any `json:"attributes"`
	OrderNum   *int           `json:"order_num"`
	CreatedAt  *string        `json:"created_at"`
	UpdatedAt  *string        `json:"updated_at"`
}

type CustomPermissionList struct {
	Modules []*CustomPermissionRes `json:"modules"`
	Count   int                    `json:"count"`
}

type CustomPermissionUpdateReq struct {
	Id         string         `json:"id"`
	Title      string         `json:"title"`
	Attributes map[string]any `json:"attributes"`
	OrderNum   int            `json:"order_num"`
}

// --- Accesses (rol x modul huquqlari) ---

type RoleAccessGetReq struct {
	RoleId   string `json:"role_id"`
	ParentId string `json:"parent_id"` // bo'sh bo'lsa - root modullar
}

// RoleAccessItem - bitta modul + shu rol uchun hozirgi ruxsatlar (hali belgilanmagan
// bo'lsa hammasi false qaytadi, alohida "yo'q" holati emas - U-Code'dagi "No" ekvivalenti).
type RoleAccessItem struct {
	CustomPermissionId string         `json:"custom_permission_id"`
	Title              string         `json:"title"`
	Attributes         map[string]any `json:"attributes"`
	HasChildren        bool           `json:"has_children"`
	CanRead            bool           `json:"read"`
	CanWrite           bool           `json:"write"`
	CanUpdate          bool           `json:"update"`
	CanDelete          bool           `json:"delete"`
}

type RoleAccessList struct {
	Items []RoleAccessItem `json:"items"`
}

// RoleAccessUpsertReq - bitta rol+modul juftligi uchun ruxsatlarni belgilash (mavjud
// bo'lmasa yaratadi, bo'lsa yangilaydi).
type RoleAccessUpsertReq struct {
	RoleId             string `json:"role_id"`
	CustomPermissionId string `json:"custom_permission_id"`
	CanRead            bool   `json:"read"`
	CanWrite           bool   `json:"write"`
	CanUpdate          bool   `json:"update"`
	CanDelete          bool   `json:"delete"`
}
