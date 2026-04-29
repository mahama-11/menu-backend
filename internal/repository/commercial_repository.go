package repository

import (
	"menu-service/internal/models"

	"gorm.io/gorm"
)

type CommercialRepository struct {
	db *gorm.DB
}

func NewCommercialRepository(db *gorm.DB) *CommercialRepository {
	return &CommercialRepository{db: db}
}

func (r *CommercialRepository) AutoMigrate() error {
	return r.db.AutoMigrate(&models.CommercialOrder{}, &models.CommercialPayment{}, &models.CommercialFulfillment{})
}

func (r *CommercialRepository) CreateOrder(item *models.CommercialOrder) error {
	if item.ID == "" {
		item.ID = buildMenuID("ord")
	}
	return r.db.Create(item).Error
}

func (r *CommercialRepository) SaveOrder(item *models.CommercialOrder) error {
	return r.db.Save(item).Error
}

func (r *CommercialRepository) FindOrderByID(orgID, orderID string) (*models.CommercialOrder, error) {
	var item models.CommercialOrder
	if err := r.db.Where("organization_id = ? AND id = ?", orgID, orderID).First(&item).Error; err != nil {
		return nil, err
	}
	return &item, nil
}

func (r *CommercialRepository) ListOrders(orgID string, limit, offset int) ([]models.CommercialOrder, error) {
	var items []models.CommercialOrder
	q := r.db.Where("organization_id = ?", orgID).Order("created_at desc")
	if limit > 0 {
		q = q.Limit(limit)
	}
	if offset > 0 {
		q = q.Offset(offset)
	}
	if err := q.Find(&items).Error; err != nil {
		return nil, err
	}
	return items, nil
}

func (r *CommercialRepository) CreatePayment(item *models.CommercialPayment) error {
	if item.ID == "" {
		item.ID = buildMenuID("pay")
	}
	return r.db.Create(item).Error
}

func (r *CommercialRepository) FindLatestPaymentByOrderID(orderID string) (*models.CommercialPayment, error) {
	var item models.CommercialPayment
	if err := r.db.Where("order_id = ?", orderID).Order("created_at desc").First(&item).Error; err != nil {
		return nil, err
	}
	return &item, nil
}

func (r *CommercialRepository) CreateFulfillment(item *models.CommercialFulfillment) error {
	if item.ID == "" {
		item.ID = buildMenuID("ful")
	}
	return r.db.Create(item).Error
}

func (r *CommercialRepository) FindLatestFulfillmentByOrderID(orderID string) (*models.CommercialFulfillment, error) {
	var item models.CommercialFulfillment
	if err := r.db.Where("order_id = ?", orderID).Order("created_at desc").First(&item).Error; err != nil {
		return nil, err
	}
	return &item, nil
}
