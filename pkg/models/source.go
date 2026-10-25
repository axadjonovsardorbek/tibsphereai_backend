package models

// Source - AI practice (real-time case/test) uchun bilimlar bazasi manbasi.
// Eski CaseSource (name+content) bilan bir xil jadvalda (case_sources) saqlanadi.
// Type: book | article | website | text. Status: uploaded -> processing -> ready | failed.

const (
	SourceTypeBook    = "book"
	SourceTypeArticle = "article"
	SourceTypeWebsite = "website"
	SourceTypeText    = "text"

	SourceStatusUploaded   = "uploaded"
	SourceStatusProcessing = "processing"
	SourceStatusReady      = "ready"
	SourceStatusFailed     = "failed"
)

// SourceCreateReq - JSON yoki multipart (maydon nomlari bir xil; fayl "file" maydonida).
type SourceCreateReq struct {
	Title       string   `json:"title" form:"title"`
	Type        string   `json:"type" form:"type"`
	Language    string   `json:"language" form:"language"`
	Author      string   `json:"author" form:"author"`
	Url         string   `json:"url" form:"url"`
	Content     string   `json:"content" form:"content"` // type=text yoki article matn
	CategoryIds []string `json:"category_ids" form:"category_ids"`
	TopicIds    []string `json:"topic_ids" form:"topic_ids"`
	// Service qatlami to'ldiradi (fayl yuklangan bo'lsa).
	FileUrl  string `json:"-"`
	FileName string `json:"-"`
	FileData []byte `json:"-"`
}

type SourceUpdateReq struct {
	Id          string   `json:"id"`
	Title       string   `json:"title"`
	Language    string   `json:"language"`
	Author      string   `json:"author"`
	Url         string   `json:"url"`
	IsActive    *bool    `json:"is_active"`
	CategoryIds []string `json:"category_ids"` // nil = o'zgartirilmaydi, [] = bog'lanishlar o'chiriladi
	TopicIds    []string `json:"topic_ids"`
}

type SourceRes struct {
	Id          *string  `json:"id"`
	Title       *string  `json:"title"`
	Type        *string  `json:"type"`
	Language    *string  `json:"language"`
	Author      *string  `json:"author"`
	Url         *string  `json:"url"`
	FileUrl     *string  `json:"file_url"`
	Status      *string  `json:"status"`
	Error       *string  `json:"error"`
	IsActive    *bool    `json:"is_active"`
	ChunkCount  *int     `json:"chunk_count"`
	CategoryIds []string `json:"category_ids"`
	TopicIds    []string `json:"topic_ids"`
	CreatedAt   *string  `json:"created_at"`
	UpdatedAt   *string  `json:"updated_at"`
}

type SourceList struct {
	Sources []*SourceRes `json:"sources"`
	Count   int          `json:"count"`
}

type SourceGetListReq struct {
	Type       string     `json:"type"`
	Status     string     `json:"status"`
	CategoryId string     `json:"category_id"`
	TopicId    string     `json:"topic_id"`
	Name       string     `json:"name"`
	Filter     Pagination `json:"filter"`
}

// SourceChunk - qidiruv (RAG) uchun matn bo'lagi.
type SourceChunk struct {
	Id         string
	SourceId   string
	Idx        int
	Locator    string
	Content    string
	Tokens     int
	Embedding  []float32
	SourceName string
	SourceType string
	Author     string
	Url        string
}

// SourceRef - foydalanuvchiga "Foydalanilgan manbalar" blokida ko'rsatiladigan manba.
// Backend RAG'da promptga HAQIQATAN kiritilgan chunklardan shakllantiradi (AI o'ylab topmaydi).
type SourceRef struct {
	SourceId string `json:"source_id"`
	Type     string `json:"type"`
	Title    string `json:"title"`
	Author   string `json:"author,omitempty"`
	Url      string `json:"url,omitempty"` // faqat website uchun
	Locator  string `json:"locator,omitempty"`
}
