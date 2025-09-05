package controllers

import (
	"context"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"

	"github.com/phillip/backend/config"
	"github.com/phillip/backend/models"
	"github.com/phillip/backend/utils"
)

// Create Housekeeper Report
func CreateHousekeeperReport(cfg *config.Config) gin.HandlerFunc {
	return func(c *gin.Context) {
		uid := c.GetString("user_id")
		userID, err := primitive.ObjectIDFromHex(uid)
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid user id"})
			return
		}

		// Bind form data
		var input struct {
			PropertyID string `form:"property_id" binding:"required"`
			Notes      string `form:"notes"`
		}
		if err := c.ShouldBind(&input); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		propertyID, err := primitive.ObjectIDFromHex(input.PropertyID)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid property ID"})
			return
		}

		// Handle damage images
		var imageURLs []string
		form, _ := c.MultipartForm()
		if form != nil {
			files := form.File["damage_images"]
			for _, fileHeader := range files {
				file, err := fileHeader.Open()
				if err != nil {
					c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to open file"})
					return
				}

				url, err := utils.UploadDamagesToCloudinary(file, fileHeader)
				file.Close()
				if err != nil {
					c.JSON(http.StatusInternalServerError, gin.H{
						"error":   "Image upload failed",
						"details": err.Error(),
					})
					return
				}

				imageURLs = append(imageURLs, url)
			}
		}

		report := models.HousekeeperReport{
			ID:            primitive.NewObjectID(),
			PropertyID:    propertyID,
			HousekeeperID: userID,
			Notes:         input.Notes,
			DamageImages:  imageURLs,
			CreatedAt:     time.Now(),
			UpdatedAt:     time.Now(),
		}

		col := cfg.MongoClient.Database(cfg.DBName).Collection("housekeeper_reports")
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		if _, err := col.InsertOne(ctx, report); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Could not create report"})
			return
		}

		c.JSON(http.StatusCreated, report)
	}
}

// List all Housekeeper Reports with property details
func ListHousekeeperReports(cfg *config.Config) gin.HandlerFunc {
	return func(c *gin.Context) {
		col := cfg.MongoClient.Database(cfg.DBName).Collection("housekeeper_reports")
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		pipeline := mongo.Pipeline{
			{{Key: "$lookup", Value: bson.M{
				"from":         "properties",
				"localField":   "property_id",
				"foreignField": "_id",
				"as":           "property",
			}}},
			{{Key: "$unwind", Value: bson.M{
				"path":                       "$property",
				"preserveNullAndEmptyArrays": true,
			}}},
		}

		cursor, err := col.Aggregate(ctx, pipeline)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Could not fetch reports"})
			return
		}

		var reports []bson.M
		if err := cursor.All(ctx, &reports); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Could not decode reports"})
			return
		}

		c.JSON(http.StatusOK, reports)
	}
}

// Get single Housekeeper Report with property details
func GetHousekeeperReport(cfg *config.Config) gin.HandlerFunc {
	return func(c *gin.Context) {
		id := c.Param("id")
		objID, err := primitive.ObjectIDFromHex(id)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid ID"})
			return
		}

		col := cfg.MongoClient.Database(cfg.DBName).Collection("housekeeper_reports")
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		pipeline := mongo.Pipeline{
			{{Key: "$match", Value: bson.M{"_id": objID}}},
			{{Key: "$lookup", Value: bson.M{
				"from":         "properties",
				"localField":   "property_id",
				"foreignField": "_id",
				"as":           "property",
			}}},
			{{Key: "$unwind", Value: bson.M{
				"path":                       "$property",
				"preserveNullAndEmptyArrays": true,
			}}},
		}

		cursor, err := col.Aggregate(ctx, pipeline)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Could not fetch report"})
			return
		}

		var reports []bson.M
		if err := cursor.All(ctx, &reports); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Could not decode report"})
			return
		}

		if len(reports) == 0 {
			c.JSON(http.StatusNotFound, gin.H{"error": "Report not found"})
			return
		}

		c.JSON(http.StatusOK, reports[0])
	}
}


// Update Housekeeper Report
func UpdateHousekeeperReport(cfg *config.Config) gin.HandlerFunc {
	return func(c *gin.Context) {
		// ✅ Extract and validate user ID and role
		role := c.GetString("role")
		requesterID := c.GetString("user_id")
		if requesterID == "" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
			return
		}

		// ✅ Validate report ID
		id := c.Param("id")
		reportID, err := primitive.ObjectIDFromHex(id)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid report ID"})
			return
		}

		// ✅ Get existing report
		col := cfg.MongoClient.Database(cfg.DBName).Collection("housekeeper_reports")
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		var existing models.HousekeeperReport
		if err := col.FindOne(ctx, bson.M{"_id": reportID}).Decode(&existing); err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "report not found"})
			return
		}

		// ✅ Ownership enforcement
		if role != "admin" && existing.HousekeeperID.Hex() != requesterID {
			c.JSON(http.StatusForbidden, gin.H{"error": "access denied"})
			return
		}

		// ✅ Bind form input
		var input struct {
			Notes        string   `form:"notes"`
			DamageImages []string `form:"damage_images"` // existing images to keep
		}
		if err := c.ShouldBind(&input); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		// ✅ Prepare update document
		update := bson.M{"updated_at": time.Now()}
		if input.Notes != "" {
			update["notes"] = input.Notes
		}

		// ✅ Process new image uploads
		newImageURLs := []string{}
		form, _ := c.MultipartForm()
		if form != nil {
			if files := form.File["new_damage_images"]; len(files) > 0 {
				for _, fileHeader := range files {
					file, err := fileHeader.Open()
					if err != nil {
						c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to open file"})
						return
					}
					url, err := utils.UploadDamagesToCloudinary(file, fileHeader)
					file.Close()
					if err != nil {
						c.JSON(http.StatusInternalServerError, gin.H{"error": "image upload failed", "details": err.Error()})
						return
					}
					newImageURLs = append(newImageURLs, url)
				}
			}
		}

		// ✅ Combine existing and new images if any
		if len(input.DamageImages) > 0 || len(newImageURLs) > 0 {
			update["damage_images"] = append(input.DamageImages, newImageURLs...)
		}

		// ❗ Make sure at least one field (other than updated_at) is being updated
		if len(update) == 1 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "no fields to update"})
			return
		}

		// ✅ Perform update
		_, err = col.UpdateOne(ctx, bson.M{"_id": reportID}, bson.M{"$set": update})
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "could not update report"})
			return
		}

		c.JSON(http.StatusOK, gin.H{"message": "report updated successfully"})
	}
}


// Delete Housekeeper Report
func DeleteHousekeeperReport(cfg *config.Config) gin.HandlerFunc {
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

		// ✅ Extract and validate report ID
		id := c.Param("id")
		objID, err := primitive.ObjectIDFromHex(id)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid report id"})
			return
		}

		role := c.GetString("role")
		col := cfg.MongoClient.Database(cfg.DBName).Collection("housekeeper_reports")
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		// ✅ Fetch report to check ownership
		var existing models.HousekeeperReport
		if err := col.FindOne(ctx, bson.M{"_id": objID}).Decode(&existing); err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "report not found"})
			return
		}

		// ✅ Enforce permissions
		if role != "admin" && existing.HousekeeperID != userID {
			c.JSON(http.StatusForbidden, gin.H{"error": "access denied"})
			return
		}

		// ✅ Delete the report
		res, err := col.DeleteOne(ctx, bson.M{"_id": objID})
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to delete report"})
			return
		}
		if res.DeletedCount == 0 {
			c.JSON(http.StatusNotFound, gin.H{"error": "report not found or already deleted"})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"message": "report deleted",
			"id":      objID.Hex(),
		})
	}
}

