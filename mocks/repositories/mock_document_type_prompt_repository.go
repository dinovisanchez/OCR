package mocks_repositories

import (
	"ia-extraction/models"

	"github.com/stretchr/testify/mock"
)

type MockDocumentTypePromptRepository struct {
	mock.Mock
}

func NewMockDocumentTypePromptRepository(t interface {
	mock.TestingT
	Cleanup(func())
}) *MockDocumentTypePromptRepository {
	m := &MockDocumentTypePromptRepository{}
	m.Mock.Test(t)
	t.Cleanup(func() { m.AssertExpectations(t) })
	return m
}

func (m *MockDocumentTypePromptRepository) FindByDocumentType(documentType string) (*models.DocumentTypePrompt, error) {
	args := m.Called(documentType)
	var prompt *models.DocumentTypePrompt
	if args.Get(0) != nil {
		prompt = args.Get(0).(*models.DocumentTypePrompt)
	}
	return prompt, args.Error(1)
}

func (m *MockDocumentTypePromptRepository) Save(prompt *models.DocumentTypePrompt) error {
	args := m.Called(prompt)
	return args.Error(0)
}

func (m *MockDocumentTypePromptRepository) Update(prompt *models.DocumentTypePrompt) error {
	args := m.Called(prompt)
	return args.Error(0)
}

func (m *MockDocumentTypePromptRepository) Delete(id int64) error {
	args := m.Called(id)
	return args.Error(0)
}

func (m *MockDocumentTypePromptRepository) FindAll() ([]*models.DocumentTypePrompt, error) {
	args := m.Called()
	var prompts []*models.DocumentTypePrompt
	if args.Get(0) != nil {
		prompts = args.Get(0).([]*models.DocumentTypePrompt)
	}
	return prompts, args.Error(1)
}
