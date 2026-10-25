package service

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"tibsphereai/internal/ai"
	"tibsphereai/internal/ingest"
	m "tibsphereai/pkg/models"
	st "tibsphereai/storage/postgres"
	"time"
)

// maxStoredContent - manbaning ajratib olingan matni bazada saqlanadigan maksimal belgi soni.
const maxStoredContent = 2_000_000

// SourceService - AI practice bilimlar bazasi (new-feature.md 4.1.2, 7-bo'lim):
// manba yuklash -> matn ajratish -> chunklash -> embedding -> status=ready/failed (fon rejimida).
type SourceService struct {
	storage st.Storage
	ai      *ai.Client
	http    *http.Client
	// async - testlarda fon ishini sinxron bajarish uchun o'chiriladi.
	async bool
}

func NewSourceService(storage *st.Storage, aiClient *ai.Client) *SourceService {
	return &SourceService{storage: *storage, ai: aiClient, http: &http.Client{Timeout: 30 * time.Second}, async: true}
}

func validSourceType(t string) bool {
	switch t {
	case m.SourceTypeBook, m.SourceTypeArticle, m.SourceTypeWebsite, m.SourceTypeText:
		return true
	}
	return false
}

func (s *SourceService) Create(ctx context.Context, req *m.SourceCreateReq) (*m.ById, error) {
	req.Title = strings.TrimSpace(req.Title)
	if req.Title == "" {
		return nil, errors.New("title is required")
	}
	if !validSourceType(req.Type) {
		return nil, fmt.Errorf("invalid type %q (book|article|website|text)", req.Type)
	}
	hasFile := len(req.FileData) > 0
	switch req.Type {
	case m.SourceTypeWebsite:
		if !strings.HasPrefix(req.Url, "http://") && !strings.HasPrefix(req.Url, "https://") {
			return nil, errors.New("url (http/https) is required for website")
		}
	case m.SourceTypeText:
		if strings.TrimSpace(req.Content) == "" {
			return nil, errors.New("content is required for text")
		}
	case m.SourceTypeBook:
		if !hasFile {
			return nil, errors.New("file (PDF/EPUB/DOCX/TXT) is required for book")
		}
	case m.SourceTypeArticle:
		if !hasFile && strings.TrimSpace(req.Content) == "" {
			return nil, errors.New("file or content is required for article")
		}
	}
	if hasFile && !ingest.SupportedExt(req.FileName) {
		return nil, fmt.Errorf("unsupported file format: %s (PDF, EPUB, DOCX, TXT)", req.FileName)
	}

	res, err := s.storage.SourceS.Create(req, "", m.SourceStatusProcessing)
	if err != nil {
		return nil, err
	}
	job := func() { s.ingestNew(res.Id, req) }
	if s.async {
		go job()
	} else {
		job()
	}
	return res, nil
}

// ingestNew - yangi manbani qayta ishlaydi (fon rejimida).
func (s *SourceService) ingestNew(id string, req *m.SourceCreateReq) {
	var (
		secs []ingest.Section
		err  error
	)
	switch {
	case len(req.FileData) > 0:
		secs, err = ingest.ExtractFile(req.FileName, req.FileData)
	case req.Type == m.SourceTypeWebsite:
		secs, err = ingest.FetchURL(s.http, req.Url)
	default:
		secs = []ingest.Section{{Text: req.Content}}
	}
	if err != nil {
		_ = s.storage.SourceS.SetStatus(id, m.SourceStatusFailed, err.Error())
		return
	}
	s.finishIngest(id, secs)
}

// finishIngest - matn bo'limlari -> content, chunk, embedding, status.
func (s *SourceService) finishIngest(id string, secs []ingest.Section) {
	var full strings.Builder
	for _, sec := range secs {
		full.WriteString(sec.Text)
		full.WriteString("\n\n")
	}
	content := truncateRunes(strings.TrimSpace(full.String()), maxStoredContent)
	if err := s.storage.SourceS.SetContent(id, content); err != nil {
		_ = s.storage.SourceS.SetStatus(id, m.SourceStatusFailed, err.Error())
		return
	}

	chunks := ingest.ChunkSections(secs)
	if len(chunks) == 0 {
		_ = s.storage.SourceS.SetStatus(id, m.SourceStatusFailed, "matn topilmadi")
		return
	}
	rows := make([]m.SourceChunk, len(chunks))
	texts := make([]string, len(chunks))
	for i, c := range chunks {
		rows[i] = m.SourceChunk{Locator: c.Locator, Content: c.Content, Tokens: c.Tokens}
		texts[i] = c.Content
	}

	// Embedding xato bersa chunklar baribir saqlanadi (qidiruv tasodifiy tanlashga tushadi);
	// sabab status yonida "error" maydonida ko'rinadi.
	note := ""
	if s.ai != nil && s.ai.Embedder != nil {
		embs, _, eErr := s.ai.Embedder.Embed(context.Background(), texts, false)
		if eErr != nil {
			note = "embedding o'tkazib yuborildi: " + eErr.Error()
		} else {
			for i := range rows {
				rows[i].Embedding = st.Normalize(embs[i])
			}
		}
	}
	if err := s.storage.SourceS.ReplaceChunks(id, rows); err != nil {
		_ = s.storage.SourceS.SetStatus(id, m.SourceStatusFailed, err.Error())
		return
	}
	_ = s.storage.SourceS.SetStatus(id, m.SourceStatusReady, note)
}

func truncateRunes(s string, n int) string {
	r := []rune(s)
	if len(r) <= n {
		return s
	}
	return string(r[:n])
}

func (s *SourceService) GetById(ctx context.Context, req *m.ById) (*m.SourceRes, error) {
	return s.storage.SourceS.GetById(req)
}

func (s *SourceService) GetAll(ctx context.Context, req *m.SourceGetListReq) (*m.PaginationResponse, *m.SourceList, error) {
	return s.storage.SourceS.GetAll(req)
}

func (s *SourceService) Update(ctx context.Context, req *m.SourceUpdateReq) (*m.Void, error) {
	return s.storage.SourceS.Update(req)
}

func (s *SourceService) Delete(ctx context.Context, req *m.ById) (*m.Void, error) {
	return s.storage.SourceS.Delete(req)
}

// Reindex - website bo'lsa sahifa qayta yuklanadi; boshqalarda saqlangan matn qayta chunklanadi.
func (s *SourceService) Reindex(ctx context.Context, req *m.ById) (*m.Void, error) {
	typ, url, _, content, err := s.storage.SourceS.GetRaw(req.Id)
	if err != nil {
		return nil, err
	}
	if typ != m.SourceTypeWebsite && strings.TrimSpace(content) == "" {
		return nil, errors.New("source has no stored text to reindex")
	}
	if err := s.storage.SourceS.SetStatus(req.Id, m.SourceStatusProcessing, ""); err != nil {
		return nil, err
	}
	job := func() {
		var secs []ingest.Section
		if typ == m.SourceTypeWebsite {
			var fErr error
			secs, fErr = ingest.FetchURL(s.http, url)
			if fErr != nil {
				_ = s.storage.SourceS.SetStatus(req.Id, m.SourceStatusFailed, fErr.Error())
				return
			}
		} else {
			secs = []ingest.Section{{Text: content}}
		}
		s.finishIngest(req.Id, secs)
	}
	if s.async {
		go job()
	} else {
		job()
	}
	return &m.Void{}, nil
}
