package services

import (
	"archive/zip"
	"bytes"
	"fmt"
	"io"
	"path/filepath"
	"strings"
)

// ZipEntry representa un archivo extraído del ZIP.
type ZipEntry struct {
	Filename string
	Data     []byte
	Kind     string // "pdf", "image" o "" (no soportado)
	MIME     string
}

// extractZipEntries descomprime zipData y retorna una entrada por cada archivo.
// Archivos no soportados se retornan con Kind="" para que el caller los reporte.
func extractZipEntries(zipData []byte) ([]ZipEntry, error) {
	reader, err := zip.NewReader(bytes.NewReader(zipData), int64(len(zipData)))
	if err != nil {
		return nil, fmt.Errorf("error al leer el ZIP: %w", err)
	}

	var entries []ZipEntry
	for _, f := range reader.File {
		if f.FileInfo().IsDir() {
			continue
		}
		name := filepath.Base(f.Name)
		if strings.HasPrefix(name, ".") {
			continue
		}

		data, err := readZipFile(f)
		if err != nil {
			return nil, fmt.Errorf("error al leer %s del ZIP: %w", f.Name, err)
		}

		ext := strings.ToLower(filepath.Ext(name))
		kind, mime := classifyExtension(ext)

		entries = append(entries, ZipEntry{
			Filename: name,
			Data:     data,
			Kind:     kind,
			MIME:     mime,
		})
	}

	return entries, nil
}

func readZipFile(f *zip.File) ([]byte, error) {
	rc, err := f.Open()
	if err != nil {
		return nil, err
	}
	defer rc.Close()
	return io.ReadAll(rc)
}

func classifyExtension(ext string) (kind, mime string) {
	switch ext {
	case ".pdf":
		return "pdf", "application/pdf"
	case ".png":
		return "image", "image/png"
	case ".jpg", ".jpeg":
		return "image", "image/jpeg"
	case ".gif":
		return "image", "image/gif"
	case ".webp":
		return "image", "image/webp"
	case ".bmp":
		return "image", "image/bmp"
	case ".tif", ".tiff":
		return "image", "image/tiff"
	case ".heic", ".heif":
		return "image", "image/heic"
	default:
		return "", ""
	}
}
