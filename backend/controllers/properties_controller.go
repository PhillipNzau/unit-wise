package controllers

import (
	"context"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"

	"github.com/phillip/backend/config"
	"github.com/phillip/backend/models"
	"github.com/phillip/backend/utils"
)

// Create Property
func CreateProperty(cfg *config.Config) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Authenticated user
		uid := c.GetString("user_id")
		userID, err := primitive.ObjectIDFromHex(uid)
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid user id"})
			return
		}

		// Bind form data
		var input struct {
			Title       string  `form:"title" binding:"required"`
			Description string  `form:"description"`
			Location    string  `form:"location" binding:"required"`
			Price       float64 `form:"price" binding:"required"`
		}

		if err := c.ShouldBind(&input); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		// Handle multiple images
		form, err := c.MultipartForm()
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid form data"})
			return
		}

		files := form.File["images"] // key must be "images" in Postman
		var imageURLs []string

		for _, fileHeader := range files {
			file, err := fileHeader.Open()
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to open file"})
				return
			}

			url, err := utils.UploadToCloudinary(file, fileHeader)
			file.Close() // ✅ close immediately after upload
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{
					"error":   "Image upload failed",
					"details": err.Error(),
					"file":    fileHeader.Filename,
				})
				return
			}

			imageURLs = append(imageURLs, url)
		}

		// Save property
		property := models.Property{
			ID:          primitive.NewObjectID(),
			UserID:      userID,
			Title:       input.Title,
			Description: input.Description,
			Location:    input.Location,
			Price:       input.Price,
			Images:      imageURLs,
			CreatedAt:   time.Now(),
			UpdatedAt:   time.Now(),
		}

		col := cfg.MongoClient.Database(cfg.DBName).Collection("properties")
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		if _, err := col.InsertOne(ctx, property); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Could not create property"})
			return
		}

		c.JSON(http.StatusCreated, property)
	}
}


// List all Properties
func ListProperties(cfg *config.Config) gin.HandlerFunc {
	return func(c *gin.Context) {
		col := cfg.MongoClient.Database(cfg.DBName).Collection("properties")
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		cursor, err := col.Find(ctx, bson.M{})
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Could not fetch properties"})
			return
		}

		var properties []models.Property
		if err := cursor.All(ctx, &properties); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Could not decode properties"})
			return
		}

		c.JSON(http.StatusOK, properties)
	}
}

// Get single Property
func GetProperty(cfg *config.Config) gin.HandlerFunc {
	return func(c *gin.Context) {
		id := c.Param("id")
		objID, err := primitive.ObjectIDFromHex(id)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid ID"})
			return
		}

		col := cfg.MongoClient.Database(cfg.DBName).Collection("properties")
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		var property models.Property
		if err := col.FindOne(ctx, bson.M{"_id": objID}).Decode(&property); err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "Property not found"})
			return
		}

		c.JSON(http.StatusOK, property)
	}
}

// Update Property
func UpdateProperty(cfg *config.Config) gin.HandlerFunc {
	return func(c *gin.Context) {
		role, _ := c.Get("role")
		requesterID, _ := c.Get("userID")

		id := c.Param("id")
		objID, err := primitive.ObjectIDFromHex(id)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid ID"})
			return
		}

		var input struct {
			Title       string  `json:"title,omitempty"`
			Description string  `json:"description,omitempty"`
			Location    string  `json:"location,omitempty"`
			Price       float64 `json:"price,omitempty"`
			Images      []string `json:"images,omitempty"`
		}
		if err := c.ShouldBindJSON(&input); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		// Check ownership
		col := cfg.MongoClient.Database(cfg.DBName).Collection("properties")
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		var existing models.Property
		if err := col.FindOne(ctx, bson.M{"_id": objID}).Decode(&existing); err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "Property not found"})
			return
		}

		if role != "admin" && existing.UserID.Hex() != requesterID.(string) {
			c.JSON(http.StatusForbidden, gin.H{"error": "Access denied"})
			return
		}

		update := bson.M{
			"title":       input.Title,
			"description": input.Description,
			"location":    input.Location,
			"price":       input.Price,
			"images":      input.Images,
			"updated_at":  time.Now(),
		}

		_, err = col.UpdateOne(ctx, bson.M{"_id": objID}, bson.M{"$set": update})
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Could not update property"})
			return
		}

		c.JSON(http.StatusOK, gin.H{"message": "Property updated successfully"})
	}
}

// Delete Property
func DeleteProperty(cfg *config.Config) gin.HandlerFunc {
	return func(c *gin.Context) {
		role, _ := c.Get("role")
		requesterID, _ := c.Get("userID")

		id := c.Param("id")
		objID, err := primitive.ObjectIDFromHex(id)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid ID"})
			return
		}

		col := cfg.MongoClient.Database(cfg.DBName).Collection("properties")
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		var existing models.Property
		if err := col.FindOne(ctx, bson.M{"_id": objID}).Decode(&existing); err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "Property not found"})
			return
		}

		if role != "admin" && existing.UserID.Hex() != requesterID.(string) {
			c.JSON(http.StatusForbidden, gin.H{"error": "Access denied"})
			return
		}

		_, err = col.DeleteOne(ctx, bson.M{"_id": objID})
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Could not delete property"})
			return
		}

		c.JSON(http.StatusOK, gin.H{"message": "Property deleted successfully"})
	}
}
