package repositories

import (
	"ia-extraction/models"

	"gorm.io/gorm"
)

type DocumentTypePromptRepository interface {
	FindByDocumentType(documentType string) (*models.DocumentTypePrompt, error)
	Save(prompt *models.DocumentTypePrompt) error
	Update(prompt *models.DocumentTypePrompt) error
	Delete(id int64) error
	FindAll() ([]*models.DocumentTypePrompt, error)
}

type PostgresDocumentTypePromptRepository struct {
	db *gorm.DB
}

func NewPostgresDocumentTypePromptRepository(db *gorm.DB) *PostgresDocumentTypePromptRepository {
	return &PostgresDocumentTypePromptRepository{db: db}
}

func (r *PostgresDocumentTypePromptRepository) FindByDocumentType(documentType string) (*models.DocumentTypePrompt, error) {
	prompt := &models.DocumentTypePrompt{}
	if err := r.db.Where("document_type = ? AND is_active = ?", documentType, true).First(prompt).Error; err != nil {
		return nil, err
	}
	return prompt, nil
}

func (r *PostgresDocumentTypePromptRepository) Save(prompt *models.DocumentTypePrompt) error {
	return r.db.Create(prompt).Error
}

func (r *PostgresDocumentTypePromptRepository) Update(prompt *models.DocumentTypePrompt) error {
	return r.db.Model(&models.DocumentTypePrompt{}).
		Where("id = ?", prompt.ID).
		Updates(map[string]interface{}{
			"prompt_complement":     prompt.PromptComplement,
			"fields":                prompt.Fields,
			"identification_number": prompt.IdentificationNumber,
			"is_active":             prompt.IsActive,
		}).Error
}

func (r *PostgresDocumentTypePromptRepository) Delete(id int64) error {
	result := r.db.Delete(&models.DocumentTypePrompt{}, id)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}
	return nil
}

func (r *PostgresDocumentTypePromptRepository) FindAll() ([]*models.DocumentTypePrompt, error) {
	var prompts []*models.DocumentTypePrompt
	if err := r.db.Order("document_type").Find(&prompts).Error; err != nil {
		return nil, err
	}
	return prompts, nil
}
