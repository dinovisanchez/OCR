package services

import (
	"encoding/json"
	"errors"
	"testing"

	mocks_repositories "ia-extraction/mocks/repositories"
	mocks_services "ia-extraction/mocks/services"
	"ia-extraction/models"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"gorm.io/gorm"
)

// --- Helper function tests ---

func TestNormalizeContentType_WithCharset(t *testing.T) {
	assert.Equal(t, "image/png", normalizeContentType("image/png; charset=utf-8"))
}

func TestNormalizeContentType_Plain(t *testing.T) {
	assert.Equal(t, "application/pdf", normalizeContentType("application/pdf"))
}

func TestNormalizeContentType_WithSpaces(t *testing.T) {
	assert.Equal(t, "image/jpeg", normalizeContentType("  image/jpeg ; boundary=something"))
}

func TestExtensionFromURL_PNG(t *testing.T) {
	assert.Equal(t, ".png", extensionFromURL("https://example.com/file.PNG"))
}

func TestExtensionFromURL_NoExtension(t *testing.T) {
	assert.Equal(t, "", extensionFromURL("https://example.com/file"))
}

func TestExtensionFromURL_PDF(t *testing.T) {
	assert.Equal(t, ".pdf", extensionFromURL("https://example.com/doc.pdf"))
}

func TestExtensionFromURL_InvalidURL(t *testing.T) {
	assert.Equal(t, "", extensionFromURL("://invalid"))
}

func TestResolveInputKind_ExplicitImage(t *testing.T) {
	kind, err := resolveInputKind("", "", "image")
	assert.NoError(t, err)
	assert.Equal(t, "image", kind)
}

func TestResolveInputKind_ExplicitPDF(t *testing.T) {
	kind, err := resolveInputKind("", "", "pdf")
	assert.NoError(t, err)
	assert.Equal(t, "pdf", kind)
}

func TestResolveInputKind_ContentTypeImage(t *testing.T) {
	kind, err := resolveInputKind("image/png", "", "")
	assert.NoError(t, err)
	assert.Equal(t, "image", kind)
}

func TestResolveInputKind_ContentTypePDF(t *testing.T) {
	kind, err := resolveInputKind("application/pdf", "", "")
	assert.NoError(t, err)
	assert.Equal(t, "pdf", kind)
}

func TestResolveInputKind_ExtensionJPG(t *testing.T) {
	kind, err := resolveInputKind("application/octet-stream", "https://example.com/photo.jpg", "")
	assert.NoError(t, err)
	assert.Equal(t, "image", kind)
}

func TestResolveInputKind_ExtensionPDF(t *testing.T) {
	kind, err := resolveInputKind("application/octet-stream", "https://example.com/doc.pdf", "")
	assert.NoError(t, err)
	assert.Equal(t, "pdf", kind)
}

func TestResolveInputKind_CannotInfer(t *testing.T) {
	_, err := resolveInputKind("application/octet-stream", "https://example.com/file", "")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "no se pudo inferir")
}

func TestImageMIME_FromContentType(t *testing.T) {
	assert.Equal(t, "image/png", imageMIME("image/png", ""))
}

func TestImageMIME_FromExtensionJPG(t *testing.T) {
	assert.Equal(t, "image/jpeg", imageMIME("application/octet-stream", "https://example.com/photo.jpg"))
}

func TestImageMIME_FromExtensionJPEG(t *testing.T) {
	assert.Equal(t, "image/jpeg", imageMIME("application/octet-stream", "https://example.com/photo.jpeg"))
}

func TestImageMIME_FromExtensionGIF(t *testing.T) {
	assert.Equal(t, "image/gif", imageMIME("application/octet-stream", "https://example.com/anim.gif"))
}

func TestImageMIME_FromExtensionWebP(t *testing.T) {
	assert.Equal(t, "image/webp", imageMIME("application/octet-stream", "https://example.com/photo.webp"))
}

func TestImageMIME_FromExtensionBMP(t *testing.T) {
	assert.Equal(t, "image/bmp", imageMIME("application/octet-stream", "https://example.com/photo.bmp"))
}

func TestImageMIME_FromExtensionTIFF(t *testing.T) {
	assert.Equal(t, "image/tiff", imageMIME("application/octet-stream", "https://example.com/photo.tiff"))
}

func TestImageMIME_FromExtensionHEIC(t *testing.T) {
	assert.Equal(t, "image/heic", imageMIME("application/octet-stream", "https://example.com/photo.heic"))
}

func TestImageMIME_DefaultJPEG(t *testing.T) {
	assert.Equal(t, "image/jpeg", imageMIME("application/octet-stream", "https://example.com/file"))
}

func TestBuildPrompt(t *testing.T) {
	p := &models.DocumentTypePrompt{
		DocumentType:     "factura",
		PromptComplement: "Reglas específicas para facturas",
		Fields: models.FieldSpecs{
			{Name: "total", Type: models.FieldTypeNumber},
			{Name: "fecha", Type: models.FieldTypeString},
		},
	}
	result := buildPrompt(p)
	assert.Contains(t, result, "factura")
	assert.Contains(t, result, "total (number)")
	assert.Contains(t, result, "fecha (string)")
	assert.Contains(t, result, "Reglas específicas para facturas")
}

func TestBuildResponseSchema(t *testing.T) {
	fields := []models.FieldSpec{
		{Name: "total", Type: models.FieldTypeNumber},
		{Name: "items", Type: models.FieldTypeArray},
		{Name: "is_paid", Type: models.FieldTypeBoolean},
		{Name: "no_type"}, // sin tipo → fallback a string
	}
	schema := buildResponseSchema(fields)

	assert.Equal(t, "object", schema.Type)
	assert.Equal(t, "boolean", schema.Properties["is_valid_document"].Type)
	assert.False(t, schema.Properties["is_valid_document"].Nullable)
	assert.Equal(t, "number", schema.Properties["total"].Type)
	assert.True(t, schema.Properties["total"].Nullable)
	assert.Equal(t, "array", schema.Properties["items"].Type)
	assert.NotNil(t, schema.Properties["items"].Items, "array fields must include items schema for Gemini")
	assert.Equal(t, "string", schema.Properties["items"].Items.Type)
	assert.Equal(t, "boolean", schema.Properties["is_paid"].Type)
	assert.Equal(t, "string", schema.Properties["no_type"].Type)
	assert.ElementsMatch(t, []string{"is_valid_document", "total", "items", "is_paid", "no_type"}, schema.Required)
}

// --- ExtractAndSave tests ---

// newMockPromptRepo retorna un mock que por defecto responde con un prompt
// válido para cualquier document_type. Los tests individuales pueden sobreescribir.
func newMockPromptRepo(t *testing.T, documentType string) *mocks_repositories.MockDocumentTypePromptRepository {
	m := mocks_repositories.NewMockDocumentTypePromptRepository(t)
	defaultPrompt := &models.DocumentTypePrompt{
		ID:               1,
		DocumentType:     documentType,
		PromptComplement: "Extrae los campos solicitados",
		Fields: models.FieldSpecs{
			{Name: "total", Type: models.FieldTypeString},
			{Name: "fecha", Type: models.FieldTypeString},
			{Name: "nombre", Type: models.FieldTypeString},
			{Name: "numero", Type: models.FieldTypeString},
		},
		IsActive: true,
	}
	m.On("FindByDocumentType", mock.AnythingOfType("string")).Return(defaultPrompt, nil).Maybe()
	return m
}

func TestExtractAndSave_ImageSuccess(t *testing.T) {
	mockRepo := mocks_repositories.NewMockExtractionRepository(t)
	mockGemini := mocks_services.NewMockGeminiClient(t)

	svc := &extractionService{
		repo:       mockRepo,
		promptRepo: newMockPromptRepo(t, "factura"),
		gemini:     mockGemini,
		fetchFn: func(rawURL string) ([]byte, string, error) {
			return []byte("fake-image-data"), "image/png", nil
		},
	}

	expectedJSON := `{"total":"100"}`
	extractionUsage := &models.TokenUsage{TotalTokens: 150}

	mockGemini.On("Call", "factura", mock.AnythingOfType("string"), mock.AnythingOfType("[]models.GeminiInlineImage"), mock.AnythingOfType("[]models.FieldSpec")).
		Return(expectedJSON, extractionUsage, nil).Once()

	mockRepo.On("Save", mock.AnythingOfType("*models.Extraction")).
		Run(func(args mock.Arguments) {
			e := args.Get(0).(*models.Extraction)
			e.ID = 1
			e.CreatedAt = "2026-01-01T00:00:00Z"
		}).
		Return(nil).Once()

	result, usage, err := svc.ExtractAndSave("https://example.com/photo.png", "factura", "image")

	assert.NoError(t, err)
	assert.JSONEq(t, expectedJSON, string(result))
	assert.Equal(t, 150, usage.TotalTokens)
}

func TestExtractAndSave_PDFSuccess(t *testing.T) {
	mockRepo := mocks_repositories.NewMockExtractionRepository(t)
	mockGemini := mocks_services.NewMockGeminiClient(t)

	svc := &extractionService{
		repo:       mockRepo,
		promptRepo: newMockPromptRepo(t, "contrato"),
		gemini:     mockGemini,
		fetchFn: func(rawURL string) ([]byte, string, error) {
			return []byte("fake-pdf-data"), "application/pdf", nil
		},
	}

	expectedJSON := `{"nombre":"Juan"}`
	mockGemini.On("Call", "contrato", mock.AnythingOfType("string"), mock.AnythingOfType("[]models.GeminiInlineImage"), mock.AnythingOfType("[]models.FieldSpec")).
		Return(expectedJSON, &models.TokenUsage{TotalTokens: 280}, nil).Once()

	mockRepo.On("Save", mock.AnythingOfType("*models.Extraction")).Return(nil).Once()

	result, usage, err := svc.ExtractAndSave("https://example.com/doc.pdf", "contrato", "pdf")

	assert.NoError(t, err)
	assert.JSONEq(t, expectedJSON, string(result))
	assert.Equal(t, 280, usage.TotalTokens)
}

func TestExtractAndSave_PromptNotConfigured(t *testing.T) {
	mockRepo := mocks_repositories.NewMockExtractionRepository(t)
	mockGemini := mocks_services.NewMockGeminiClient(t)
	mockPromptRepo := mocks_repositories.NewMockDocumentTypePromptRepository(t)

	mockPromptRepo.On("FindByDocumentType", "desconocido").
		Return((*models.DocumentTypePrompt)(nil), gorm.ErrRecordNotFound).Once()

	svc := &extractionService{
		repo:       mockRepo,
		promptRepo: mockPromptRepo,
		gemini:     mockGemini,
		fetchFn: func(rawURL string) ([]byte, string, error) {
			t.Fatal("fetchFn no debe llamarse cuando no hay prompt configurado")
			return nil, "", nil
		},
	}

	result, usage, err := svc.ExtractAndSave("https://example.com/doc.pdf", "desconocido", "pdf")

	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Nil(t, usage)

	var notConfigured *ErrPromptNotConfigured
	assert.ErrorAs(t, err, &notConfigured)
	assert.Equal(t, "desconocido", notConfigured.DocumentType)
	assert.Contains(t, err.Error(), "no hay prompt configurado")
}

func TestExtractAndSave_FetchError(t *testing.T) {
	svc := &extractionService{
		repo:       mocks_repositories.NewMockExtractionRepository(t),
		promptRepo: newMockPromptRepo(t, "factura"),
		gemini:     mocks_services.NewMockGeminiClient(t),
		fetchFn: func(rawURL string) ([]byte, string, error) {
			return nil, "", errors.New("network error")
		},
	}

	result, usage, err := svc.ExtractAndSave("https://example.com/photo.png", "factura", "image")

	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Nil(t, usage)
	assert.Contains(t, err.Error(), "network error")
}

func TestExtractAndSave_CannotInferMediaType(t *testing.T) {
	svc := &extractionService{
		repo:       mocks_repositories.NewMockExtractionRepository(t),
		promptRepo: newMockPromptRepo(t, "factura"),
		gemini:     mocks_services.NewMockGeminiClient(t),
		fetchFn: func(rawURL string) ([]byte, string, error) {
			return []byte("data"), "application/octet-stream", nil
		},
	}

	result, usage, err := svc.ExtractAndSave("https://example.com/file", "factura", "")

	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Nil(t, usage)
	assert.Contains(t, err.Error(), "no se pudo inferir")
}

func TestExtractAndSave_GeminiExtractionError(t *testing.T) {
	mockRepo := mocks_repositories.NewMockExtractionRepository(t)
	mockGemini := mocks_services.NewMockGeminiClient(t)

	svc := &extractionService{
		repo:       mockRepo,
		promptRepo: newMockPromptRepo(t, "factura"),
		gemini:     mockGemini,
		fetchFn: func(rawURL string) ([]byte, string, error) {
			return []byte("fake-image-data"), "image/png", nil
		},
	}

	mockGemini.On("Call", "factura", mock.AnythingOfType("string"), mock.AnythingOfType("[]models.GeminiInlineImage"), mock.AnythingOfType("[]models.FieldSpec")).
		Return("", (*models.TokenUsage)(nil), errors.New("gemini API error")).Once()

	result, usage, err := svc.ExtractAndSave("https://example.com/photo.png", "factura", "image")

	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Nil(t, usage)
	assert.Contains(t, err.Error(), "gemini API error")
}

func TestExtractAndSave_SaveError(t *testing.T) {
	mockRepo := mocks_repositories.NewMockExtractionRepository(t)
	mockGemini := mocks_services.NewMockGeminiClient(t)

	svc := &extractionService{
		repo:       mockRepo,
		promptRepo: newMockPromptRepo(t, "factura"),
		gemini:     mockGemini,
		fetchFn: func(rawURL string) ([]byte, string, error) {
			return []byte("fake-image-data"), "image/png", nil
		},
	}

	mockGemini.On("Call", "factura", mock.AnythingOfType("string"), mock.AnythingOfType("[]models.GeminiInlineImage"), mock.AnythingOfType("[]models.FieldSpec")).
		Return(`{"total":"100"}`, &models.TokenUsage{TotalTokens: 100}, nil).Once()

	mockRepo.On("Save", mock.AnythingOfType("*models.Extraction")).
		Return(errors.New("db connection error")).Once()

	result, _, err := svc.ExtractAndSave("https://example.com/photo.png", "factura", "image")

	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "db connection error")
}

func TestExtractAndSave_InvalidJSONResponse(t *testing.T) {
	mockRepo := mocks_repositories.NewMockExtractionRepository(t)
	mockGemini := mocks_services.NewMockGeminiClient(t)

	svc := &extractionService{
		repo:       mockRepo,
		promptRepo: newMockPromptRepo(t, "factura"),
		gemini:     mockGemini,
		fetchFn: func(rawURL string) ([]byte, string, error) {
			return []byte("fake-pdf-data"), "application/pdf", nil
		},
	}

	invalidJSON := "this is not valid json"

	mockGemini.On("Call", "factura", mock.AnythingOfType("string"), mock.AnythingOfType("[]models.GeminiInlineImage"), mock.AnythingOfType("[]models.FieldSpec")).
		Return(invalidJSON, &models.TokenUsage{TotalTokens: 50}, nil).Once()

	mockRepo.On("Save", mock.AnythingOfType("*models.Extraction")).Return(nil).Once()

	result, usage, err := svc.ExtractAndSave("https://example.com/doc.pdf", "factura", "pdf")

	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, 50, usage.TotalTokens)

	var fallback map[string]string
	err = json.Unmarshal(result, &fallback)
	assert.NoError(t, err)
	assert.Equal(t, invalidJSON, fallback["raw"])
}

func TestExtractAndSave_WithCustomPrompt(t *testing.T) {
	mockRepo := mocks_repositories.NewMockExtractionRepository(t)
	mockGemini := mocks_services.NewMockGeminiClient(t)
	mockPromptRepo := mocks_repositories.NewMockDocumentTypePromptRepository(t)

	customPrompt := &models.DocumentTypePrompt{
		ID:               1,
		DocumentType:     "factura",
		PromptComplement: "Reglas especiales para facturas: el total debe incluir IVA",
		Fields:           models.FieldSpecs{{Name: "total", Type: models.FieldTypeString}},
		IsActive:         true,
	}
	mockPromptRepo.On("FindByDocumentType", "factura").Return(customPrompt, nil).Once()

	svc := &extractionService{
		repo:       mockRepo,
		promptRepo: mockPromptRepo,
		gemini:     mockGemini,
		fetchFn: func(rawURL string) ([]byte, string, error) {
			return []byte("fake-pdf-data"), "application/pdf", nil
		},
	}

	expectedJSON := `{"total":"119.00"}`
	mockGemini.On("Call", "factura", mock.MatchedBy(func(prompt string) bool {
		return assert.Contains(t, prompt, "Reglas especiales para facturas") &&
			assert.Contains(t, prompt, "Tipo de documento esperado: factura") &&
			assert.Contains(t, prompt, "Campos a extraer: total")
	}), mock.AnythingOfType("[]models.GeminiInlineImage"), mock.AnythingOfType("[]models.FieldSpec")).
		Return(expectedJSON, &models.TokenUsage{TotalTokens: 360}, nil).Once()

	mockRepo.On("Save", mock.AnythingOfType("*models.Extraction")).Return(nil).Once()

	result, usage, err := svc.ExtractAndSave("https://example.com/factura.pdf", "factura", "pdf")

	assert.NoError(t, err)
	assert.JSONEq(t, expectedJSON, string(result))
	assert.Equal(t, 360, usage.TotalTokens)
}

func TestExtractAndSave_InvalidDocumentType_SkipsExtraction(t *testing.T) {
	mockRepo := mocks_repositories.NewMockExtractionRepository(t)
	mockGemini := mocks_services.NewMockGeminiClient(t)

	svc := &extractionService{
		repo:       mockRepo,
		promptRepo: newMockPromptRepo(t, "factura"),
		gemini:     mockGemini,
		fetchFn: func(rawURL string) ([]byte, string, error) {
			return []byte("fake-pdf-data"), "application/pdf", nil
		},
	}

	callUsage := &models.TokenUsage{TotalTokens: 55}
	mockGemini.On("Call", "factura", mock.AnythingOfType("string"), mock.AnythingOfType("[]models.GeminiInlineImage"), mock.AnythingOfType("[]models.FieldSpec")).
		Return("", callUsage, &ErrInvalidDocumentType{DocumentType: "factura"}).Once()

	result, usage, err := svc.ExtractAndSave("https://example.com/contrato.pdf", "factura", "pdf")

	assert.Error(t, err)
	assert.Nil(t, result)
	assert.NotNil(t, usage)
	assert.Equal(t, 55, usage.TotalTokens)

	var invalidDoc *ErrInvalidDocumentType
	assert.ErrorAs(t, err, &invalidDoc)
	assert.Equal(t, "factura", invalidDoc.DocumentType)

	mockRepo.AssertNotCalled(t, "Save", mock.Anything)
}

func TestExtractAndSave_InferImageFromURL(t *testing.T) {
	mockRepo := mocks_repositories.NewMockExtractionRepository(t)
	mockGemini := mocks_services.NewMockGeminiClient(t)

	svc := &extractionService{
		repo:       mockRepo,
		promptRepo: newMockPromptRepo(t, "factura"),
		gemini:     mockGemini,
		fetchFn: func(rawURL string) ([]byte, string, error) {
			return []byte("fake-image-data"), "application/octet-stream", nil
		},
	}

	mockGemini.On("Call", "factura", mock.AnythingOfType("string"), mock.AnythingOfType("[]models.GeminiInlineImage"), mock.AnythingOfType("[]models.FieldSpec")).
		Return(`{"total":"50"}`, &models.TokenUsage{TotalTokens: 75}, nil).Once()

	mockRepo.On("Save", mock.AnythingOfType("*models.Extraction")).Return(nil).Once()

	result, usage, err := svc.ExtractAndSave("https://example.com/photo.jpeg", "factura", "")

	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, 75, usage.TotalTokens)
}

// --- ExtractZipAndSave tests ---

func TestExtractZipAndSave_Success(t *testing.T) {
	mockRepo := mocks_repositories.NewMockExtractionRepository(t)
	mockGemini := mocks_services.NewMockGeminiClient(t)

	zipData := createTestZip(map[string][]byte{
		"factura1.pdf": []byte("pdf1"),
		"factura2.pdf": []byte("pdf2"),
	})

	svc := &extractionService{
		repo:       mockRepo,
		promptRepo: newMockPromptRepo(t, "factura"),
		gemini:     mockGemini,
		fetchFn: func(rawURL string) ([]byte, string, error) {
			return zipData, "application/zip", nil
		},
	}

	mockGemini.On("Call", "factura", mock.AnythingOfType("string"), mock.AnythingOfType("[]models.GeminiInlineImage"), mock.AnythingOfType("[]models.FieldSpec")).
		Return(`{"total":"100"}`, &models.TokenUsage{TotalTokens: 70}, nil).Times(2)
	mockRepo.On("Save", mock.AnythingOfType("*models.Extraction")).Return(nil).Times(2)

	results, usage, err := svc.ExtractZipAndSave("https://example.com/facturas.zip", "factura")

	assert.NoError(t, err)
	assert.Len(t, results, 2)
	for _, r := range results {
		assert.Equal(t, "success", r.Status)
		assert.NotNil(t, r.Data)
	}
	assert.Equal(t, 140, usage.TotalTokens)
}

func TestExtractZipAndSave_MixedResults(t *testing.T) {
	mockRepo := mocks_repositories.NewMockExtractionRepository(t)
	mockGemini := mocks_services.NewMockGeminiClient(t)

	zipData := createOrderedZip([]zipFile{
		{name: "factura.pdf", data: []byte("valid-pdf")},
		{name: "contrato.pdf", data: []byte("wrong-doc")},
		{name: "readme.txt", data: []byte("text file")},
	})

	svc := &extractionService{
		repo:       mockRepo,
		promptRepo: newMockPromptRepo(t, "factura"),
		gemini:     mockGemini,
		fetchFn: func(rawURL string) ([]byte, string, error) {
			return zipData, "application/zip", nil
		},
	}

	mockGemini.On("Call", "factura", mock.AnythingOfType("string"), mock.AnythingOfType("[]models.GeminiInlineImage"), mock.AnythingOfType("[]models.FieldSpec")).
		Return(`{"total":"150"}`, &models.TokenUsage{TotalTokens: 80}, nil).Once()
	mockGemini.On("Call", "factura", mock.AnythingOfType("string"), mock.AnythingOfType("[]models.GeminiInlineImage"), mock.AnythingOfType("[]models.FieldSpec")).
		Return("", &models.TokenUsage{TotalTokens: 25}, &ErrInvalidDocumentType{DocumentType: "factura"}).Once()

	mockRepo.On("Save", mock.AnythingOfType("*models.Extraction")).Return(nil).Once()

	results, usage, err := svc.ExtractZipAndSave("https://example.com/mix.zip", "factura")

	assert.NoError(t, err)
	assert.Len(t, results, 3)

	assert.Equal(t, "factura.pdf", results[0].Filename)
	assert.Equal(t, "success", results[0].Status)

	assert.Equal(t, "contrato.pdf", results[1].Filename)
	assert.Equal(t, "invalid_document", results[1].Status)

	assert.Equal(t, "readme.txt", results[2].Filename)
	assert.Equal(t, "error", results[2].Status)
	assert.Contains(t, results[2].Error, "tipo de archivo no soportado")

	assert.NotNil(t, usage)
}

func TestExtractZipAndSave_EmptyZip(t *testing.T) {
	zipData := createTestZip(map[string][]byte{})

	svc := &extractionService{
		repo:       mocks_repositories.NewMockExtractionRepository(t),
		promptRepo: newMockPromptRepo(t, "factura"),
		gemini:     mocks_services.NewMockGeminiClient(t),
		fetchFn: func(rawURL string) ([]byte, string, error) {
			return zipData, "application/zip", nil
		},
	}

	_, _, err := svc.ExtractZipAndSave("https://example.com/empty.zip", "factura")

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "no contiene archivos")
}

func TestExtractZipAndSave_FetchError(t *testing.T) {
	svc := &extractionService{
		repo:       mocks_repositories.NewMockExtractionRepository(t),
		promptRepo: newMockPromptRepo(t, "factura"),
		gemini:     mocks_services.NewMockGeminiClient(t),
		fetchFn: func(rawURL string) ([]byte, string, error) {
			return nil, "", errors.New("download failed")
		},
	}

	_, _, err := svc.ExtractZipAndSave("https://example.com/bad.zip", "factura")

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "download failed")
}

func TestExtractZipAndSave_CorruptZip(t *testing.T) {
	svc := &extractionService{
		repo:       mocks_repositories.NewMockExtractionRepository(t),
		promptRepo: newMockPromptRepo(t, "factura"),
		gemini:     mocks_services.NewMockGeminiClient(t),
		fetchFn: func(rawURL string) ([]byte, string, error) {
			return []byte("not a zip"), "application/zip", nil
		},
	}

	_, _, err := svc.ExtractZipAndSave("https://example.com/corrupt.zip", "factura")

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "error al leer el ZIP")
}

func TestExtractZipAndSave_PromptNotConfigured(t *testing.T) {
	mockPromptRepo := mocks_repositories.NewMockDocumentTypePromptRepository(t)
	mockPromptRepo.On("FindByDocumentType", "desconocido").
		Return((*models.DocumentTypePrompt)(nil), gorm.ErrRecordNotFound).Once()

	svc := &extractionService{
		repo:       mocks_repositories.NewMockExtractionRepository(t),
		promptRepo: mockPromptRepo,
		gemini:     mocks_services.NewMockGeminiClient(t),
		fetchFn: func(rawURL string) ([]byte, string, error) {
			t.Fatal("fetchFn no debe llamarse cuando no hay prompt configurado")
			return nil, "", nil
		},
	}

	_, _, err := svc.ExtractZipAndSave("https://example.com/x.zip", "desconocido")

	assert.Error(t, err)
	var notConfigured *ErrPromptNotConfigured
	assert.ErrorAs(t, err, &notConfigured)
	assert.Equal(t, "desconocido", notConfigured.DocumentType)
}
