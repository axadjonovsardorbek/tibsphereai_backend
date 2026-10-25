package ingest

import (
	"archive/zip"
	"bytes"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestChunkSections_ShortStaysOne(t *testing.T) {
	cs := ChunkSections([]Section{{Locator: "p. 1", Text: "Qisqa matn."}})
	require.Len(t, cs, 1)
	assert.Equal(t, "p. 1", cs[0].Locator)
}

func TestChunkSections_LongSplitsWithOverlap(t *testing.T) {
	para := strings.Repeat("Bu klinik gap bo'lib, bemor holatini tasvirlaydi. ", 10) // ~500 belgi
	text := strings.TrimSpace(strings.Repeat(para+"\n\n", 20))                       // ~10 000 belgi
	cs := ChunkSections([]Section{{Text: text}})
	require.Greater(t, len(cs), 2)
	for _, c := range cs {
		assert.LessOrEqual(t, len([]rune(c.Content)), ChunkChars+10)
		assert.Greater(t, c.Tokens, 0)
	}
	// ustma-ustlik: keyingi chunkning boshi oldingisining oxirida uchraydi
	tail := []rune(cs[0].Content)
	probe := string(tail[len(tail)-100:])
	assert.Contains(t, cs[0].Content+cs[1].Content, probe)
}

func TestExtractFile_TXT(t *testing.T) {
	secs, err := ExtractFile("a.txt", []byte("Bu yetarlicha uzun matn bo'lib, tekshiruv uchun yozilgan."))
	require.NoError(t, err)
	require.Len(t, secs, 1)
}

func TestExtractFile_Empty_And_BadExt(t *testing.T) {
	_, err := ExtractFile("a.txt", nil)
	assert.Error(t, err)
	_, err = ExtractFile("a.exe", []byte("xxxxxxxxxxxxxxxxxxxxxxxxxxxxxx"))
	assert.Error(t, err)
}

func TestExtractFile_DOCX(t *testing.T) {
	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)
	w, _ := zw.Create("word/document.xml")
	w.Write([]byte(`<?xml version="1.0"?><w:document xmlns:w="x"><w:body><w:p><w:r><w:t>Birinchi paragraf matni bu yerda.</w:t></w:r></w:p><w:p><w:r><w:t>Ikkinchi paragraf ham yetarlicha uzun.</w:t></w:r></w:p></w:body></w:document>`))
	zw.Close()
	secs, err := ExtractFile("a.docx", buf.Bytes())
	require.NoError(t, err)
	assert.Contains(t, secs[0].Text, "Birinchi paragraf")
	assert.Contains(t, secs[0].Text, "Ikkinchi paragraf")
}

func TestExtractFile_EPUB(t *testing.T) {
	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)
	w, _ := zw.Create("OEBPS/ch1.xhtml")
	w.Write([]byte(`<html><head><title>x</title></head><body><h1>Sarlavha</h1><p>Epub bobining yetarlicha uzun matni shu yerda.</p><script>bad()</script></body></html>`))
	zw.Close()
	secs, err := ExtractFile("a.epub", buf.Bytes())
	require.NoError(t, err)
	assert.Contains(t, secs[0].Text, "Epub bobining")
	assert.NotContains(t, secs[0].Text, "bad()")
	assert.Equal(t, "bo'lim 1", secs[0].Locator)
}

func TestHTMLToText(t *testing.T) {
	s := HTMLToText(`<nav>menu</nav><script>x()</script><p>Salom &amp; xush kelibsiz</p><style>a{}</style>`)
	assert.Contains(t, s, "Salom & xush kelibsiz")
	assert.NotContains(t, s, "menu")
	assert.NotContains(t, s, "x()")
}
