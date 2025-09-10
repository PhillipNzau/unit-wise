package controllers

import (
	"context"
	"net/http"
	"time"

	"github.com/phillip/backend/config"
	"github.com/phillip/backend/models"
	"github.com/phillip/backend/utils"

	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// CreateBooking - creates a new booking and updates property availability if confirmed
func CreateBooking(cfg *config.Config) gin.HandlerFunc {
	return func(c *gin.Context) {
		// ✅ Extract logged-in user from context (set by auth middleware)
		uid := c.GetString("user_id")
		userID, err := primitive.ObjectIDFromHex(uid)
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid user id"})
			return
		}

		// ✅ Bind input payload
		var input struct {
			PropertyID string    `json:"property_id" binding:"required"`
			StartDate  time.Time `json:"start_date" binding:"required"`
			EndDate    time.Time `json:"end_date" binding:"required"`
			Status     string    `json:"status"` // optional
		}
		if err := c.ShouldBindJSON(&input); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		propertyID, err := primitive.ObjectIDFromHex(input.PropertyID)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid property ID"})
			return
		}

		// ✅ Build booking object
		booking := models.Booking{
			ID:         primitive.NewObjectID(),
			UserID:     userID, // <-- logged-in user
			PropertyID: propertyID,
			StartDate:  input.StartDate,
			EndDate:    input.EndDate,
			Status:     input.Status,
			CreatedAt:  time.Now(),
			UpdatedAt:  time.Now(),
		}
		if booking.Status == "" {
			booking.Status = "pending"
		}

		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		bookingCol := cfg.MongoClient.Database(cfg.DBName).Collection("bookings")
		propertyCol := cfg.MongoClient.Database(cfg.DBName).Collection("properties")

		// ✅ Insert booking
		if _, err := bookingCol.InsertOne(ctx, booking); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Could not create booking"})
			return
		}

		// ✅ Update property availability if confirmed
		if booking.Status == "confirmed" {
			_, err = propertyCol.UpdateOne(ctx, bson.M{"_id": booking.PropertyID},
				bson.M{"$set": bson.M{"availability": false}})
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update property availability"})
				return
			}
		}

		// ✅ Send notifications to property owner + housekeepers
		var property models.Property
		if err := propertyCol.FindOne(ctx, bson.M{"_id": booking.PropertyID}).Decode(&property); err == nil {
			recipients := append([]primitive.ObjectID{property.UserID}, property.Housekeepers...)

			switch booking.Status {
			case "confirmed":
				_ = utils.CreateNotification(cfg, recipients, "Booking Confirmed", "A booking has been confirmed for your property.")
			case "cancelled":
				_ = utils.CreateNotification(cfg, recipients, "Booking Cancelled", "A booking has been cancelled for your property.")
			}
		}

		c.JSON(http.StatusCreated, booking)
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
		// ✅ Extract user from context
		uid := c.GetString("user_id")
		userID, err := primitive.ObjectIDFromHex(uid)
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid user id"})
			return
		}

		// ✅ Get booking ID from params
		bookingID := c.Param("id")
		objID, err := primitive.ObjectIDFromHex(bookingID)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid booking id"})
			return
		}

		// ✅ Bind payload with pointers so optional fields don’t overwrite
		var input struct {
			Status    string     `json:"status"`
			StartDate *time.Time `json:"start_date,omitempty"`
			EndDate   *time.Time `json:"end_date,omitempty"`
		}
		if err := c.ShouldBindJSON(&input); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		bookingCol := cfg.MongoClient.Database(cfg.DBName).Collection("bookings")
		propertyCol := cfg.MongoClient.Database(cfg.DBName).Collection("properties")

		// ✅ Fetch booking (to validate ownership + property info)
		var existing models.Booking
		if err := bookingCol.FindOne(ctx, bson.M{"_id": objID, "user_id": userID}).Decode(&existing); err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "Booking not found"})
			return
		}

		// ✅ Build update document dynamically
		updateFields := bson.M{
			"status":     input.Status,
			"updated_at": time.Now(),
		}
		if input.StartDate != nil {
			updateFields["start_date"] = *input.StartDate
		}
		if input.EndDate != nil {
			updateFields["end_date"] = *input.EndDate
		}

		_, err = bookingCol.UpdateOne(ctx, bson.M{"_id": objID}, bson.M{"$set": updateFields})
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Could not update booking"})
			return
		}

		// ✅ Update property availability if confirmed/cancelled
		if input.Status == "confirmed" {
			_, _ = propertyCol.UpdateOne(ctx, bson.M{"_id": existing.PropertyID},
				bson.M{"$set": bson.M{"availability": false}})
		}
		if input.Status == "cancelled" {
			_, _ = propertyCol.UpdateOne(ctx, bson.M{"_id": existing.PropertyID},
				bson.M{"$set": bson.M{"availability": true}})
		}

		// ✅ Notify property owner + housekeepers
		var property models.Property
		if err := propertyCol.FindOne(ctx, bson.M{"_id": existing.PropertyID}).Decode(&property); err == nil {
			recipients := append([]primitive.ObjectID{property.UserID}, property.Housekeepers...)

			switch input.Status {
			case "confirmed":
				_ = utils.CreateNotification(cfg, recipients, "Booking Confirmed", "A booking has been confirmed for your property.")
			case "cancelled":
				_ = utils.CreateNotification(cfg, recipients, "Booking Cancelled", "A booking has been cancelled for your property.")
			case "completed":
				_ = utils.CreateNotification(cfg, recipients, "Booking Completed", "A booking has been completed for your property.")
			}
		}

		// ✅ Return updated booking
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
