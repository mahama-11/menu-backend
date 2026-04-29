package models

import "time"

type CommercialOrder struct {
	ID                string     `gorm:"type:varchar(64);primaryKey" json:"id"`
	UserID            string     `gorm:"type:varchar(64);index:idx_commercial_order_org_created,priority:2;not null" json:"user_id"`
	OrganizationID    string     `gorm:"type:varchar(64);index:idx_commercial_order_org_created,priority:1;not null" json:"organization_id"`
	ProductCode       string     `gorm:"type:varchar(64);not null;default:menu" json:"product_code"`
	SKUCode           string     `gorm:"type:varchar(128);index;not null" json:"sku_code"`
	PackageCode       string     `gorm:"type:varchar(128);index;not null" json:"package_code"`
	PackageType       string     `gorm:"type:varchar(64);not null" json:"package_type"`
	Currency          string     `gorm:"type:varchar(16);not null" json:"currency"`
	Quantity          int64      `gorm:"not null;default:1" json:"quantity"`
	UnitAmount        int64      `gorm:"not null;default:0" json:"unit_amount"`
	TotalAmount       int64      `gorm:"not null;default:0" json:"total_amount"`
	Status            string     `gorm:"type:varchar(32);index;not null;default:pending_payment" json:"status"`
	PaymentStatus     string     `gorm:"type:varchar(32);not null;default:pending" json:"payment_status"`
	FulfillmentStatus string     `gorm:"type:varchar(32);not null;default:pending" json:"fulfillment_status"`
	Metadata          string     `gorm:"type:text" json:"metadata"`
	PaidAt            *time.Time `json:"paid_at,omitempty"`
	FulfilledAt       *time.Time `json:"fulfilled_at,omitempty"`
	CreatedAt         time.Time  `gorm:"index:idx_commercial_order_org_created,priority:3,sort:desc;autoCreateTime" json:"created_at"`
	UpdatedAt         time.Time  `gorm:"autoUpdateTime" json:"updated_at"`
}

type CommercialPayment struct {
	ID                string     `gorm:"type:varchar(64);primaryKey" json:"id"`
	OrderID           string     `gorm:"type:varchar(64);index;not null" json:"order_id"`
	UserID            string     `gorm:"type:varchar(64);not null" json:"user_id"`
	OrganizationID    string     `gorm:"type:varchar(64);index;not null" json:"organization_id"`
	Amount            int64      `gorm:"not null;default:0" json:"amount"`
	Currency          string     `gorm:"type:varchar(16);not null" json:"currency"`
	PaymentMethod     string     `gorm:"type:varchar(32);not null;default:manual" json:"payment_method"`
	ProviderCode      string     `gorm:"type:varchar(64);not null;default:manual_success" json:"provider_code"`
	ExternalPaymentID string     `gorm:"type:varchar(128)" json:"external_payment_id"`
	Status            string     `gorm:"type:varchar(32);index;not null;default:succeeded" json:"status"`
	Metadata          string     `gorm:"type:text" json:"metadata"`
	PaidAt            *time.Time `json:"paid_at,omitempty"`
	CreatedAt         time.Time  `gorm:"autoCreateTime" json:"created_at"`
	UpdatedAt         time.Time  `gorm:"autoUpdateTime" json:"updated_at"`
}

type CommercialFulfillment struct {
	ID                string     `gorm:"type:varchar(64);primaryKey" json:"id"`
	OrderID           string     `gorm:"type:varchar(64);index;not null" json:"order_id"`
	UserID            string     `gorm:"type:varchar(64);not null" json:"user_id"`
	OrganizationID    string     `gorm:"type:varchar(64);index;not null" json:"organization_id"`
	PackageCode       string     `gorm:"type:varchar(128);not null" json:"package_code"`
	FulfillmentMode   string     `gorm:"type:varchar(32);not null" json:"fulfillment_mode"`
	Status            string     `gorm:"type:varchar(32);index;not null;default:succeeded" json:"status"`
	AssetCode         string     `gorm:"type:varchar(64)" json:"asset_code"`
	Amount            int64      `gorm:"not null;default:0" json:"amount"`
	AllowancePolicyID string     `gorm:"type:varchar(64)" json:"allowance_policy_id"`
	CycleKey          string     `gorm:"type:varchar(32)" json:"cycle_key"`
	WalletAccountID   string     `gorm:"type:varchar(64)" json:"wallet_account_id"`
	WalletBucketID    string     `gorm:"type:varchar(64)" json:"wallet_bucket_id"`
	WalletLedgerID    string     `gorm:"type:varchar(64)" json:"wallet_ledger_id"`
	Metadata          string     `gorm:"type:text" json:"metadata"`
	ExpiresAt         *time.Time `json:"expires_at,omitempty"`
	FulfilledAt       *time.Time `json:"fulfilled_at,omitempty"`
	CreatedAt         time.Time  `gorm:"autoCreateTime" json:"created_at"`
	UpdatedAt         time.Time  `gorm:"autoUpdateTime" json:"updated_at"`
}
