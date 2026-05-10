package repository

import (
	"context"

	"ecom/internal/order/entity"
	"ecom/pkg/dbs"
)

type OrderRepo struct {
	db *dbs.Database
}

func NewOrderRepository(db *dbs.Database) *OrderRepo {
	return &OrderRepo{db: db}
}

func (r *OrderRepo) Create(ctx context.Context, order *entity.Order) error {
	return r.db.GetDB().WithContext(ctx).Create(order).Error
}

func (r *OrderRepo) GetByID(ctx context.Context, id string) (*entity.Order, error) {
	var order entity.Order
	err := r.db.GetDB().WithContext(ctx).
		Preload("Items").
		Preload("ShippingAddress").
		Preload("DeliveryRequest").
		Where("id = ?", id).
		First(&order).Error
	if err != nil {
		return nil, err
	}
	return &order, nil
}

func (r *OrderRepo) GetByUserID(ctx context.Context, userID string) ([]entity.Order, error) {
	var orders []entity.Order
	err := r.db.GetDB().WithContext(ctx).
		Preload("Items").
		Preload("ShippingAddress").
		Preload("DeliveryRequest").
		Where("user_id = ?", userID).
		Order("created_at DESC").
		Find(&orders).Error
	return orders, err
}

func (r *OrderRepo) GetAll(ctx context.Context) ([]entity.Order, error) {
	var orders []entity.Order
	err := r.db.GetDB().WithContext(ctx).
		Preload("Items").
		Preload("ShippingAddress").
		Preload("DeliveryRequest").
		Order("created_at DESC").
		Find(&orders).Error
	return orders, err
}

func (r *OrderRepo) Update(ctx context.Context, order *entity.Order) error {
	return r.db.GetDB().WithContext(ctx).
		Omit("ShippingAddress", "Items", "DeliveryRequest").
		Save(order).Error
}

func (r *OrderRepo) CreateDeliveryRequest(ctx context.Context, req *entity.DeliveryRequest) error {
	return r.db.GetDB().WithContext(ctx).Create(req).Error
}

func (r *OrderRepo) GetDeliveryRequests(ctx context.Context, status string) ([]entity.DeliveryRequest, error) {
	var requests []entity.DeliveryRequest
	query := r.db.GetDB().WithContext(ctx).
		Preload("Order").
		Preload("Order.Items").
		Preload("Order.ShippingAddress")

	if status != "" {
		query = query.Where("delivery_requests.status = ?", status)
	}

	err := query.Order("delivery_requests.created_at DESC").Find(&requests).Error
	return requests, err
}

func (r *OrderRepo) GetDeliveryRequestByID(ctx context.Context, id string) (*entity.DeliveryRequest, error) {
	var req entity.DeliveryRequest
	err := r.db.GetDB().WithContext(ctx).
		Preload("Order").
		Preload("Order.Items").
		Preload("Order.ShippingAddress").
		Where("id = ?", id).
		First(&req).Error
	if err != nil {
		return nil, err
	}
	return &req, nil
}

func (r *OrderRepo) UpdateDeliveryRequest(ctx context.Context, req *entity.DeliveryRequest) error {
	return r.db.GetDB().WithContext(ctx).Save(req).Error
}
