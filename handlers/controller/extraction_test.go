package controller

import (
	"bytes"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	mocks_services "ia-extraction/mocks/services"
	"ia-extraction/models"
	"ia-extraction/services"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

func setupRouter() *gin.Engine {
	gin.SetMode(gin.TestMode)
	return gin.New()
}

func TestExtractionController_Extract_Success(t *testing.T) {
	mockService := mocks_services.NewMockExtractionService(t)
	ctrl := NewExtractionController(mockService)

	router := setupRouter()
	router.POST("/extraction_information", ctrl.Extract)

	reqBody := map[string]interface{}{
		"link":          "https://example.com/doc.pdf",
		"document_type": "factura",
		"media_type":    "pdf",
	}
	body, _ := json.Marshal(reqBody)

	expectedResponse := json.RawMessage(`{"total":"100","fecha":"2026-01-01"}`)
	expectedUsage := &models.TokenUsage{TotalTokens: 150}

	mockService.On("ExtractAndSave",
		"https://example.com/doc.pdf",
		"factura",
		"pdf",
	).Return(expectedResponse, expectedUsage, nil).Once()

	req, _ := http.NewRequest("POST", "/extraction_information", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp map[string]json.RawMessage
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	assert.NoError(t, err)
	assert.JSONEq(t, `{"total":"100","fecha":"2026-01-01"}`, string(resp["data"]))
	assert.Contains(t, string(resp["token_usage"]), `"total_tokens":150`)
}

func TestExtractionController_Extract_InvalidBody(t *testing.T) {
	mockService := mocks_services.NewMockExtractionService(t)
	ctrl := NewExtractionController(mockService)

	router := setupRouter()
	router.POST("/extraction_information", ctrl.Extract)

	req, _ := http.NewRequest("POST", "/extraction_information", bytes.NewBufferString(`{"invalid json`))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestExtractionController_Extract_ServiceError(t *testing.T) {
	mockService := mocks_services.NewMockExtractionService(t)
	ctrl := NewExtractionController(mockService)

	router := setupRouter()
	router.POST("/extraction_information", ctrl.Extract)

	reqBody := map[string]interface{}{
		"link":          "https://example.com/doc.pdf",
		"document_type": "factura",
	}
	body, _ := json.Marshal(reqBody)

	mockService.On("ExtractAndSave",
		"https://example.com/doc.pdf",
		"factura",
		"",
	).Return(nil, nil, errors.New("extraction failed")).Once()

	req, _ := http.NewRequest("POST", "/extraction_information", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusInternalServerError, w.Code)

	var resp map[string]string
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	assert.NoError(t, err)
	assert.Equal(t, "extraction failed", resp["error"])
}

func TestExtractionController_Extract_EmptyBody(t *testing.T) {
	mockService := mocks_services.NewMockExtractionService(t)
	ctrl := NewExtractionController(mockService)

	router := setupRouter()
	router.POST("/extraction_information", ctrl.Extract)

	req, _ := http.NewRequest("POST", "/extraction_information", bytes.NewBufferString(`{}`))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	mockService.On("ExtractAndSave",
		"",
		"",
		"",
	).Return(nil, nil, errors.New("link required")).Once()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

func TestExtractionController_Extract_InvalidDocumentType(t *testing.T) {
	mockService := mocks_services.NewMockExtractionService(t)
	ctrl := NewExtractionController(mockService)

	router := setupRouter()
	router.POST("/extraction_information", ctrl.Extract)

	reqBody := map[string]interface{}{
		"link":          "https://example.com/contrato.pdf",
		"document_type": "factura",
		"media_type":    "pdf",
	}
	body, _ := json.Marshal(reqBody)

	expectedUsage := &models.TokenUsage{TotalTokens: 200}
	mockService.On("ExtractAndSave",
		"https://example.com/contrato.pdf",
		"factura",
		"pdf",
	).Return(nil, expectedUsage, &services.ErrInvalidDocumentType{DocumentType: "factura"}).Once()

	req, _ := http.NewRequest("POST", "/extraction_information", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnprocessableEntity, w.Code)

	var resp map[string]json.RawMessage
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	assert.NoError(t, err)

	var errMsg string
	_ = json.Unmarshal(resp["error"], &errMsg)
	assert.Contains(t, errMsg, "no cumple con el formato esperado")

	var docType string
	_ = json.Unmarshal(resp["document_type"], &docType)
	assert.Equal(t, "factura", docType)
}

func TestExtractionController_Extract_PromptNotConfigured(t *testing.T) {
	mockService := mocks_services.NewMockExtractionService(t)
	ctrl := NewExtractionController(mockService)

	router := setupRouter()
	router.POST("/extraction_information", ctrl.Extract)

	reqBody := map[string]interface{}{
		"link":          "https://example.com/doc.pdf",
		"document_type": "desconocido",
		"media_type":    "pdf",
	}
	body, _ := json.Marshal(reqBody)

	mockService.On("ExtractAndSave",
		"https://example.com/doc.pdf",
		"desconocido",
		"pdf",
	).Return(nil, nil, &services.ErrPromptNotConfigured{DocumentType: "desconocido"}).Once()

	req, _ := http.NewRequest("POST", "/extraction_information", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)

	var resp map[string]json.RawMessage
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	assert.NoError(t, err)

	var errMsg string
	_ = json.Unmarshal(resp["error"], &errMsg)
	assert.Contains(t, errMsg, "no hay prompt configurado")

	var docType string
	_ = json.Unmarshal(resp["document_type"], &docType)
	assert.Equal(t, "desconocido", docType)
}

func TestExtractionController_Extract_ImageMediaType(t *testing.T) {
	mockService := mocks_services.NewMockExtractionService(t)
	ctrl := NewExtractionController(mockService)

	router := setupRouter()
	router.POST("/extraction_information", ctrl.Extract)

	reqBody := map[string]interface{}{
		"link":          "https://example.com/photo.png",
		"document_type": "cedula",
		"media_type":    "image",
	}
	body, _ := json.Marshal(reqBody)

	expectedResponse := json.RawMessage(`{"nombre":"Juan","numero":"123456"}`)
	expectedUsage := &models.TokenUsage{TotalTokens: 120}

	mockService.On("ExtractAndSave",
		"https://example.com/photo.png",
		"cedula",
		"image",
	).Return(expectedResponse, expectedUsage, nil).Once()

	req, _ := http.NewRequest("POST", "/extraction_information", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp map[string]json.RawMessage
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	assert.NoError(t, err)
	assert.JSONEq(t, `{"nombre":"Juan","numero":"123456"}`, string(resp["data"]))
}

func TestExtractionController_Extract_ZipSuccess(t *testing.T) {
	mockService := mocks_services.NewMockExtractionService(t)
	ctrl := NewExtractionController(mockService)

	router := setupRouter()
	router.POST("/extraction_information", ctrl.Extract)

	reqBody := map[string]interface{}{
		"link":          "https://example.com/facturas.zip",
		"document_type": "factura",
		"media_type":    "zip",
	}
	body, _ := json.Marshal(reqBody)

	expectedResults := []models.ZipEntryResult{
		{Filename: "factura1.pdf", Status: "success", Data: json.RawMessage(`{"total":"100"}`)},
		{Filename: "readme.txt", Status: "error", Error: "tipo de archivo no soportado: .txt"},
	}
	expectedUsage := &models.TokenUsage{TotalTokens: 200}

	mockService.On("ExtractZipAndSave",
		"https://example.com/facturas.zip",
		"factura",
	).Return(expectedResults, expectedUsage, nil).Once()

	req, _ := http.NewRequest("POST", "/extraction_information", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp map[string]json.RawMessage
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	assert.NoError(t, err)
	assert.Contains(t, string(resp["data"]), "factura1.pdf")
	assert.Contains(t, string(resp["data"]), "success")
	assert.Contains(t, string(resp["token_usage"]), `"total_tokens":200`)
}

func TestExtractionController_Extract_ZipByExtension(t *testing.T) {
	mockService := mocks_services.NewMockExtractionService(t)
	ctrl := NewExtractionController(mockService)

	router := setupRouter()
	router.POST("/extraction_information", ctrl.Extract)

	reqBody := map[string]interface{}{
		"link":          "https://example.com/docs.zip",
		"document_type": "factura",
	}
	body, _ := json.Marshal(reqBody)

	mockService.On("ExtractZipAndSave",
		"https://example.com/docs.zip",
		"factura",
	).Return([]models.ZipEntryResult{}, &models.TokenUsage{}, nil).Once()

	req, _ := http.NewRequest("POST", "/extraction_information", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestExtractionController_Extract_ZipError(t *testing.T) {
	mockService := mocks_services.NewMockExtractionService(t)
	ctrl := NewExtractionController(mockService)

	router := setupRouter()
	router.POST("/extraction_information", ctrl.Extract)

	reqBody := map[string]interface{}{
		"link":          "https://example.com/corrupt.zip",
		"document_type": "factura",
		"media_type":    "zip",
	}
	body, _ := json.Marshal(reqBody)

	mockService.On("ExtractZipAndSave",
		"https://example.com/corrupt.zip",
		"factura",
	).Return(nil, nil, errors.New("error al leer el ZIP")).Once()

	req, _ := http.NewRequest("POST", "/extraction_information", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
	assert.Contains(t, w.Body.String(), "error al leer el ZIP")
}

func TestExtractionController_Extract_ZipPromptNotConfigured(t *testing.T) {
	mockService := mocks_services.NewMockExtractionService(t)
	ctrl := NewExtractionController(mockService)

	router := setupRouter()
	router.POST("/extraction_information", ctrl.Extract)

	reqBody := map[string]interface{}{
		"link":          "https://example.com/x.zip",
		"document_type": "desconocido",
		"media_type":    "zip",
	}
	body, _ := json.Marshal(reqBody)

	mockService.On("ExtractZipAndSave",
		"https://example.com/x.zip",
		"desconocido",
	).Return(nil, nil, &services.ErrPromptNotConfigured{DocumentType: "desconocido"}).Once()

	req, _ := http.NewRequest("POST", "/extraction_information", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Contains(t, w.Body.String(), "no hay prompt configurado")
}
