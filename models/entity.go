package models

import (
	"database/sql/driver"
	"encoding/json"
	"fmt"
	"time"
)

type Extraction struct {
	ID                     int64   `json:"id"`
	DocumentType           string  `json:"document_type"`
	DocumentIdentification *string `json:"document_identification,omitempty"`
	Request                string  `json:"request"`
	Response               string  `json:"response"`
	CreatedAt              string  `json:"created_at"`
}

// GeminiInlineImage is raw image/document bytes sent to Gemini as inlineData.
type GeminiInlineImage struct {
	MIME string
	Data []byte
}

type TokenUsage struct {
	PromptTokens     int `json:"prompt_tokens"`
	CandidatesTokens int `json:"candidates_tokens"`
	TotalTokens      int `json:"total_tokens"`
	ThinkingTokens   int `json:"thinking_tokens,omitempty"`
}

// FieldType representa el tipo de dato esperado para un campo de extracción.
// Mapea directamente a tipos válidos de JSON Schema soportados por Gemini.
type FieldType string

const (
	FieldTypeString  FieldType = "string"
	FieldTypeInteger FieldType = "integer"
	FieldTypeNumber  FieldType = "number"
	FieldTypeBoolean FieldType = "boolean"
	FieldTypeArray   FieldType = "array"
	FieldTypeObject  FieldType = "object"
)

// FieldSpec describe un campo a extraer: nombre y tipo esperado.
type FieldSpec struct {
	Name string    `json:"name"`
	Type FieldType `json:"type"`
}

// FieldSpecs es una lista de FieldSpec persistida como JSONB en Postgres.
type FieldSpecs []FieldSpec

// Value implementa driver.Valuer para serializar a JSONB.
func (f FieldSpecs) Value() (driver.Value, error) {
	if f == nil {
		return "[]", nil
	}
	return json.Marshal(f)
}

// Scan implementa sql.Scanner para deserializar desde JSONB.
func (f *FieldSpecs) Scan(value interface{}) error {
	if value == nil {
		*f = FieldSpecs{}
		return nil
	}
	var data []byte
	switch v := value.(type) {
	case []byte:
		data = v
	case string:
		data = []byte(v)
	default:
		return fmt.Errorf("FieldSpecs.Scan: tipo no soportado %T", value)
	}
	return json.Unmarshal(data, f)
}

type DocumentTypePrompt struct {
	ID                   int64      `json:"id" gorm:"column:id;primaryKey;autoIncrement"`
	DocumentType         string     `json:"document_type" gorm:"column:document_type;type:varchar(255);uniqueIndex;not null"`
	PromptComplement     string     `json:"prompt_complement" gorm:"column:prompt_complement;type:text;not null"`
	Fields               FieldSpecs `json:"fields" gorm:"column:fields;type:jsonb;not null;default:'[]'"`
	IdentificationNumber string     `json:"identification_number" gorm:"column:identification_number;type:varchar(255)"`
	IsActive             bool       `json:"is_active" gorm:"column:is_active;default:true"`
	CreatedAt            time.Time  `json:"created_at" gorm:"column:created_at;autoCreateTime"`
	UpdatedAt            time.Time  `json:"updated_at" gorm:"column:updated_at;autoUpdateTime"`
}

func (DocumentTypePrompt) TableName() string {
	return "document_type_prompts"
}

type ZipEntryResult struct {
	Filename string          `json:"filename"`
	Status   string          `json:"status"`
	Data     json.RawMessage `json:"data,omitempty"`
	Error    string          `json:"error,omitempty"`
}
