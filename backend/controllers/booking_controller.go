package controllers

import (
	"context"
	"net/http"
	"time"

	"github.com/phillip/backend/config"
	"github.com/phillip/backend/models"

	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// CreateBooking - creates a new booking and updates property availability if confirmed
func CreateBooking(cfg *config.Config) gin.HandlerFunc {
	return func(c *gin.Context) {
		var input models.Booking
		if err := c.ShouldBindJSON(&input); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		input.ID = primitive.NewObjectID()
		input.CreatedAt = time.Now()
		input.UpdatedAt = time.Now()
		if input.Status == "" {
			input.Status = "pending"
		}

		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		bookingCol := cfg.MongoClient.Database(cfg.DBName).Collection("bookings")
		propertyCol := cfg.MongoClient.Database(cfg.DBName).Collection("properties")

		_, err := bookingCol.InsertOne(ctx, input)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Could not create booking"})
			return
		}

		// Update property availability if confirmed
		if input.Status == "confirmed" {
			_, err = propertyCol.UpdateOne(ctx, bson.M{"_id": input.PropertyID},
				bson.M{"$set": bson.M{"availability": false}})
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update property availability"})
				return
			}
		}

		c.JSON(http.StatusCreated, input)


	}
}

// GetBooking - fetch a booking by ID
func GetBooking(cfg *config.Config) gin.HandlerFunc {
	return func(c *gin.Context) {
		id := c.Param("id")
		objID, err := primitive.ObjectIDFromHex(id)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid booking ID"})
			return
		}

		var booking models.Booking
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		err = cfg.MongoClient.Database(cfg.DBName).Collection("bookings").FindOne(ctx, bson.M{"_id": objID}).Decode(&booking)
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "Booking not found"})
			return
		}

		c.JSON(http.StatusOK, booking)
	}
}

// ListBookings - get all bookings
func ListBookings(cfg *config.Config) gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		cursor, err := cfg.MongoClient.Database(cfg.DBName).Collection("bookings").Find(ctx, bson.M{})
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch bookings"})
			return
		}
		defer cursor.Close(ctx)

		var bookings []models.Booking
		if err = cursor.All(ctx, &bookings); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Error decoding bookings"})
			return
		}

		c.JSON(http.StatusOK, bookings)
	}
}

// UpdateBooking - update booking details or status, also updates property availability
func UpdateBooking(cfg *config.Config) gin.HandlerFunc {
	return func(c *gin.Context) {
		id := c.Param("id")
		objID, err := primitive.ObjectIDFromHex(id)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid booking ID"})
			return
		}

		var input models.Booking
		if err := c.ShouldBindJSON(&input); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		input.UpdatedAt = time.Now()

		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		bookingCol := cfg.MongoClient.Database(cfg.DBName).Collection("bookings")
		propertyCol := cfg.MongoClient.Database(cfg.DBName).Collection("properties")

		var existing models.Booking
		if err := bookingCol.FindOne(ctx, bson.M{"_id": objID}).Decode(&existing); err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "Booking not found"})
			return
		}

		// Update booking
		update := bson.M{
			"start_date": input.StartDate,
			"end_date":   input.EndDate,
			"status":     input.Status,
			"updated_at": input.UpdatedAt,
		}

		_, err = bookingCol.UpdateOne(ctx, bson.M{"_id": objID}, bson.M{"$set": update})
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update booking"})
			return
		}

		// Update property availability based on booking status
		switch input.Status {
			case "confirmed":
				_, _ = propertyCol.UpdateOne(ctx, bson.M{"_id": existing.PropertyID},
					bson.M{"$set": bson.M{"available": false}})
			case "cancelled":
				_, _ = propertyCol.UpdateOne(ctx, bson.M{"_id": existing.PropertyID},
					bson.M{"$set": bson.M{"available": true}})
		}

		c.JSON(http.StatusOK, gin.H{"message": "Booking updated successfully"})
	}
}

// DeleteBooking - deletes a booking and resets property availability if necessary
func DeleteBooking(cfg *config.Config) gin.HandlerFunc {
	return func(c *gin.Context) {
		id := c.Param("id")
		objID, err := primitive.ObjectIDFromHex(id)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid booking ID"})
			return
		}

		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		bookingCol := cfg.MongoClient.Database(cfg.DBName).Collection("bookings")
		propertyCol := cfg.MongoClient.Database(cfg.DBName).Collection("properties")

		var existing models.Booking
		if err := bookingCol.FindOne(ctx, bson.M{"_id": objID}).Decode(&existing); err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "Booking not found"})
			return
		}

		_, err = bookingCol.DeleteOne(ctx, bson.M{"_id": objID})
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete booking"})
			return
		}

		// Free up property if booking was confirmed
		if existing.Status == "confirmed" {
			_, _ = propertyCol.UpdateOne(ctx, bson.M{"_id": existing.PropertyID},
				bson.M{"$set": bson.M{"availability": true}})
		}

		c.JSON(http.StatusOK, gin.H{"message": "Booking deleted successfully"})
	}
}
