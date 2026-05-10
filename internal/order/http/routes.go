package http

import (
	cartRepo "ecom/internal/cart/repository"
	"ecom/internal/order/repository"
	"ecom/internal/order/service"
	userRepo "ecom/internal/user/repository"
	"ecom/pkg/config"
	"ecom/pkg/dbs"
	"ecom/pkg/jwt"
	"ecom/pkg/middleware"

	"github.com/gin-gonic/gin"
	"github.com/quangdangfit/gocommon/validation"
)

func Routes(r *gin.RouterGroup, db *dbs.Database, validator validation.Validation) {
	orderRepo := repository.NewOrderRepository(db)
	cr := cartRepo.NewCartRepository(db)
	ur := userRepo.NewUserRepository(db)
	cfg := config.GetEnv()
	orderSvc := service.NewOrderService(orderRepo, cr, ur, cfg)
	orderCtrl := NewOrderController(orderSvc, validator)

	authMiddleware := middleware.JWT(jwt.AccessTokenType, db)
	adminMiddleware := middleware.RequireAdmin()

	// Guest checkout — no auth required
	r.POST("/orders/guest", orderCtrl.GuestCheckout)

	// Customer routes (auth required)
	orderRoute := r.Group("/orders")
	orderRoute.Use(authMiddleware)
	{
		orderRoute.POST("", orderCtrl.CreateOrder)
		orderRoute.POST("/:id/verify", orderCtrl.VerifyPayment)
		orderRoute.GET("", orderCtrl.GetMyOrders)
		orderRoute.GET("/:id", orderCtrl.GetOrder)
	}

	// Admin routes
	adminRoute := r.Group("/admin")
	adminRoute.Use(authMiddleware, adminMiddleware)
	{
		adminRoute.GET("/orders", orderCtrl.GetAllOrders)
		adminRoute.GET("/delivery-requests", orderCtrl.GetDeliveryRequests)
		adminRoute.POST("/delivery-requests/:id/approve", orderCtrl.ApproveDelivery)
		adminRoute.POST("/delivery-requests/:id/reject", orderCtrl.RejectDelivery)
	}
}
