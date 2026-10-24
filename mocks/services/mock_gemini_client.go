package mocks_services

import (
	"ia-extraction/models"

	"github.com/stretchr/testify/mock"
)

type MockGeminiClient struct {
	mock.Mock
}

func NewMockGeminiClient(t interface {
	mock.TestingT
	Cleanup(func())
}) *MockGeminiClient {
	m := &MockGeminiClient{}
	m.Mock.Test(t)
	t.Cleanup(func() { m.AssertExpectations(t) })
	return m
}

func (m *MockGeminiClient) Call(documentType string, prompt string, images []models.GeminiInlineImage, fields []models.FieldSpec) (string, *models.TokenUsage, error) {
	args := m.Called(documentType, prompt, images, fields)
	var usage *models.TokenUsage
	if args.Get(1) != nil {
		usage = args.Get(1).(*models.TokenUsage)
	}
	return args.String(0), usage, args.Error(2)
}
