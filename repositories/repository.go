package repositories

import (
	"database/sql"

	"ia-extraction/models"

	_ "github.com/lib/pq"
)

type ExtractionRepository interface {
	Save(extraction *models.Extraction) error
}

// Implementación para Postgres
type PostgresExtractionRepository struct {
	db *sql.DB
}

func NewPostgresExtractionRepository(db *sql.DB) *PostgresExtractionRepository {
	return &PostgresExtractionRepository{db: db}
}

func (r *PostgresExtractionRepository) Save(extraction *models.Extraction) error {
	query := `INSERT INTO extractions (document_type, document_identification, request, response) VALUES ($1, $2, $3, $4) RETURNING id, created_at`
	err := r.db.QueryRow(query, extraction.DocumentType, extraction.DocumentIdentification, extraction.Request, extraction.Response).Scan(&extraction.ID, &extraction.CreatedAt)
	if err != nil {
		return err
	}
	return nil
} 