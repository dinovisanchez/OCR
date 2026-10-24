package services

import (
	"archive/zip"
	"bytes"
	"testing"

	"github.com/stretchr/testify/assert"
)

type zipFile struct {
	name string
	data []byte
}

// createTestZip crea un ZIP con archivos en el orden dado (determinístico).
func createTestZip(files map[string][]byte) []byte {
	buf := new(bytes.Buffer)
	w := zip.NewWriter(buf)
	for name, data := range files {
		f, _ := w.Create(name)
		f.Write(data)
	}
	w.Close()
	return buf.Bytes()
}

// createOrderedZip crea un ZIP con orden determinístico.
func createOrderedZip(files []zipFile) []byte {
	buf := new(bytes.Buffer)
	w := zip.NewWriter(buf)
	for _, zf := range files {
		f, _ := w.Create(zf.name)
		f.Write(zf.data)
	}
	w.Close()
	return buf.Bytes()
}

func TestExtractZipEntries_MixedFiles(t *testing.T) {
	zipData := createTestZip(map[string][]byte{
		"factura.pdf":  []byte("fake-pdf"),
		"foto.jpg":     []byte("fake-jpg"),
		"imagen.png":   []byte("fake-png"),
		"readme.txt":   []byte("some text"),
		"data.csv":     []byte("col1,col2"),
		".hidden.pdf":  []byte("hidden"),
	})

	entries, err := extractZipEntries(zipData)
	assert.NoError(t, err)
	// 5 archivos (no incluye .hidden.pdf)
	assert.Len(t, entries, 5)

	byName := map[string]ZipEntry{}
	for _, e := range entries {
		byName[e.Filename] = e
	}

	assert.Equal(t, "pdf", byName["factura.pdf"].Kind)
	assert.Equal(t, "application/pdf", byName["factura.pdf"].MIME)

	assert.Equal(t, "image", byName["foto.jpg"].Kind)
	assert.Equal(t, "image/jpeg", byName["foto.jpg"].MIME)

	assert.Equal(t, "image", byName["imagen.png"].Kind)
	assert.Equal(t, "image/png", byName["imagen.png"].MIME)

	// Archivos no soportados: Kind vacío
	assert.Equal(t, "", byName["readme.txt"].Kind)
	assert.Equal(t, "", byName["data.csv"].Kind)
}

func TestExtractZipEntries_EmptyZip(t *testing.T) {
	zipData := createTestZip(map[string][]byte{})

	entries, err := extractZipEntries(zipData)
	assert.NoError(t, err)
	assert.Empty(t, entries)
}

func TestExtractZipEntries_InvalidZip(t *testing.T) {
	_, err := extractZipEntries([]byte("not a zip file"))
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "error al leer el ZIP")
}

func TestExtractZipEntries_SkipsDirectories(t *testing.T) {
	buf := new(bytes.Buffer)
	w := zip.NewWriter(buf)
	// Directorio
	_, _ = w.Create("subdir/")
	// Archivo dentro del directorio
	f, _ := w.Create("subdir/factura.pdf")
	f.Write([]byte("pdf-data"))
	w.Close()

	entries, err := extractZipEntries(buf.Bytes())
	assert.NoError(t, err)
	assert.Len(t, entries, 1)
	assert.Equal(t, "factura.pdf", entries[0].Filename)
	assert.Equal(t, "pdf", entries[0].Kind)
}

func TestExtractZipEntries_AllImageFormats(t *testing.T) {
	zipData := createTestZip(map[string][]byte{
		"a.gif":  []byte("gif"),
		"b.webp": []byte("webp"),
		"c.bmp":  []byte("bmp"),
		"d.tiff": []byte("tiff"),
		"e.heic": []byte("heic"),
	})

	entries, err := extractZipEntries(zipData)
	assert.NoError(t, err)
	assert.Len(t, entries, 5)

	for _, e := range entries {
		assert.Equal(t, "image", e.Kind)
		assert.NotEmpty(t, e.MIME)
	}
}

func TestClassifyExtension(t *testing.T) {
	tests := []struct {
		ext  string
		kind string
		mime string
	}{
		{".pdf", "pdf", "application/pdf"},
		{".png", "image", "image/png"},
		{".jpg", "image", "image/jpeg"},
		{".jpeg", "image", "image/jpeg"},
		{".gif", "image", "image/gif"},
		{".webp", "image", "image/webp"},
		{".bmp", "image", "image/bmp"},
		{".tif", "image", "image/tiff"},
		{".tiff", "image", "image/tiff"},
		{".heic", "image", "image/heic"},
		{".heif", "image", "image/heic"},
		{".txt", "", ""},
		{".docx", "", ""},
	}
	for _, tt := range tests {
		kind, mime := classifyExtension(tt.ext)
		assert.Equal(t, tt.kind, kind, "ext: %s", tt.ext)
		assert.Equal(t, tt.mime, mime, "ext: %s", tt.ext)
	}
}
