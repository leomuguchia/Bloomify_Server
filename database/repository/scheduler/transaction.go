package schedulerRepo

import (
	"bloomify/models"
	"context"
	"fmt"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
)

func (repo *MongoSchedulerRepo) EmbedBookingReference(
	providerID, slotID, date, bookingID string,
	units int, priority bool,
) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	filter := bson.M{
		"id": providerID,
		"timeSlots": bson.M{
			"$elemMatch": bson.M{
				"id":      slotID,
				"date":    date,
				"blocked": false,
			},
		},
	}

	// Determine which field to increment
	incrementField := "bookedUnitsStandard"
	if priority {
		incrementField = "bookedUnitsPriority"
	}

	pipeline := mongo.Pipeline{
		bson.D{
			{
				Key: "$set", Value: bson.D{
					{
						Key: "timeSlots", Value: bson.D{
							{
								Key: "$map", Value: bson.D{
									{Key: "input", Value: "$timeSlots"},
									{Key: "as", Value: "ts"},
									{Key: "in", Value: bson.D{
										{
											Key: "$cond", Value: bson.D{
												{
													Key: "if", Value: bson.D{
														{
															Key: "$and", Value: bson.A{
																bson.D{{Key: "$eq", Value: bson.A{"$$ts.id", slotID}}},
																bson.D{{Key: "$eq", Value: bson.A{"$$ts.date", date}}},
																bson.D{{Key: "$eq", Value: bson.A{"$$ts.blocked", false}}},
																bson.D{{Key: "$gte", Value: bson.A{
																	bson.D{{Key: "$subtract", Value: bson.A{
																		"$$ts.capacity",
																		bson.D{{Key: "$add", Value: bson.A{"$$ts.bookedUnitsPriority", "$$ts.bookedUnitsStandard"}}},
																	}}},
																	units,
																}}},
															},
														},
													},
												},
												{
													Key: "then", Value: bson.D{
														{
															Key: "$mergeObjects", Value: bson.A{
																"$$ts",
																bson.D{
																	{
																		Key: "bookingIds", Value: bson.D{
																			{
																				Key: "$concatArrays", Value: bson.A{
																					"$$ts.bookingIds", bson.A{bookingID},
																				},
																			},
																		},
																	},
																	{
																		Key: incrementField, Value: bson.D{
																			{
																				Key: "$add", Value: bson.A{
																					"$$ts." + incrementField,
																					units,
																				},
																			},
																		},
																	},
																},
															},
														},
													},
												},
												{
													Key: "else", Value: "$$ts",
												},
											},
										},
									}},
								},
							},
						},
					},
				},
			},
		},
	}

	_, err := repo.providerColl.UpdateOne(ctx, filter, pipeline)
	if err != nil {
		return fmt.Errorf("failed to embed booking reference: %w", err)
	}

	return nil
}

func (repo *MongoSchedulerRepo) BookSingleSlotTransactionally(
	ctx context.Context,
	providerID string,
	date string,
	slot models.TimeSlot,
	booking *models.Booking,
) error {
	client := repo.providerColl.Database().Client()
	sess, err := client.StartSession()
	if err != nil {
		return fmt.Errorf("could not start mongo session: %w", err)
	}
	defer sess.EndSession(ctx)

	txnFn := func(sc mongo.SessionContext) error {
		if _, err := repo.bookingColl.InsertOne(sc, booking); err != nil {
			return fmt.Errorf("insert booking failed: %w", err)
		}

		// Simplified safe update logic
		filter := bson.M{
			"id": providerID,
			"timeSlots": bson.M{
				"$elemMatch": bson.M{
					"id":      slot.ID,
					"date":    date,
					"blocked": false,
				},
			},
		}

		incField := "timeSlots.$.bookedUnitsStandard"
		if booking.Priority {
			incField = "timeSlots.$.bookedUnitsPriority"
		}

		update := bson.M{
			"$addToSet": bson.M{"timeSlots.$.bookingIds": booking.ID},
			"$inc":      bson.M{incField: booking.Units},
		}

		res, err := repo.providerColl.UpdateOne(sc, filter, update)
		if err != nil {
			return fmt.Errorf("embed booking reference failed: %w", err)
		}
		if res.MatchedCount == 0 {
			return fmt.Errorf("no matching timeslot found or insufficient capacity")
		}

		return nil
	}

	if err := mongo.WithSession(ctx, sess, func(sc mongo.SessionContext) error {
		if err := sc.StartTransaction(); err != nil {
			return err
		}
		if err := txnFn(sc); err != nil {
			_ = sc.AbortTransaction(sc)
			return err
		}
		return sc.CommitTransaction(sc)
	}); err != nil {
		return fmt.Errorf("booking transaction failed: %w", err)
	}

	return nil
}

func (repo *MongoSchedulerRepo) SetEmbeddedTimeSlotBlocked(providerID, slotID, date string, blocked bool, reason string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	filter := bson.M{
		"id": providerID,
		"timeSlots": bson.M{
			"$elemMatch": bson.M{
				"id":   slotID,
				"date": date,
			},
		},
	}
	update := bson.M{
		"$set": bson.M{
			"timeSlots.$.blocked":     blocked,
			"timeSlots.$.blockReason": reason,
		},
	}
	_, err := repo.providerColl.UpdateOne(ctx, filter, update)
	if err != nil {
		return fmt.Errorf("failed to update blocked flag for timeslot: %w", err)
	}
	return nil
}
