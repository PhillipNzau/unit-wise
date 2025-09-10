package models

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

type Booking struct {
	ID         primitive.ObjectID `bson:"_id,omitempty" json:"id"`
	UserID     primitive.ObjectID `bson:"user_id" json:"user_id"`
	PropertyID primitive.ObjectID `bson:"property_id" json:"property_id"`
	StartDate  time.Time          `bson:"start_date" json:"start_date"`
	EndDate    time.Time          `bson:"end_date" json:"end_date"`
	Status     string             `bson:"status" json:"status"` // pending, confirmed, cancelled, completed
	CreatedAt  time.Time          `bson:"created_at" json:"created_at"`
	UpdatedAt  time.Time          `bson:"updated_at" json:"updated_at"`
}
