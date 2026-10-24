package controller

import (
	"log"
	"net/http"
	"strconv"

	"ia-extraction/handlers/request"
	"ia-extraction/models"
	"ia-extraction/repositories"

	"github.com/gin-gonic/gin"
)

type DocumentTypePromptController struct {
	Repo repositories.DocumentTypePromptRepository
}

func NewDocumentTypePromptController(repo repositories.DocumentTypePromptRepository) *DocumentTypePromptController {
	return &DocumentTypePromptController{Repo: repo}
}

func (c *DocumentTypePromptController) List(ginCtx *gin.Context) {
	prompts, err := c.Repo.FindAll()
	if err != nil {
		log.Printf("FindAll error: %v", err)
		ginCtx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	ginCtx.JSON(http.StatusOK, gin.H{"data": prompts})
}

func (c *DocumentTypePromptController) GetByDocumentType(ginCtx *gin.Context) {
	docType := ginCtx.Param("document_type")
	prompt, err := c.Repo.FindByDocumentType(docType)
	if err != nil {
		ginCtx.JSON(http.StatusNotFound, gin.H{"error": "prompt not found"})
		return
	}
	ginCtx.JSON(http.StatusOK, gin.H{"data": prompt})
}

func (c *DocumentTypePromptController) Create(ginCtx *gin.Context) {
	req := request.DocumentTypePromptRequest{}
	if err := ginCtx.ShouldBindJSON(&req); err != nil {
		ginCtx.JSON(http.StatusBadRequest, gin.H{"error": "Bad request"})
		return
	}

	isActive := true
	if req.IsActive != nil {
		isActive = *req.IsActive
	}

	prompt := &models.DocumentTypePrompt{
		DocumentType:         req.DocumentType,
		PromptComplement:     req.PromptComplement,
		Fields:               req.Fields,
		IdentificationNumber: req.IdentificationNumber,
		IsActive:             isActive,
	}

	if err := c.Repo.Save(prompt); err != nil {
		log.Printf("Save error: %v", err)
		ginCtx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	ginCtx.JSON(http.StatusCreated, gin.H{"data": prompt})
}

func (c *DocumentTypePromptController) Update(ginCtx *gin.Context) {
	id, err := strconv.ParseInt(ginCtx.Param("id"), 10, 64)
	if err != nil {
		ginCtx.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}

	req := request.DocumentTypePromptRequest{}
	if err := ginCtx.ShouldBindJSON(&req); err != nil {
		ginCtx.JSON(http.StatusBadRequest, gin.H{"error": "Bad request"})
		return
	}

	isActive := true
	if req.IsActive != nil {
		isActive = *req.IsActive
	}

	prompt := &models.DocumentTypePrompt{
		ID:                   id,
		PromptComplement:     req.PromptComplement,
		Fields:               req.Fields,
		IdentificationNumber: req.IdentificationNumber,
		IsActive:             isActive,
	}

	if err := c.Repo.Update(prompt); err != nil {
		log.Printf("Update error: %v", err)
		ginCtx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	ginCtx.JSON(http.StatusOK, gin.H{"data": prompt})
}

func (c *DocumentTypePromptController) Delete(ginCtx *gin.Context) {
	id, err := strconv.ParseInt(ginCtx.Param("id"), 10, 64)
	if err != nil {
		ginCtx.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}

	if err := c.Repo.Delete(id); err != nil {
		log.Printf("Delete error: %v", err)
		ginCtx.JSON(http.StatusNotFound, gin.H{"error": "prompt not found"})
		return
	}
	ginCtx.JSON(http.StatusOK, gin.H{"message": "deleted"})
}
