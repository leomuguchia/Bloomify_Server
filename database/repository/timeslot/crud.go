// File: database/repository/timeslot/crud.go
package timeslotRepo

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"

	"bloomify/models"
)

func (r *mongoTimeSlotRepo) CreateMany(ctx context.Context, slots []models.TimeSlot) ([]string, error) {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	docs := make([]interface{}, len(slots))
	for i, slot := range slots {
		if slot.ID == "" {
			slot.ID = uuid.New().String()
		}
		docs[i] = slot
	}

	res, err := r.coll.InsertMany(ctx, docs, &options.InsertManyOptions{Ordered: boolPtr(true)})
	if err != nil {
		return nil, err
	}

	ids := make([]string, len(res.InsertedIDs))
	for i, raw := range res.InsertedIDs {
		switch v := raw.(type) {
		case string:
			ids[i] = v
		case uuid.UUID:
			ids[i] = v.String()
		case primitive.ObjectID:
			ids[i] = v.Hex()
		default:
			return nil, errors.New("unexpected type for inserted ID")
		}
	}
	return ids, nil
}

func (r *mongoTimeSlotRepo) DeleteByID(ctx context.Context, providerID, slotID string) error {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	filter := bson.M{"id": slotID, "providerId": providerID}
	res, err := r.coll.DeleteOne(ctx, filter)
	if err != nil {
		return err
	}
	if res.DeletedCount == 0 {
		return mongo.ErrNoDocuments
	}
	return nil
}

func (r *mongoTimeSlotRepo) GetByProviderIDAndDate(ctx context.Context, providerID, date string) ([]models.TimeSlot, error) {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	filter := bson.M{"providerId": providerID, "date": date}
	cursor, err := r.coll.Find(ctx, filter)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var slots []models.TimeSlot
	if err := cursor.All(ctx, &slots); err != nil {
		return nil, err
	}
	return slots, nil
}

func (r *mongoTimeSlotRepo) GetByIDWithDate(ctx context.Context, providerID, slotID, date string) (*models.TimeSlot, error) {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	filter := bson.M{"providerId": providerID, "id": slotID, "date": date}
	var slot models.TimeSlot
	err := r.coll.FindOne(ctx, filter).Decode(&slot)
	if err != nil {
		return nil, err
	}
	return &slot, nil
}

func boolPtr(b bool) *bool { return &b }
