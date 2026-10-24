# Servicio de Extracción de Información (Clean Architecture)

## Descripción
Servicio backend en Go que recibe un enlace a una imagen o PDF y un json con las respuesta esperada, extrae información relevante usando un modelo de IA (ChatGPT o4-mini) y la retorna en formato JSON. Utiliza PostgreSQL para almacenar tanto el request como el response en formato JSONB y está estructurado siguiendo Clean Architecture.

## Estructura del Proyecto

```
.
├── cmd/                    # Punto de entrada principal (main.go)
│   └── main.go
├── config/                 # Configuración y variables de entorno
│   └── config.go
├── handlers/               # HTTP handlers (delivery/controllers)
│   └── extraction_handler.go
├── services/               # Lógica de negocio (casos de uso)
│   └── extraction_service.go
├── repositories/           # Acceso a datos (PostgreSQL, interfaces y persistencia)
│   └── extraction_repository.go
├── models/                 # Definición de entidades/structs
│   └── extraction.go
├── db/
│   └── migrations/         # Scripts de migración de base de datos
├── .env                    # Variables de entorno (no versionar)
├── .env.example            # Ejemplo de variables de entorno
├── go.mod                  # Dependencias de Go (incluye github.com/ledongthuc/pdf)
├── go.sum                  # Checksum de dependencias
├── Dockerfile
├── docker-compose.prod.yml
├── docker-compose.test.yml
└── README.md
```
- **cmd/**: Contiene el archivo principal para iniciar la aplicación.
- **config/**: Configuración de la app y carga de variables de entorno.
- **handlers/**: Encargados de recibir y responder las peticiones HTTP.
- **services/**: Lógica de negocio, donde se realiza la extracción y procesamiento de PDFs.
- **repositories/**: Acceso y persistencia de datos en la base de datos.
- **models/**: Definición de las entidades y estructuras de datos.
- **db/migrations/**: Scripts para la gestión y migración de la base de datos.
- **go.mod/go.sum**: Gestión de dependencias, incluyendo la librería de extracción de PDF.
- **.env/.env.example**: Variables de entorno necesarias para la configuración.
- **Dockerfile y docker-compose**: Archivos para contenerización y despliegue.

## Ejecución Local

1. Copia `.env.example` a `.env` y completa los valores.
2. Levanta los servicios:
   ```bash
   docker-compose -f docker-compose.test.yml up --build
   ```
3. El backend estará disponible en `http://localhost:8080`.

## Producción

```bash
docker-compose -f docker-compose.prod.yml up -d --build
```

## Migraciones

- Agrega tus scripts SQL en `db/migrations/`.
- Para aplicar migraciones manualmente:
  ```bash
  docker-compose -f docker-compose.test.yml exec app goose -dir db/migrations postgres "$DATABASE_URL" up
  ```

## Variables de Entorno

- `PORT`: Puerto del servicio HTTP
- `DATABASE_URL`: URL de conexión a PostgreSQL
- `OPENAI_API_KEY`: API Key de OpenAI

## Tests

```bash
docker-compose -f docker-compose.test.yml exec app go test ./...
```

## Problema que resuelve
Permite extraer información estructurada de imágenes o PDFs accesibles por URL, usando IA, dando como respuesta el json solicitado como respuesta con la informacion extraida desde el link sea imagen o pdf.


## Funcionamiento actualizado

1. **Recepción de Enlace:**  
   El servicio recibe una petición HTTP con un enlace a una imagen o PDF y un Json de respuesta esperado.

2. **Descarga y extracción de PDF:**  
   Si el enlace es un PDF, el servicio descarga el archivo y extrae su texto usando la librería mencionada.

3. **Procesamiento y Envío a ChatGPT:**  
   El servicio utiliza el modelo **o4-mini** de ChatGPT y le envía el siguiente prompt:
   ```
   Extrae la información del link adjunto y genera un json con la estructura enviada
   [Texto extraído del PDF]
   ```
   El texto extraído del PDF se adjunta al final del prompt.

4. **Respuesta y Salida:**  
   El modelo responde con un JSON que contiene los campos requeridos en el Json, datos que fueron extraídos del contenido del enlace.  
   El servicio imprime esta respuesta en la consola y almacena tanto el request como el response en la base de datos.

## Ejemplo de Uso

### Request

```
 POST http://localhost:8080/ms-ia-extraction/extraction_information' \
{
  "link": "https://www.bia.app/upload-olibia/odoo.production.lead.25938_x_studio_rut.pdf",
  "prompt": "A partir del siguiente texto, extrae y organiza la información en un JSON estructurado. El JSON debe reflejar claramente los campos de entidad comercial, información fiscal, representantes, fechas relevantes y otros elementos clave. Asegúrate de preservar los nombres propios tal como aparecen. Usa un esquema como el que se muestra a continuación: {\"nombre_cliente\": \"\", \"tipo_documento\": \"\", \"numero_documento\": \"\", \"fecha_emision\": \"\", \"direccion\" : \"\", \"municipio\" : \"\", \"departamento\" : \"\", \"telefono\" : \"\", \"email\" : \"\", \"ciiu_principal\" : \"\", \"ciiu_secundarios\" : \"\", \"responsabilidades\": [], \"obligaciones\" : [], \"representante_legal\" : \"\", \"identicacion_representante\" : \"\" }"
}
```
### Respuesta esperada (impresa en consola)

```json\n{\n  \"nombre_cliente\": \"SUPERMERCADOS OPTIMERKA SAS\",\n  \"tipo_documento\": \"Persona jurídica\",\n  \"numero_documento\": \"901835098\",\n  \"fecha_emision\": \"2024-07-22\",\n  \"direccion\": \"CARERRA 1 15 73\",\n  \"municipio\": \"Pitalito\",\n  \"departamento\": \"Huila\",\n  \"telefono\": \"3118708302\",\n  \"email\": \"optimerkafacturase@gmail.com\",\n  \"ciiu_principal\": \"4711\",\n  \"ciiu_secundarios\": [\"4723\"],\n  \"responsabilidades\": [\n    \"Impto. renta y compl. régimen ordinar\",\n    \"Retención en la fuente a título de rent\",\n    \"Retención en la fuente en el impuesto\",\n    \"Informante de exogena\",\n    \"Obligación facturar por ingresos bienes\",\n    \"Obligado a llevar contabilidad\",\n    \"Impuesto sobre las ventas - IVA\",\n    \"Facturador electrónico\",\n    \"Informante de Beneficiarios Finales\"\n  ],\n  \"obligaciones\": [],\n  \"representante_legal\": \"COY CARO EVELIO\",\n  \"identicacion_representante\": \"12228491\"\n}\n```

## Requisitos

- Acceso a la API de ChatGPT con el modelo **o4-mini**
- El enlace debe ser accesible públicamente y contener una imagen o PDF legible

## Cambios recientes

### Extracción de texto de PDF

- Ahora el servicio descarga el PDF desde el enlace proporcionado, extrae su texto y lo adjunta al final del prompt antes de enviarlo a OpenAI.
- Se agregó la dependencia `github.com/ledongthuc/pdf` para la extracción de texto desde archivos PDF.
- El flujo actualizado es:
  1. Recibe el enlace a un PDF y el JSON esperado.
  2. Descarga el PDF y extrae su texto.
  3. Añade el texto extraído al final del prompt.
  4. Envía el prompt completo (instrucción + texto del PDF) a ChatGPT.
  5. Almacena el request y response en la base de datos.

### Dependencias nuevas

- `github.com/ledongthuc/pdf`: Para extraer texto de archivos PDF.
