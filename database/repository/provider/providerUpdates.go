package providerRepo

import (
	"fmt"
	"time"

	"go.mongodb.org/mongo-driver/bson"
)

func (r *MongoProviderRepo) UpdateSet(id string, updateDoc bson.M) error {
	return r.updateWithOperator(id, "$set", updateDoc)
}

func (r *MongoProviderRepo) UpdatePush(id string, updateDoc bson.M) error {
	return r.updateWithOperator(id, "$push", updateDoc)
}

func (r *MongoProviderRepo) updateWithOperator(id, operator string, updateDoc bson.M) error {
	ctx, cancel := newContext(5 * time.Second)
	defer cancel()

	update := bson.M{operator: updateDoc}
	filter := bson.M{"id": id}

	result, err := r.coll.UpdateOne(ctx, filter, update)
	if err != nil {
		return fmt.Errorf("failed to update provider with id %s: %w", id, err)
	}
	if result.MatchedCount == 0 {
		return fmt.Errorf("provider with id %s not found", id)
	}
	return nil
}
