package controller

import (
	"errors"
	"log"
	"net/http"
	"strings"

	"ia-extraction/handlers/request"
	"ia-extraction/services"

	"github.com/gin-gonic/gin"
)

type ExtractionController struct {
	Service services.ExtractionService
}

func NewExtractionController(service services.ExtractionService) *ExtractionController {
	return &ExtractionController{Service: service}
}

func (h *ExtractionController) Extract(ginCtx *gin.Context) {
	req := request.ExtractionRequest{}
	if err := ginCtx.ShouldBindJSON(&req); err != nil {
		ginCtx.JSON(http.StatusBadRequest, gin.H{"error": "Bad request"})
		return
	}

	if isZipRequest(req) {
		h.extractZip(ginCtx, req)
		return
	}

	iaResponse, tokenUsage, err := h.Service.ExtractAndSave(req.Link, req.DocumentType, req.MediaType)
	if err != nil {
		var notConfigured *services.ErrPromptNotConfigured
		if errors.As(err, &notConfigured) {
			ginCtx.JSON(http.StatusBadRequest, gin.H{
				"error":         err.Error(),
				"document_type": notConfigured.DocumentType,
			})
			return
		}
		var invalidDoc *services.ErrInvalidDocumentType
		if errors.As(err, &invalidDoc) {
			ginCtx.JSON(http.StatusUnprocessableEntity, gin.H{
				"error":            err.Error(),
				"document_type":    invalidDoc.DocumentType,
				"is_valid_document": false,
				"token_usage":      tokenUsage,
			})
			return
		}
		var protectedDoc *services.ErrProtectedDocument
		if errors.As(err, &protectedDoc) {
			ginCtx.JSON(http.StatusUnprocessableEntity, gin.H{
				"error": err.Error(),
			})
			return
		}
		log.Printf("ExtractAndSave error: %v", err)
		ginCtx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	ginCtx.JSON(http.StatusOK, gin.H{
		"data":        iaResponse,
		"token_usage": tokenUsage,
	})
}

func (h *ExtractionController) extractZip(ginCtx *gin.Context, req request.ExtractionRequest) {
	results, tokenUsage, err := h.Service.ExtractZipAndSave(req.Link, req.DocumentType)
	if err != nil {
		var notConfigured *services.ErrPromptNotConfigured
		if errors.As(err, &notConfigured) {
			ginCtx.JSON(http.StatusBadRequest, gin.H{
				"error":         err.Error(),
				"document_type": notConfigured.DocumentType,
			})
			return
		}
		log.Printf("ExtractZipAndSave error: %v", err)
		ginCtx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	ginCtx.JSON(http.StatusOK, gin.H{
		"data":        results,
		"token_usage": tokenUsage,
	})
}

func isZipRequest(req request.ExtractionRequest) bool {
	mt := strings.ToLower(strings.TrimSpace(req.MediaType))
	if mt == "zip" {
		return true
	}
	return strings.HasSuffix(strings.ToLower(req.Link), ".zip")
} 