package main

import (
	cartEntity "ecom/internal/cart/entity"
	categoryEntity "ecom/internal/category/entity"
	orderEntity "ecom/internal/order/entity"
	productEntity "ecom/internal/product/entity"
	"ecom/internal/server/http"
	userEntity "ecom/internal/user/entity"
	"ecom/pkg/config"
	"ecom/pkg/dbs"

	"github.com/quangdangfit/gocommon/logger"
	"github.com/quangdangfit/gocommon/validation"
)

func main() {
	cfg := config.LoadConfig()
	logger.Initialize(cfg.Environment)

	db, err := dbs.NewDatabase(cfg.DatabaseURI)
	if err != nil {
		logger.Fatal("Cannot connect to database", err)
	}

	err = db.AutoMigrate(
		&userEntity.User{},
		&userEntity.OTP{},
		&categoryEntity.Category{},
		&productEntity.Product{},
		&productEntity.ProductVariant{},
		&productEntity.NutritionalInfo{},
		&cartEntity.Cart{},
		&cartEntity.CartItem{},
		&orderEntity.Order{},
		&orderEntity.OrderItem{},
		&orderEntity.ShippingAddress{},
		&orderEntity.DeliveryRequest{},
	)
	if err != nil {
		logger.Fatal("Failed to migrate")
	}
	validtor := validation.New()

	server := http.NewServer(validtor, db)
	if err = server.Run(); err != nil {
		logger.Fatal(err)
	}
}
