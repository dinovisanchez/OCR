package services

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"ia-extraction/models"
	"io"
	"net/http"
	"net/url"
	"os"
)

const geminiModel = "gemini-3-flash-preview"

type geminiGenerateRequest struct {
	SystemInstruction *geminiContentBlock   `json:"systemInstruction,omitempty"`
	Contents          []geminiContentBlock  `json:"contents"`
	GenerationConfig  *geminiGenerationConfig `json:"generationConfig,omitempty"`
}

type geminiContentBlock struct {
	Role  string       `json:"role,omitempty"`
	Parts []geminiPart `json:"parts"`
}

type geminiInlineData struct {
	MimeType string `json:"mimeType"`
	Data     string `json:"data"`
}

type geminiPart struct {
	Text       string            `json:"text,omitempty"`
	InlineData *geminiInlineData `json:"inlineData,omitempty"`
}

type geminiThinkingConfig struct {
	ThinkingBudget int `json:"thinkingBudget"`
}

type geminiGenerationConfig struct {
	ResponseMimeType string                `json:"responseMimeType,omitempty"`
	ResponseSchema   *geminiSchema         `json:"responseSchema,omitempty"`
	Temperature      *float64              `json:"temperature,omitempty"`
	ThinkingConfig   *geminiThinkingConfig `json:"thinkingConfig,omitempty"`
}

// geminiSchema es un subconjunto de OpenAPI Schema que Gemini acepta como responseSchema.
// Soporta: type, properties, items, required, nullable.
type geminiSchema struct {
	Type       string                   `json:"type"`
	Properties map[string]*geminiSchema `json:"properties,omitempty"`
	Items      *geminiSchema            `json:"items,omitempty"`
	Required   []string                 `json:"required,omitempty"`
	Nullable   bool                     `json:"nullable,omitempty"`
}

// buildResponseSchema convierte FieldSpecs en un schema "object" que fuerza a
// Gemini a devolver exactamente esos campos con los tipos esperados.
// Todos los campos son nullable para que Gemini pueda devolver null cuando un
// dato no esté presente. Además agrega la llave `is_valid_document` (boolean,
// no-nullable) que Gemini debe marcar según si el documento corresponde o no
// al tipo esperado; esto permite unificar validación y extracción en una sola
// llamada.
//
// Notas sobre tipos compuestos:
//   - `array` requiere `items` en el schema de Gemini; por defecto los items
//     son strings nullables (cubre la mayoría de casos). Para arrays de otro
//     tipo hay que extender FieldSpec.
//   - `object` sin propiedades queda permisivo; Gemini acepta cualquier objeto.
func buildResponseSchema(fields []models.FieldSpec) *geminiSchema {
	props := make(map[string]*geminiSchema, len(fields)+1)
	required := make([]string, 0, len(fields)+1)

	props["is_valid_document"] = &geminiSchema{Type: "boolean"}
	required = append(required, "is_valid_document")

	for _, f := range fields {
		t := string(f.Type)
		if t == "" {
			t = "string"
		}
		fieldSchema := &geminiSchema{Type: t, Nullable: true}
		if t == "array" {
			fieldSchema.Items = &geminiSchema{Type: "string", Nullable: true}
		}
		props[f.Name] = fieldSchema
		required = append(required, f.Name)
	}
	return &geminiSchema{
		Type:       "object",
		Properties: props,
		Required:   required,
	}
}

type geminiUsageMetadata struct {
	PromptTokenCount     int `json:"promptTokenCount"`
	CandidatesTokenCount int `json:"candidatesTokenCount"`
	TotalTokenCount      int `json:"totalTokenCount"`
	ThinkingTokenCount   int `json:"thinkingTokenCount,omitempty"`
}

type geminiGenerateResponse struct {
	Candidates []struct {
		Content struct {
			Parts []geminiPart `json:"parts"`
		} `json:"content"`
	} `json:"candidates"`
	UsageMetadata *geminiUsageMetadata `json:"usageMetadata,omitempty"`
	Error         *struct {
		Code    int    `json:"code"`
		Message string `json:"message"`
		Status  string `json:"status"`
	} `json:"error"`
}

type GeminiClient interface {
	// Call ejecuta validación + extracción en una sola llamada. El response
	// schema incluye `is_valid_document` para que Gemini marque si el documento
	// corresponde al `documentType` esperado. Si es false, devuelve
	// `ErrInvalidDocumentType`; si es true, devuelve el JSON de extracción sin
	// esa llave.
	Call(documentType string, prompt string, images []models.GeminiInlineImage, fields []models.FieldSpec) (string, *models.TokenUsage, error)
}

type geminiClient struct{}

func NewGeminiClient() GeminiClient {
	return &geminiClient{}
}

const extractionSystemText = `Eres un extractor de datos de documentos e imágenes. Tu tarea consiste en validar y extraer en una sola respuesta.

Pasos:
1. Revisa si el documento corresponde al "Tipo de documento esperado" indicado en el prompt.
2. Si NO corresponde: devuelve "is_valid_document": false y null en TODOS los demás campos.
3. Si SÍ corresponde: devuelve "is_valid_document": true y los valores extraídos de los campos solicitados.

Reglas:
- Devuelve SOLO un objeto JSON válido según el schema, sin markdown, sin texto adicional, sin comentarios ni explicaciones.
- El valor de cada campo extraído puede ser string, número, booleano, array u objeto anidado según corresponda al contenido del documento.
- Si un campo no existe en el documento (pero el documento sí es del tipo esperado), usa null para ese campo.
- No inventes datos. Extrae únicamente lo que está presente en el documento.
- Sé estricto en la validación: si el documento no corresponde claramente al tipo esperado, marca is_valid_document como false.`

func (g *geminiClient) Call(documentType string, prompt string, images []models.GeminiInlineImage, fields []models.FieldSpec) (string, *models.TokenUsage, error) {
	schema := buildResponseSchema(fields)
	resp, usage, err := callGemini(extractionSystemText, prompt, images, schema)
	if err != nil {
		return "", usage, err
	}

	// Parsear respuesta para separar el flag de validación del resto del JSON.
	var parsed map[string]json.RawMessage
	if err := json.Unmarshal([]byte(resp), &parsed); err != nil {
		return "", usage, fmt.Errorf("gemini response parse: %w: %s", err, resp)
	}
	validRaw, ok := parsed["is_valid_document"]
	if !ok {
		return "", usage, fmt.Errorf("gemini response missing is_valid_document: %s", resp)
	}
	var isValid bool
	if err := json.Unmarshal(validRaw, &isValid); err != nil {
		return "", usage, fmt.Errorf("gemini is_valid_document parse: %w", err)
	}
	if !isValid {
		return "", usage, &ErrInvalidDocumentType{DocumentType: documentType}
	}

	// Documento válido: devolver el JSON sin la llave de validación.
	delete(parsed, "is_valid_document")
	cleaned, err := json.Marshal(parsed)
	if err != nil {
		return "", usage, fmt.Errorf("gemini response re-marshal: %w", err)
	}
	return string(cleaned), usage, nil
}

func callGemini(systemText, prompt string, images []models.GeminiInlineImage, schema *geminiSchema) (string, *models.TokenUsage, error) {
	apiKey := os.Getenv("GEMINI_API_KEY")
	if apiKey == "" {
		return "", nil, fmt.Errorf("GEMINI_API_KEY not set")
	}

	parts := make([]geminiPart, 0, 1+len(images))
	parts = append(parts, geminiPart{Text: prompt})
	for _, img := range images {
		if len(img.Data) == 0 {
			continue
		}
		mime := img.MIME
		if mime == "" {
			mime = "image/jpeg"
		}
		parts = append(parts, geminiPart{
			InlineData: &geminiInlineData{
				MimeType: mime,
				Data:     base64.StdEncoding.EncodeToString(img.Data),
			},
		})
	}

	reqBody := geminiGenerateRequest{
		SystemInstruction: &geminiContentBlock{
			Parts: []geminiPart{{Text: systemText}},
		},
		Contents: []geminiContentBlock{
			{Role: "user", Parts: parts},
		},
		GenerationConfig: &geminiGenerationConfig{
			ResponseMimeType: "application/json",
			ResponseSchema:   schema,
			Temperature:      func() *float64 { v := 0.0; return &v }(),
			ThinkingConfig:   &geminiThinkingConfig{ThinkingBudget: 1024},
		},
	}
	bodyBytes, err := json.Marshal(reqBody)
	if err != nil {
		return "", nil, err
	}

	endpoint := fmt.Sprintf(
		"https://generativelanguage.googleapis.com/v1beta/models/%s:generateContent?key=%s",
		geminiModel,
		url.QueryEscape(apiKey),
	)

	req, err := http.NewRequest(http.MethodPost, endpoint, bytes.NewBuffer(bodyBytes))
	if err != nil {
		return "", nil, err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", nil, err
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", nil, err
	}

	var geminiResp geminiGenerateResponse
	if err := json.Unmarshal(respBody, &geminiResp); err != nil {
		return "", nil, fmt.Errorf("gemini response parse: %w: %s", err, string(respBody))
	}
	if geminiResp.Error != nil {
		return "", nil, fmt.Errorf("gemini API error: %s (%s)", geminiResp.Error.Message, geminiResp.Error.Status)
	}
	if resp.StatusCode != http.StatusOK {
		return "", nil, fmt.Errorf("gemini HTTP %d: %s", resp.StatusCode, string(respBody))
	}
	if len(geminiResp.Candidates) == 0 {
		return "", nil, fmt.Errorf("no response from Gemini")
	}
	partsOut := geminiResp.Candidates[0].Content.Parts
	if len(partsOut) == 0 {
		return "", nil, fmt.Errorf("empty Gemini candidate content")
	}
	var out string
	for _, p := range partsOut {
		out += p.Text
	}

	var usage *models.TokenUsage
	if u := geminiResp.UsageMetadata; u != nil {
		usage = &models.TokenUsage{
			PromptTokens:     u.PromptTokenCount,
			CandidatesTokens: u.CandidatesTokenCount,
			TotalTokens:      u.TotalTokenCount,
			ThinkingTokens:   u.ThinkingTokenCount,
		}
	}
	return out, usage, nil
}
