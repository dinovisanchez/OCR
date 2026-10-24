package mocks_services

import (
	"encoding/json"

	"ia-extraction/models"

	"github.com/stretchr/testify/mock"
)

type MockExtractionService struct {
	mock.Mock
}

func NewMockExtractionService(t interface {
	mock.TestingT
	Cleanup(func())
}) *MockExtractionService {
	m := &MockExtractionService{}
	m.Mock.Test(t)
	t.Cleanup(func() { m.AssertExpectations(t) })
	return m
}

func (m *MockExtractionService) ExtractAndSave(link string, documentType string, mediaType string) (json.RawMessage, *models.TokenUsage, error) {
	args := m.Called(link, documentType, mediaType)
	var raw json.RawMessage
	if args.Get(0) != nil {
		raw = args.Get(0).(json.RawMessage)
	}
	var usage *models.TokenUsage
	if args.Get(1) != nil {
		usage = args.Get(1).(*models.TokenUsage)
	}
	return raw, usage, args.Error(2)
}

func (m *MockExtractionService) ExtractZipAndSave(link string, documentType string) ([]models.ZipEntryResult, *models.TokenUsage, error) {
	args := m.Called(link, documentType)
	var results []models.ZipEntryResult
	if args.Get(0) != nil {
		results = args.Get(0).([]models.ZipEntryResult)
	}
	var usage *models.TokenUsage
	if args.Get(1) != nil {
		usage = args.Get(1).(*models.TokenUsage)
	}
	return results, usage, args.Error(2)
}
