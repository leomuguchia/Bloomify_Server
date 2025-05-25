package provider

import (
	"bloomify/models"
	"context"
	"fmt"

	"go.mongodb.org/mongo-driver/bson"
)

func (s *DefaultProviderService) GetHistoricalRecords(ctx context.Context, providerID string) ([]models.HistoricalRecord, error) {
	return s.RecordsRepo.GetByProviderID(ctx, providerID)
}

func (s *DefaultProviderService) AddHistoricalRecord(c context.Context, record models.HistoricalRecord) (string, error) {
	// Step 1: Save the historical record
	recordID, err := s.RecordsRepo.Create(c, record)
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

func (s *DefaultProviderService) DeleteHistoricalRecord(c context.Context, recordID string) error {
	return s.RecordsRepo.DeleteByID(c, recordID)
}
