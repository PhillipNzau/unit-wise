package routes

import (
	"github.com/gin-gonic/gin"
	"github.com/phillip/backend/config"
	"github.com/phillip/backend/controllers"
	"github.com/phillip/backend/middleware"
)

func SetupRoutes(r *gin.Engine, cfg *config.Config) {
	// public
	r.POST("/auth/register", controllers.Register(cfg))
	r.POST("/auth/login", controllers.Login(cfg))
	r.POST("/auth/refresh", controllers.RefreshToken(cfg))

	// otp
	r.POST("/auth/request-otp", controllers.RequestOTP(cfg))
	r.POST("/auth/verify-otp", controllers.VerifyOTP(cfg))

	// protected
	auth := middleware.AuthMiddleware(cfg)
	
	creds := r.Group("/credentials")
	creds.Use(auth)
	{
		creds.POST("", controllers.CreateCredential(cfg))
		creds.GET("", controllers.ListCredentials(cfg))
		creds.GET(":id", controllers.GetCredential(cfg))
		creds.PUT(":id", controllers.UpdateCredential(cfg))
		creds.DELETE(":id", controllers.DeleteCredential(cfg))
	}

	users := r.Group("/users")
	users.Use(auth)
	{
		// users.POST("", controllers.ListUsers(cfg))
		users.GET("", controllers.ListUsers(cfg))
		users.GET(":id", controllers.GetUser(cfg))
		users.PATCH(":id", controllers.UpdateUser(cfg))
		users.DELETE(":id", controllers.DeleteUser(cfg))
	}

	props := r.Group("/properties")
	props.Use(auth) // ensure user is logged in
	{
		props.POST("", controllers.CreateProperty(cfg))
		props.GET("", controllers.ListProperties(cfg))
		props.GET("/:id", controllers.GetProperty(cfg))
		props.PATCH("/:id", controllers.UpdateProperty(cfg))
		props.DELETE("/:id", controllers.DeleteProperty(cfg))
	}

}
