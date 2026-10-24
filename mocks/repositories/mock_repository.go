package mocks_repositories

import (
	"ia-extraction/models"

	"github.com/stretchr/testify/mock"
)

type MockExtractionRepository struct {
	mock.Mock
}

func NewMockExtractionRepository(t interface {
	mock.TestingT
	Cleanup(func())
}) *MockExtractionRepository {
	m := &MockExtractionRepository{}
	m.Mock.Test(t)
	t.Cleanup(func() { m.AssertExpectations(t) })
	return m
}

func (m *MockExtractionRepository) Save(extraction *models.Extraction) error {
	args := m.Called(extraction)
	return args.Error(0)
}
