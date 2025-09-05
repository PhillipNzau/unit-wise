package models

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

type HousekeeperReport struct {
	ID            primitive.ObjectID `bson:"_id,omitempty" json:"id"`
	PropertyID    primitive.ObjectID `bson:"property_id" json:"property_id"`
	HousekeeperID primitive.ObjectID `bson:"housekeeper_id" json:"housekeeper_id"`
	Notes         string             `bson:"notes" json:"notes"`
	DamageImages  []string           `bson:"damage_images" json:"damage_images"`
	CreatedAt     time.Time          `bson:"created_at" json:"created_at"`
	UpdatedAt     time.Time          `bson:"updated_at" json:"updated_at"`
}
