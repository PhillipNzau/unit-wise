package models

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

type Property struct {
	ID            primitive.ObjectID   `bson:"_id,omitempty" json:"id"`
	UserID        primitive.ObjectID   `bson:"user_id" json:"user_id"`
	Title         string               `bson:"title" json:"title"`
	Description   string               `bson:"description" json:"description"`
	Location      string               `bson:"location" json:"location"`
	Price         float64              `bson:"price" json:"price"`
	Images        []string             `bson:"images" json:"images"`
	Available  bool                 `bson:"availability" json:"availability"`
	Housekeepers  []primitive.ObjectID `bson:"housekeepers,omitempty" json:"housekeepers,omitempty"`
	CreatedAt     time.Time            `bson:"created_at" json:"created_at"`
	UpdatedAt     time.Time            `bson:"updated_at" json:"updated_at"`
}
