package services

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"ia-extraction/models"
	"ia-extraction/repositories"
	"io"
	"net/http"
	"net/url"
	"path/filepath"
	"strings"

	"gorm.io/gorm"
)

type ExtractionService interface {
	ExtractAndSave(link string, documentType string, mediaType string) (json.RawMessage, *models.TokenUsage, error)
	ExtractZipAndSave(link string, documentType string) ([]models.ZipEntryResult, *models.TokenUsage, error)
}

// ErrInvalidDocumentType se retorna cuando Gemini determina que el documento
// no corresponde al document_type esperado (is_valid_document: false).
type ErrInvalidDocumentType struct {
	DocumentType string
}

func (e *ErrInvalidDocumentType) Error() string {
	return fmt.Sprintf("el documento no cumple con el formato esperado para el tipo: %s", e.DocumentType)
}

// ErrPromptNotConfigured se retorna cuando el document_type no tiene un prompt
// registrado en la BD. Sin configuración no se puede ejecutar la extracción.
type ErrPromptNotConfigured struct {
	DocumentType string
}

// ErrProtectedDocument se retorna cuando el PDF está protegido con contraseña
// y Gemini no puede leer su contenido.
type ErrProtectedDocument struct{}

func (e *ErrProtectedDocument) Error() string {
	return "el documento no pudo ser leído porque está protegido con contraseña"
}

// isEncryptedPDF detecta PDFs con contraseña buscando la entrada /Encrypt,
// que siempre está presente en el trailer de un PDF cifrado.
func isEncryptedPDF(data []byte) bool {
	return bytes.Contains(data, []byte("/Encrypt"))
}

func (e *ErrPromptNotConfigured) Error() string {
	return fmt.Sprintf("no hay prompt configurado para el tipo de documento: %s", e.DocumentType)
}

type FetchFunc func(rawURL string) ([]byte, string, error)

type extractionService struct {
	repo       repositories.ExtractionRepository
	promptRepo repositories.DocumentTypePromptRepository
	gemini     GeminiClient
	fetchFn    FetchFunc
}

func NewExtractionService(repo repositories.ExtractionRepository, promptRepo repositories.DocumentTypePromptRepository, gemini GeminiClient) ExtractionService {
	return &extractionService{repo: repo, promptRepo: promptRepo, gemini: gemini, fetchFn: fetchResource}
}

func fetchResource(rawURL string) ([]byte, string, error) {
	resp, err := http.Get(rawURL)
	if err != nil {
		return nil, "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, "", fmt.Errorf("descarga HTTP %d", resp.StatusCode)
	}
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, "", err
	}
	return body, resp.Header.Get("Content-Type"), nil
}

func normalizeContentType(ct string) string {
	ct = strings.TrimSpace(strings.Split(ct, ";")[0])
	return strings.ToLower(ct)
}

func extensionFromURL(link string) string {
	u, err := url.Parse(link)
	if err != nil {
		return ""
	}
	return strings.ToLower(filepath.Ext(u.Path))
}

// resolveInputKind devuelve "image", "pdf" o "zip". mediaType opcional: "image", "pdf", "zip" o vacío (inferir).
func resolveInputKind(contentType, link, mediaType string) (string, error) {
	mt := strings.ToLower(strings.TrimSpace(mediaType))
	switch mt {
	case "image":
		return "image", nil
	case "pdf":
		return "pdf", nil
	case "zip":
		return "zip", nil
	}
	nct := normalizeContentType(contentType)
	if strings.HasPrefix(nct, "image/") {
		return "image", nil
	}
	if nct == "application/pdf" {
		return "pdf", nil
	}
	if nct == "application/zip" || nct == "application/x-zip-compressed" {
		return "zip", nil
	}
	switch extensionFromURL(link) {
	case ".png", ".jpg", ".jpeg", ".gif", ".webp", ".bmp", ".tif", ".tiff", ".heic", ".heif":
		return "image", nil
	case ".pdf":
		return "pdf", nil
	case ".zip":
		return "zip", nil
	}
	return "", fmt.Errorf("no se pudo inferir el tipo de medio; indica media_type \"pdf\", \"image\" o \"zip\"")
}

func imageMIME(contentType, link string) string {
	nct := normalizeContentType(contentType)
	if strings.HasPrefix(nct, "image/") {
		return nct
	}
	switch extensionFromURL(link) {
	case ".png":
		return "image/png"
	case ".jpg", ".jpeg":
		return "image/jpeg"
	case ".gif":
		return "image/gif"
	case ".webp":
		return "image/webp"
	case ".bmp":
		return "image/bmp"
	case ".tif", ".tiff":
		return "image/tiff"
	case ".heic", ".heif":
		return "image/heic"
	default:
		return "image/jpeg"
	}
}

func buildPrompt(prompt *models.DocumentTypePrompt) string {
	fieldDescriptions := make([]string, len(prompt.Fields))
	for i, f := range prompt.Fields {
		fieldDescriptions[i] = fmt.Sprintf("%s (%s)", f.Name, f.Type)
	}
	return fmt.Sprintf(
		"%s\n\nTipo de documento esperado: %s\nCampos a extraer: %s\n\nDevuelve un objeto JSON con exactamente esos campos como llaves, respetando el tipo indicado entre paréntesis.",
		prompt.PromptComplement,
		prompt.DocumentType,
		strings.Join(fieldDescriptions, ", "),
	)
}

// extractDocumentIdentification busca en la respuesta JSON de Gemini el valor
// asociado a la llave `fieldName` (tal como está configurada en el prompt) y
// lo devuelve como string. Devuelve nil si:
//   - `fieldName` está vacío (el prompt no configuró identification_number),
//   - la respuesta no es JSON válido,
//   - la llave no existe,
//   - el valor es null o string vacío.
// Los valores numéricos se convierten a su representación en string.
func extractDocumentIdentification(llmResp, fieldName string) *string {
	if fieldName == "" {
		return nil
	}
	var parsed map[string]json.RawMessage
	if err := json.Unmarshal([]byte(llmResp), &parsed); err != nil {
		return nil
	}
	raw, ok := parsed[fieldName]
	if !ok {
		return nil
	}

	// Intento 1: string directo.
	var str string
	if err := json.Unmarshal(raw, &str); err == nil {
		trimmed := strings.TrimSpace(str)
		if trimmed == "" {
			return nil
		}
		return &trimmed
	}

	// Intento 2: número → su representación textual (sin delegar a fmt para enteros).
	trimmedRaw := strings.TrimSpace(string(raw))
	if trimmedRaw == "" || trimmedRaw == "null" {
		return nil
	}
	return &trimmedRaw
}

// processSingleDocument ejecuta validación + extracción en una sola llamada a
// Gemini (el schema incluye `is_valid_document`) y persiste el resultado.
// Es la lógica core reutilizada por ExtractAndSave y ExtractZipAndSave.
// El prompt ya debe estar resuelto por el caller.
func (s *extractionService) processSingleDocument(
	image models.GeminiInlineImage, sourceLabel, link string,
	dtPrompt *models.DocumentTypePrompt,
) (json.RawMessage, *models.TokenUsage, error) {

	prompt := buildPrompt(dtPrompt)
	requestLog := prompt + "\n\n[extracción desde " + sourceLabel + ": " + link + "]"
	llmResp, totalUsage, err := s.gemini.Call(dtPrompt.DocumentType, prompt, []models.GeminiInlineImage{image}, dtPrompt.Fields)
	if err != nil {
		return nil, totalUsage, err
	}

	e := &models.Extraction{
		DocumentType:           dtPrompt.DocumentType,
		DocumentIdentification: extractDocumentIdentification(llmResp, dtPrompt.IdentificationNumber),
		Request:                requestLog,
		Response:               llmResp,
	}
	if errsave := s.repo.Save(e); errsave != nil {
		return nil, totalUsage, errsave
	}

	raw := json.RawMessage(llmResp)
	if !json.Valid(raw) {
		fallback, _ := json.Marshal(map[string]string{"raw": llmResp})
		return json.RawMessage(fallback), totalUsage, nil
	}
	return raw, totalUsage, nil
}

func (s *extractionService) ExtractAndSave(link string, documentType string, mediaType string) (json.RawMessage, *models.TokenUsage, error) {
	dtPrompt, err := s.promptRepo.FindByDocumentType(documentType)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil, &ErrPromptNotConfigured{DocumentType: documentType}
		}
		return nil, nil, fmt.Errorf("error consultando prompt para %q: %w", documentType, err)
	}

	body, ct, err := s.fetchFn(link)
	if err != nil {
		return nil, nil, err
	}

	kind, err := resolveInputKind(ct, link, mediaType)
	if err != nil {
		return nil, nil, err
	}

	var image models.GeminiInlineImage
	var sourceLabel string
	switch kind {
	case "image":
		image = models.GeminiInlineImage{MIME: imageMIME(ct, link), Data: body}
		sourceLabel = "imagen"
	case "pdf":
		if isEncryptedPDF(body) {
			return nil, nil, &ErrProtectedDocument{}
		}
		image = models.GeminiInlineImage{MIME: "application/pdf", Data: body}
		sourceLabel = "PDF"
	default:
		return nil, nil, fmt.Errorf("tipo de medio no soportado")
	}

	return s.processSingleDocument(image, sourceLabel, link, dtPrompt)
}

func (s *extractionService) ExtractZipAndSave(link string, documentType string) ([]models.ZipEntryResult, *models.TokenUsage, error) {
	// Lookup del prompt una sola vez para todo el ZIP
	dtPrompt, err := s.promptRepo.FindByDocumentType(documentType)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil, &ErrPromptNotConfigured{DocumentType: documentType}
		}
		return nil, nil, fmt.Errorf("error consultando prompt para %q: %w", documentType, err)
	}

	body, _, err := s.fetchFn(link)
	if err != nil {
		return nil, nil, err
	}

	entries, err := extractZipEntries(body)
	if err != nil {
		return nil, nil, err
	}
	if len(entries) == 0 {
		return nil, nil, fmt.Errorf("el ZIP no contiene archivos")
	}

	var results []models.ZipEntryResult
	var allUsages []*models.TokenUsage

	for _, entry := range entries {
		if entry.Kind == "" {
			ext := filepath.Ext(entry.Filename)
			results = append(results, models.ZipEntryResult{
				Filename: entry.Filename,
				Status:   "error",
				Error:    fmt.Sprintf("tipo de archivo no soportado: %s", ext),
			})
			continue
		}

		if entry.Kind == "pdf" && isEncryptedPDF(entry.Data) {
			results = append(results, models.ZipEntryResult{
				Filename: entry.Filename,
				Status:   "protected_document",
				Error:    (&ErrProtectedDocument{}).Error(),
			})
			continue
		}

		var sourceLabel string
		if entry.Kind == "pdf" {
			sourceLabel = "PDF"
		} else {
			sourceLabel = "imagen"
		}

		image := models.GeminiInlineImage{MIME: entry.MIME, Data: entry.Data}
		data, usage, err := s.processSingleDocument(image, sourceLabel, entry.Filename, dtPrompt)
		allUsages = append(allUsages, usage)

		if err != nil {
			var invalidDoc *ErrInvalidDocumentType
			if errors.As(err, &invalidDoc) {
				results = append(results, models.ZipEntryResult{
					Filename: entry.Filename,
					Status:   "invalid_document",
					Error:    err.Error(),
				})
			} else {
				results = append(results, models.ZipEntryResult{
					Filename: entry.Filename,
					Status:   "error",
					Error:    err.Error(),
				})
			}
			continue
		}

		results = append(results, models.ZipEntryResult{
			Filename: entry.Filename,
			Status:   "success",
			Data:     data,
		})
	}

	return results, sumUsage(allUsages...), nil
}

// sumUsage agrega los token usages de varias llamadas a Gemini para reportar
// el total al cliente.
func sumUsage(usages ...*models.TokenUsage) *models.TokenUsage {
	total := &models.TokenUsage{}
	any := false
	for _, u := range usages {
		if u == nil {
			continue
		}
		any = true
		total.PromptTokens += u.PromptTokens
		total.CandidatesTokens += u.CandidatesTokens
		total.TotalTokens += u.TotalTokens
		total.ThinkingTokens += u.ThinkingTokens
	}
	if !any {
		return nil
	}
	return total
}
