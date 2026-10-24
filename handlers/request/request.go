package request

import "ia-extraction/models"

type ExtractionRequest struct {
	Link         string `json:"link"`
	MediaType    string `json:"media_type,omitempty"`    // vacío = inferir; "pdf" | "image" | "zip"
	DocumentType string `json:"document_type"`           // tipo de documento esperado: "factura", "contrato", etc.
}

type DocumentTypePromptRequest struct {
	DocumentType         string             `json:"document_type" binding:"required"`
	PromptComplement     string             `json:"prompt_complement" binding:"required"`
	Fields               []models.FieldSpec `json:"fields" binding:"required,min=1,dive"`
	IdentificationNumber string             `json:"identification_number,omitempty"`
	IsActive             *bool              `json:"is_active,omitempty"`
}
