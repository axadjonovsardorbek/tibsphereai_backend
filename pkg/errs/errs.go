// Package errs - butun backend uchun bitta xato-javob strukturasi.
// Har xato: HTTP kod + slug (mashina o'qiydigan, frontend shu bo'yicha branch qiladi)
// + 3 tilli matn (uz/ru/en). Handlerlar to'g'ridan-to'g'ri *AppError yoki oddiy error
// qaytarishi mumkin - Respond ikkalasini ham to'g'ri formatlaydi.
package errs

type Lang string

const (
	LangUZ Lang = "uz"
	LangRU Lang = "ru"
	LangEN Lang = "en"
)

// Text - bitta xabarning uch tildagi tarjimasi.
type Text struct {
	Uz string `json:"uz"`
	Ru string `json:"ru"`
	En string `json:"en"`
}

// Pick - so'ralgan tilni qaytaradi, topilmasa uz'ga, undan keyin en'ga tushadi.
func (t Text) Pick(lang Lang) string {
	switch lang {
	case LangRU:
		if t.Ru != "" {
			return t.Ru
		}
	case LangEN:
		if t.En != "" {
			return t.En
		}
	}
	if t.Uz != "" {
		return t.Uz
	}
	return t.En
}

// AppError - butun loyihadagi yagona xato strukturasi.
type AppError struct {
	HTTPCode int    `json:"-"`
	Slug     string `json:"-"`
	Text     Text   `json:"-"`
	err      error  // original (wrap qilingan) xato - debug uchun details'ga chiqadi
}

func (e *AppError) Error() string {
	if e.err != nil {
		return e.Slug + ": " + e.err.Error()
	}
	return e.Slug
}

func (e *AppError) Unwrap() error { return e.err }

// New - katalog uchun yangi statik AppError e'lon qiladi.
func New(httpCode int, slug string, text Text) *AppError {
	return &AppError{HTTPCode: httpCode, Slug: slug, Text: text}
}

// Wrap - katalogdagi AppError'ga original (debug) xatoni biriktirib, nusxasini qaytaradi
// (statik katalog qiymatini mutatsiya qilmaslik uchun).
func (e *AppError) Wrap(err error) *AppError {
	cp := *e
	cp.err = err
	return &cp
}

// WithHTTPCode - kerak bo'lsa, xuddi shu slug/matn bilan boshqa HTTP kod ishlatish uchun.
func (e *AppError) WithHTTPCode(code int) *AppError {
	cp := *e
	cp.HTTPCode = code
	return &cp
}
