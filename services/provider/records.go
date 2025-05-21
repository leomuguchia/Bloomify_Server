package provider

import (
	"bloomify/models"
	"fmt"

	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/bson"
)

func (s *DefaultProviderService) GetHistoricalRecords(c *gin.Context, providerID string) ([]models.HistoricalRecord, error) {
	ctx := c.Request.Context()
	return s.RecordsRepo.GetByProviderID(ctx, providerID)
}

func (s *DefaultProviderService) AddHistoricalRecord(c *gin.Context, record models.HistoricalRecord) (string, error) {
	ctx := c.Request.Context()

	// Step 1: Save the historical record
	recordID, err := s.RecordsRepo.Create(ctx, record)
	if err != nil {
		return "", fmt.Errorf("failed to save historical record: %w", err)
	}

	// Step 2: Push the recordID into provider.HistoricalRecordsIDs
	update := bson.M{
		"$push": bson.M{
			"historicalRecordsIds": recordID,
		},
	}
	if err := s.Repo.UpdateWithDocument(record.ProviderID, update); err != nil {
		return "", fmt.Errorf("failed to push record ID into provider: %w", err)
	}

	return recordID, nil
}

func (s *DefaultProviderService) DeleteHistoricalRecord(c *gin.Context, recordID string) error {
	ctx := c.Request.Context()
	return s.RecordsRepo.DeleteByID(ctx, recordID)
}
