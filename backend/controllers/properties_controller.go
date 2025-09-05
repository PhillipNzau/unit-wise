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
			Available   *bool   `form:"available"` // pointer so it's optional
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
			Available:   input.Available == nil || *input.Available,
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
		// ✅ Validate requester identity
		role := c.GetString("role")
		requesterID := c.GetString("user_id")
		if requesterID == "" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
			return
		}

		// ✅ Validate property ID
		id := c.Param("id")
		objID, err := primitive.ObjectIDFromHex(id)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid property ID"})
			return
		}

		// ✅ Fetch existing property
		col := cfg.MongoClient.Database(cfg.DBName).Collection("properties")
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		var existing models.Property
		if err := col.FindOne(ctx, bson.M{"_id": objID}).Decode(&existing); err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "Property not found"})
			return
		}

		// ✅ Check permission
		if role != "admin" && existing.UserID.Hex() != requesterID {
			c.JSON(http.StatusForbidden, gin.H{"error": "Access denied"})
			return
		}

		// ✅ Bind input (form-data)
		var input struct {
			Title       string   `form:"title"`
			Description string   `form:"description"`
			Location    string   `form:"location"`
			Price       float64  `form:"price"`
			Available   *bool    `form:"available"`
			Images      []string `form:"images"` // existing image URLs to keep
		}

		if err := c.ShouldBind(&input); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		// ✅ Prepare update document
		update := bson.M{"updated_at": time.Now()}

		if input.Title != "" {
			update["title"] = input.Title
		}
		if input.Description != "" {
			update["description"] = input.Description
		}
		if input.Location != "" {
			update["location"] = input.Location
		}
		if input.Price > 0 {
			update["price"] = input.Price
		}
		if input.Available != nil {
			update["available"] = *input.Available
		}

		// ✅ Handle new image uploads (multipart form)
		newImageURLs := []string{}
		form, _ := c.MultipartForm()
		if form != nil {
			files := form.File["new_images"]
			for _, fileHeader := range files {
				file, err := fileHeader.Open()
				if err != nil {
					c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to open image"})
					return
				}
				url, err := utils.UploadToCloudinary(file, fileHeader)
				file.Close()
				if err != nil {
					c.JSON(http.StatusInternalServerError, gin.H{"error": "image upload failed", "details": err.Error()})
					return
				}
				newImageURLs = append(newImageURLs, url)
			}
		}

		// ✅ Merge existing and new images
		if input.Images != nil || len(newImageURLs) > 0 {
			update["images"] = append(input.Images, newImageURLs...)
		}

		// ❗ Reject empty update
		if len(update) == 1 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "no fields to update"})
			return
		}

		// ✅ Apply update
		_, err = col.UpdateOne(ctx, bson.M{"_id": objID}, bson.M{"$set": update})
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Could not update property"})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"message": "Property updated successfully",
			"images":  update["images"],
		})
	}
}



// Delete Property
func DeleteProperty(cfg *config.Config) gin.HandlerFunc {
	return func(c *gin.Context) {
		// ✅ Extract and validate user ID
		uid := c.GetString("user_id")
		if uid == "" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
			return
		}
		userID, err := primitive.ObjectIDFromHex(uid)
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid user id"})
			return
		}

		// ✅ Extract and validate property ID
		id := c.Param("id")
		objID, err := primitive.ObjectIDFromHex(id)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid property id"})
			return
		}

		role := c.GetString("role")
		col := cfg.MongoClient.Database(cfg.DBName).Collection("properties")
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		// ✅ Fetch property to check ownership
		var existing models.Property
		if err := col.FindOne(ctx, bson.M{"_id": objID}).Decode(&existing); err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "property not found"})
			return
		}

		// ✅ Enforce permissions
		if role != "admin" && existing.UserID != userID {
			c.JSON(http.StatusForbidden, gin.H{"error": "access denied"})
			return
		}

		// ✅ Delete the property
		res, err := col.DeleteOne(ctx, bson.M{"_id": objID})
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to delete property"})
			return
		}
		if res.DeletedCount == 0 {
			c.JSON(http.StatusNotFound, gin.H{"error": "property not found or already deleted"})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"message": "property deleted",
			"id":      objID.Hex(),
		})
	}
}
