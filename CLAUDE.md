# ia-extraction

Microservicio en Go que extrae datos estructurados de PDFs e imágenes usando Gemini (multimodal).

## Arquitectura

Patrón de capas: Controller → Service → Repository.

```
main.go                          # Entrypoint, carga .env y arranca router
router/router.go                 # Gin router, inyección de dependencias
handlers/controller/extraction.go # Controller HTTP (Gin)
handlers/request/request.go      # DTOs de request
services/extraction_service.go   # Lógica: descarga documento, envía a Gemini, persiste
services/gemini_client.go        # Cliente HTTP a Gemini API (gemini-3-flash-preview)
repositories/repository.go       # Interfaz + implementación Postgres con database/sql
models/entity.go                 # Entidades: Extraction, GeminiInlineImage, TokenUsage
config/config.go                 # Variables de entorno y configuración
provider/postgres/database.go    # Conexión GORM + tracing SignalFx
provider/postgres/migrate.go     # Migraciones con golang-migrate
```

## Stack

- **Go 1.24** (module: `ia-extraction`)
- **Gin** para HTTP
- **GORM** + `lib/pq` para Postgres
- **golang-migrate** para migraciones SQL (`provider/postgres/migrations/`)
- **Gemini API** (`gemini-3-flash-preview`, multimodal — soporta PDFs e imágenes con contenido visual)
- **bia-commons-go** (librería interna Bia Energy) para router y utils
- **SignalFx** para tracing de DB

## Endpoints

### Extracción
```
POST /ms-ia-extraction/extraction_information
Body:
{
  "link": "<url-al-documento>",
  "media_type": "pdf" | "image" | "zip" | "",  // opcional, se infiere si está vacío
  "document_type": "factura"                    // tipo de documento esperado
}
Response:
{
  "data": { ... },                              // diccionario con las llaves definidas en fields
  "token_usage": { "total_tokens": 0, ... }
}
```

Requiere que exista un prompt registrado para `document_type` (con sus `fields`). Si no existe → HTTP 400.

Si el `link` apunta a un `.zip`, el response `data` es un array: `[{ "filename", "status", "data" | "error" }]`.

### Gestión de prompts por tipo de documento
```
GET    /ms-ia-extraction/document_type_prompts
GET    /ms-ia-extraction/document_type_prompts/:document_type
POST   /ms-ia-extraction/document_type_prompts
PUT    /ms-ia-extraction/document_type_prompts/:id
DELETE /ms-ia-extraction/document_type_prompts/:id
```

> Las rutas usan el prefijo `/ms-ia-extraction` directo (sin segmento `/api`), tal como lo monta `bia-commons-go` con `BiaRouterConfig.PrefixPath`. El grupo aplica `middlewares.AuthValidator`.

POST/PUT body:
```json
{
  "document_type": "factura",
  "prompt_complement": "Reglas específicas para facturas...",
  "fields": [
    {"name": "total", "type": "number"},
    {"name": "fecha_emision", "type": "string"},
    {"name": "nit_emisor", "type": "string"},
    {"name": "items", "type": "array"},
    {"name": "is_paid", "type": "boolean"}
  ],
  "is_active": true
}
```

Tipos válidos para `fields[].type`: `string`, `integer`, `number`, `boolean`, `array`, `object`. El tipo se usa para construir un **JSON Schema** que se envía a Gemini como `responseSchema`, forzando el formato de la respuesta.

## Variables de entorno

| Variable        | Default              |
|-----------------|----------------------|
| MS_NAME         | bia-ia-extraction    |
| DB_USER         | postgres             |
| DB_PASSWORD     | (vacío)              |
| DB_HOST         | localhost            |
| DB_PORT         | 5432                 |
| DB_NAME         | bia-ia-extraction    |
| GO_ENVIRONMENT  | develop              |
| GEMINI_API_KEY  | (requerida)          |
| PORT            | 8080                 |

## Flujo principal

1. Recibe POST con `link`, `document_type` y opcionalmente `media_type`
2. Consulta el prompt asociado al `document_type` en la tabla `document_type_prompts`. Si no existe → HTTP 400
3. Descarga el documento (PDF, imagen o ZIP) desde la URL
4. Infiere el tipo de medio a partir de `media_type`, Content-Type o extensión de la URL
5. Si es ZIP: descomprime y procesa cada archivo interno por separado
6. **Validación** (llamada liviana a Gemini): si el documento no corresponde al tipo esperado → HTTP 422, sin ejecutar extracción
7. Si pasa validación: construye el prompt con `prompt_complement` + `fields` del registro, envía a Gemini
8. Persiste request y response en tabla `extractions`
9. Devuelve un diccionario con las llaves de `fields` (o un array si fue ZIP) + token usage

## Base de datos

**`extractions`**: id (SERIAL PK), document_type (TEXT), document_identification (BIGINT), request (TEXT), response (TEXT), created_at (TIMESTAMPTZ).

**`document_type_prompts`**: id (BIGSERIAL PK), document_type (VARCHAR UNIQUE), prompt_complement (TEXT), fields (JSONB con `[{"name", "type"}]`), is_active (BOOLEAN), created_at, updated_at. Configura los prompts y campos a extraer por tipo de documento. El endpoint de extracción requiere un registro aquí para funcionar. Los `fields` se traducen a JSON Schema y se pasan a Gemini como `responseSchema`.

## Comandos

```bash
# Ejecutar
go run main.go

# Requiere .env con al menos GEMINI_API_KEY y credenciales de DB
```
