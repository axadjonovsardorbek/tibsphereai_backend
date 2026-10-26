// Package ingest - manba (kitob/maqola/sayt) fayllaridan matn ajratish va chunklash
// (new-feature.md 7-bo'lim: ingestion pipeline).
package ingest

import (
	"archive/zip"
	"bytes"
	"encoding/xml"
	"errors"
	"fmt"
	"html"
	"io"
	"net/http"
	"path"
	"regexp"
	"sort"
	"strings"

	"github.com/ledongthuc/pdf"
)

// MaxFileBytes - yuklanadigan fayl hajmi chegarasi.
const MaxFileBytes = 50 << 20

// Section - manbaning joylashuvi (sahifa/bo'lim) bilan matn bo'lagi.
type Section struct {
	Locator string // "p. 12", "bo'lim 3" yoki bo'sh
	Text    string
}

var (
	reScriptStyle = regexp.MustCompile(`(?is)<(script|style|noscript|head|nav|footer|svg)[^>]*>.*?</(script|style|noscript|head|nav|footer|svg)>`)
	reBlockTag    = regexp.MustCompile(`(?i)</?(p|div|br|li|tr|h[1-6]|section|article|table|ul|ol)[^>]*>`)
	reTag         = regexp.MustCompile(`(?s)<[^>]+>`)
	reSpaces      = regexp.MustCompile(`[ \t\f\r]+`)
	reBlankLines  = regexp.MustCompile(`\n{3,}`)
)

// SupportedExt - ruxsat etilgan fayl kengaytmalari.
func SupportedExt(filename string) bool {
	switch strings.ToLower(path.Ext(filename)) {
	case ".pdf", ".epub", ".docx", ".txt", ".md":
		return true
	}
	return false
}

// ExtractFile - fayl turini kengaytmasidan aniqlab, matn bo'limlarini qaytaradi.
func ExtractFile(filename string, data []byte) ([]Section, error) {
	if len(data) == 0 {
		return nil, errors.New("fayl bo'sh")
	}
	if len(data) > MaxFileBytes {
		return nil, fmt.Errorf("fayl juda katta (max %d MB)", MaxFileBytes>>20)
	}
	if err := checkSignature(filename, data); err != nil {
		return nil, err
	}
	var (
		secs []Section
		err  error
	)
	switch strings.ToLower(path.Ext(filename)) {
	case ".pdf":
		secs, err = extractPDF(data)
	case ".docx":
		secs, err = extractDOCX(data)
	case ".epub":
		secs, err = extractEPUB(data)
	case ".txt", ".md":
		secs = []Section{{Text: string(data)}}
	default:
		return nil, fmt.Errorf("qo'llab-quvvatlanmaydigan format: %s (PDF, EPUB, DOCX, TXT)", path.Ext(filename))
	}
	if err != nil {
		return nil, err
	}
	return cleanSections(secs)
}

func cleanSections(in []Section) ([]Section, error) {
	var out []Section
	for _, s := range in {
		t := normalize(s.Text)
		if len([]rune(t)) < 20 {
			continue
		}
		out = append(out, Section{Locator: s.Locator, Text: t})
	}
	if len(out) == 0 {
		return nil, errors.New("fayldan matn ajratib bo'lmadi (skanerlangan rasm bo'lishi mumkin)")
	}
	return out, nil
}

func normalize(s string) string {
	s = strings.ReplaceAll(s, " ", " ")
	s = reSpaces.ReplaceAllString(s, " ")
	lines := strings.Split(s, "\n")
	for i := range lines {
		lines[i] = strings.TrimSpace(lines[i])
	}
	s = strings.Join(lines, "\n")
	s = reBlankLines.ReplaceAllString(s, "\n\n")
	return strings.TrimSpace(s)
}

func extractPDF(data []byte) (secs []Section, err error) {
	// ledongthuc/pdf buzilgan fayllarda panic qilishi mumkin.
	defer func() {
		if r := recover(); r != nil {
			secs, err = nil, fmt.Errorf("PDF'ni o'qib bo'lmadi: %v", r)
		}
	}()
	r, err := pdf.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		return nil, fmt.Errorf("PDF'ni o'qib bo'lmadi: %w", err)
	}
	for i := 1; i <= r.NumPage(); i++ {
		page := r.Page(i)
		if page.V.IsNull() {
			continue
		}
		txt, perr := page.GetPlainText(nil)
		if perr != nil {
			continue
		}
		secs = append(secs, Section{Locator: fmt.Sprintf("p. %d", i), Text: txt})
	}
	return secs, nil
}

func extractDOCX(data []byte) ([]Section, error) {
	zr, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		return nil, fmt.Errorf("DOCX'ni o'qib bo'lmadi: %w", err)
	}
	for _, f := range zr.File {
		if f.Name != "word/document.xml" {
			continue
		}
		rc, err := f.Open()
		if err != nil {
			return nil, err
		}
		defer rc.Close()
		return []Section{{Text: docxText(rc)}}, nil
	}
	return nil, errors.New("DOCX ichida word/document.xml topilmadi")
}

// docxText - <w:p> paragraflari ichidagi <w:t> matnlarini yig'adi.
func docxText(r io.Reader) string {
	dec := xml.NewDecoder(r)
	var b strings.Builder
	inT := false
	for {
		tok, err := dec.Token()
		if err != nil {
			break
		}
		switch t := tok.(type) {
		case xml.StartElement:
			if t.Name.Local == "t" {
				inT = true
			} else if t.Name.Local == "tab" {
				b.WriteString(" ")
			}
		case xml.EndElement:
			switch t.Name.Local {
			case "t":
				inT = false
			case "p":
				b.WriteString("\n")
			}
		case xml.CharData:
			if inT {
				b.Write(t)
			}
		}
	}
	return b.String()
}

func extractEPUB(data []byte) ([]Section, error) {
	zr, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		return nil, fmt.Errorf("EPUB'ni o'qib bo'lmadi: %w", err)
	}
	var files []*zip.File
	for _, f := range zr.File {
		switch strings.ToLower(path.Ext(f.Name)) {
		case ".xhtml", ".html", ".htm":
			files = append(files, f)
		}
	}
	sort.Slice(files, func(i, j int) bool { return files[i].Name < files[j].Name })
	var secs []Section
	for i, f := range files {
		rc, err := f.Open()
		if err != nil {
			continue
		}
		b, _ := io.ReadAll(io.LimitReader(rc, 8<<20))
		rc.Close()
		secs = append(secs, Section{Locator: fmt.Sprintf("bo'lim %d", i+1), Text: HTMLToText(string(b))})
	}
	return secs, nil
}

// HTMLToText - HTML'dan skript/stil/navigatsiyani olib tashlab, oddiy matn qaytaradi.
func HTMLToText(s string) string {
	s = reScriptStyle.ReplaceAllString(s, " ")
	s = reBlockTag.ReplaceAllString(s, "\n")
	s = reTag.ReplaceAllString(s, " ")
	return html.UnescapeString(s)
}

// FetchURL - veb-sahifani yuklab, toza matn qaytaradi (qayta indekslashda ham ishlatiladi).
func FetchURL(client *http.Client, u string) ([]Section, error) {
	if !strings.HasPrefix(u, "http://") && !strings.HasPrefix(u, "https://") {
		return nil, errors.New("URL http:// yoki https:// bilan boshlanishi kerak")
	}
	req, err := http.NewRequest(http.MethodGet, u, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", "Mozilla/5.0 (compatible; TibStationBot/1.0)")
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("sahifani yuklab bo'lmadi: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("sahifa status %d qaytardi", resp.StatusCode)
	}
	body, err := io.ReadAll(io.LimitReader(resp.Body, 5<<20))
	if err != nil {
		return nil, err
	}
	return cleanSections([]Section{{Text: HTMLToText(string(body))}})
}

// checkSignature - kengaytma bilan fayl haqiqiy tarkibi (magic bytes) mos kelishini tekshiradi:
// ".pdf" nomi ostida boshqa narsa (masalan ijro etiladigan fayl) yuklanishining oldini oladi.
func checkSignature(filename string, data []byte) error {
	switch strings.ToLower(path.Ext(filename)) {
	case ".pdf":
		if !bytes.HasPrefix(bytes.TrimLeft(data[:min(len(data), 1024)], "\x00\n\r\t "), []byte("%PDF-")) {
			return errors.New("fayl PDF emas (imzo mos kelmadi)")
		}
	case ".docx", ".epub":
		if !bytes.HasPrefix(data, []byte("PK\x03\x04")) {
			return fmt.Errorf("fayl %s emas (imzo mos kelmadi)", strings.ToUpper(strings.TrimPrefix(path.Ext(filename), ".")))
		}
	case ".txt", ".md":
		if bytes.IndexByte(data[:min(len(data), 8192)], 0) >= 0 {
			return errors.New("matn faylida binar belgilar bor")
		}
	}
	return nil
}
