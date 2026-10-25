package ingest

import (
	"strings"
	"unicode/utf8"
)

const (
	// ChunkChars - bitta chunk uzunligi (~600 token, ~4 belgi/token).
	ChunkChars = 2400
	// ChunkOverlap - qo'shni chunklar o'rtasidagi takror (~100 token).
	ChunkOverlap = 400
)

// Chunk - qidiruv birligi.
type Chunk struct {
	Locator string
	Content string
	Tokens  int
}

// EstimateTokens - taxminiy token soni (~4 belgi).
func EstimateTokens(s string) int {
	return (utf8.RuneCountInString(s) + 3) / 4
}

// ChunkSections - har bir bo'limni paragraf/gap chegarasida ~ChunkChars uzunlikdagi
// chunklarga bo'ladi, qo'shnilari ChunkOverlap belgi bilan ustma-ust tushadi.
func ChunkSections(secs []Section) []Chunk {
	var out []Chunk
	for _, s := range secs {
		for _, c := range splitText(s.Text, ChunkChars, ChunkOverlap) {
			out = append(out, Chunk{Locator: s.Locator, Content: c, Tokens: EstimateTokens(c)})
		}
	}
	return out
}

func splitText(text string, size, overlap int) []string {
	r := []rune(strings.TrimSpace(text))
	if len(r) == 0 {
		return nil
	}
	if len(r) <= size {
		return []string{string(r)}
	}
	var out []string
	start := 0
	for start < len(r) {
		end := start + size
		if end >= len(r) {
			out = append(out, strings.TrimSpace(string(r[start:])))
			break
		}
		// chegarani paragraf, bo'lmasa gap oxiriga suramiz (oynaning oxirgi 30%i ichida).
		cut := breakPoint(r, start+size*7/10, end)
		out = append(out, strings.TrimSpace(string(r[start:cut])))
		next := cut - overlap
		if next <= start {
			next = cut
		}
		start = next
	}
	return out
}

// breakPoint - [lo, hi] oralig'ida eng yaxshi bo'linish joyi: "\n\n" > ". " > bo'sh joy > hi.
func breakPoint(r []rune, lo, hi int) int {
	for i := hi; i > lo; i-- {
		if r[i-1] == '\n' && i-2 >= 0 && r[i-2] == '\n' {
			return i
		}
	}
	for i := hi; i > lo; i-- {
		if (r[i-1] == '.' || r[i-1] == '!' || r[i-1] == '?') && i < len(r) && (r[i] == ' ' || r[i] == '\n') {
			return i
		}
	}
	for i := hi; i > lo; i-- {
		if r[i-1] == ' ' || r[i-1] == '\n' {
			return i
		}
	}
	return hi
}
