package user

import (
	"encoding/json"
	"errors"
	"fmt"
	"maps"
	"sort"
	"strings"
	"time"

	"menu-service/internal/models"
	audit "menu-service/internal/modules/audit"
	"menu-service/internal/modules/auth"
	"menu-service/internal/platform"
	"menu-service/internal/repository"
)

type Service struct {
	repo       *repository.UserRepository
	commercial *repository.CommercialRepository
	studio     *repository.StudioRepository
	platform   *platform.Client
	auth       *auth.Service
	audit      *audit.Service
}

type ActivityItem struct {
	ID           string `json:"id"`
	ActionType   string `json:"action_type"`
	ActionName   string `json:"action_name"`
	Status       string `json:"status"`
	CreditsUsed  int64  `json:"credits_used"`
	CreatedAt    string `json:"created_at"`
	ResultURL    string `json:"result_url,omitempty"`
	EventID      string `json:"event_id,omitempty"`
	JobID        string `json:"job_id,omitempty"`
	ErrorMessage string `json:"error_message,omitempty"`
}

type ActivitiesResult struct {
	Activities []ActivityItem `json:"activities"`
	TotalCount int64          `json:"total_count"`
}

type UpdateProfileInput struct {
	Name               string `json:"name"`
	RestaurantName     string `json:"restaurant_name"`
	LanguagePreference string `json:"language_preference" binding:"omitempty,oneof=en zh th"`
}

type WalletHistoryEntry struct {
	ID               string         `json:"id"`
	Category         string         `json:"category"`
	Title            string         `json:"title"`
	Description      string         `json:"description,omitempty"`
	Direction        string         `json:"direction"`
	Amount           int64          `json:"amount"`
	AssetCode        string         `json:"asset_code,omitempty"`
	Currency         string         `json:"currency,omitempty"`
	Status           string         `json:"status"`
	OccurredAt       string         `json:"occurred_at"`
	ReferenceType    string         `json:"reference_type,omitempty"`
	ReferenceID      string         `json:"reference_id,omitempty"`
	JobID            string         `json:"job_id,omitempty"`
	EventID          string         `json:"event_id,omitempty"`
	SettlementID     string         `json:"settlement_id,omitempty"`
	BillableItemCode string         `json:"billable_item_code,omitempty"`
	ChargeMode       string         `json:"charge_mode,omitempty"`
	QuotaConsumed    int64          `json:"quota_consumed,omitempty"`
	CreditsConsumed  int64          `json:"credits_consumed,omitempty"`
	WalletDebited    int64          `json:"wallet_debited,omitempty"`
	FlowStatus       string         `json:"flow_status,omitempty"`
	Metadata         map[string]any `json:"metadata,omitempty"`
}

type WalletHistoryResult struct {
	Items []WalletHistoryEntry `json:"items"`
}

type QuotaSummaryResult struct {
	ProductCode      string `json:"product_code"`
	BillableItemCode string `json:"billable_item_code"`
	Granted          int64  `json:"granted"`
	Consumed         int64  `json:"consumed"`
	Reserved         int64  `json:"reserved"`
	Remaining        int64  `json:"remaining"`
}

type CommercialOfferingsResult struct {
	ProductCode   string                  `json:"product_code"`
	Offerings     *platform.OfferingsView `json:"offerings"`
	WalletSummary *platform.WalletSummary `json:"wallet_summary,omitempty"`
}

type SimulateCommercialConsumptionInput struct {
	TargetOrgID      string `json:"target_org_id,omitempty"`
	BillableItemCode string `json:"billable_item_code,omitempty"`
	Units            int64  `json:"units,omitempty"`
	ResourceType     string `json:"resource_type,omitempty"`
	ReservationKey   string `json:"reservation_key,omitempty"`
	FinalizationID   string `json:"finalization_id,omitempty"`
	EventID          string `json:"event_id,omitempty"`
	ReferenceID      string `json:"reference_id,omitempty"`
	Metadata         string `json:"metadata,omitempty"`
}

type SimulateCommercialConsumptionResult struct {
	TargetOrgID      string                        `json:"target_org_id"`
	ProductCode      string                        `json:"product_code"`
	ResourceType     string                        `json:"resource_type"`
	BillableItemCode string                        `json:"billable_item_code"`
	Units            int64                         `json:"units"`
	Reservation      *platform.ResourceReservation `json:"reservation,omitempty"`
	Settlement       *platform.SettlementRecord    `json:"settlement,omitempty"`
	BeforeWallet     *platform.WalletSummary       `json:"before_wallet,omitempty"`
	AfterWallet      *platform.WalletSummary       `json:"after_wallet,omitempty"`
}

type AssignCommercialPackageInput struct {
	PackageCode string `json:"package_code"`
	TargetOrgID string `json:"target_org_id,omitempty"`
	CycleKey    string `json:"cycle_key,omitempty"`
	ReferenceID string `json:"reference_id,omitempty"`
	Metadata    string `json:"metadata,omitempty"`
}

type AssignCommercialPackageResult struct {
	TargetOrgID       string                  `json:"target_org_id"`
	PackageCode       string                  `json:"package_code"`
	PackageType       string                  `json:"package_type"`
	FulfillmentMode   string                  `json:"fulfillment_mode"`
	AssetCode         string                  `json:"asset_code"`
	Amount            int64                   `json:"amount"`
	GrantedQuotaUnits int64                   `json:"granted_quota_units,omitempty"`
	AllowancePolicyID string                  `json:"allowance_policy_id,omitempty"`
	CycleKey          string                  `json:"cycle_key,omitempty"`
	ExpiresAt         *time.Time              `json:"expires_at,omitempty"`
	WalletAccount     *platform.WalletAccount `json:"wallet_account,omitempty"`
	WalletBucket      *platform.WalletBucket  `json:"wallet_bucket,omitempty"`
	WalletLedger      *platform.WalletLedger  `json:"wallet_ledger,omitempty"`
	WalletSummary     *platform.WalletSummary `json:"wallet_summary,omitempty"`
}

type CreateCommercialOrderInput struct {
	SKUCode     string `json:"sku_code,omitempty"`
	PackageCode string `json:"package_code,omitempty"`
	Quantity    int64  `json:"quantity,omitempty"`
	Metadata    string `json:"metadata,omitempty"`
}

type ConfirmCommercialOrderPaymentInput struct {
	PaymentMethod     string `json:"payment_method,omitempty"`
	ProviderCode      string `json:"provider_code,omitempty"`
	PaymentAssetCode  string `json:"payment_asset_code,omitempty"`
	ExternalPaymentID string `json:"external_payment_id,omitempty"`
	Metadata          string `json:"metadata,omitempty"`
}

type CommercialOrderView struct {
	Order         *models.CommercialOrder       `json:"order,omitempty"`
	Payment       *models.CommercialPayment     `json:"payment,omitempty"`
	Fulfillment   *models.CommercialFulfillment `json:"fulfillment,omitempty"`
	WalletSummary *platform.WalletSummary       `json:"wallet_summary,omitempty"`
}

type CommercialOrdersResult struct {
	Items []CommercialOrderView `json:"items"`
}

type commercialOrderBundle struct {
	SKU        *platform.SKU
	Package    *platform.CommercialPackage
	UnitAmount int64
	Currency   string
}

const menuPaymentAssetCode = "MENU_CASH"
const menuCreditsPaymentAssetCode = "MENU_CREDIT"
const menuCreditsPerRMB = int64(10)
const menuQuotaBillableItemCode = "menu.render.call"

func NewService(repo *repository.UserRepository, commercialRepo *repository.CommercialRepository, studioRepo *repository.StudioRepository, platformClient *platform.Client, authService *auth.Service, auditService *audit.Service) *Service {
	return &Service{repo: repo, commercial: commercialRepo, studio: studioRepo, platform: platformClient, auth: authService, audit: auditService}
}

func (s *Service) Activities(userID, orgID string, limit, offset int) (*ActivitiesResult, error) {
	items, total, err := s.repo.ListActivities(userID, orgID, limit, offset)
	if err != nil {
		return nil, err
	}
	out := make([]ActivityItem, 0, len(items))
	for _, item := range items {
		out = append(out, mapActivity(item))
	}
	return &ActivitiesResult{Activities: out, TotalCount: total}, nil
}

func (s *Service) Profile(userID, orgID string) (*auth.UserSummary, error) {
	session, err := s.auth.Session(userID, orgID)
	if err != nil {
		return nil, err
	}
	return &session.User, nil
}

func (s *Service) Credits(userID, orgID string) (*auth.CreditsSummary, error) {
	return s.auth.Credits(userID, orgID)
}

func (s *Service) WalletSummary(orgID string) (*auth.WalletSummary, error) {
	return s.auth.WalletSummary(orgID)
}

func (s *Service) QuotaSummary(orgID string) (*QuotaSummaryResult, error) {
	if s.platform == nil {
		return nil, errors.New("platform client unavailable")
	}
	balance, err := s.platform.GetQuotaBalance("organization", orgID, menuQuotaBillableItemCode)
	if err != nil {
		return nil, err
	}
	return &QuotaSummaryResult{
		ProductCode:      "menu",
		BillableItemCode: balance.BillableItemCode,
		Granted:          balance.Granted,
		Consumed:         balance.Consumed,
		Reserved:         balance.Reserved,
		Remaining:        balance.Available,
	}, nil
}

func (s *Service) AuditHistory(userID, orgID, targetType, status string, limit, offset int) (*audit.HistoryResult, error) {
	if s.audit == nil {
		return &audit.HistoryResult{Items: []audit.HistoryItem{}, Total: 0}, nil
	}
	return s.audit.History(orgID, userID, targetType, status, limit, offset)
}

func (s *Service) WalletHistory(orgID string, limit int) (*WalletHistoryResult, error) {
	entries := make([]WalletHistoryEntry, 0)

	if s.platform != nil {
		rewards, err := s.platform.ListRewards("menu", "organization", orgID)
		if err != nil {
			return nil, err
		}
		for _, item := range rewards {
			entries = append(entries, mapRewardHistory(item))
		}

		commissions, err := s.platform.ListCommissions("menu", "organization", orgID, "")
		if err != nil {
			return nil, err
		}
		for _, item := range commissions {
			entries = append(entries, mapCommissionHistory(item))
		}

		accounts, err := s.platform.ListWalletAccounts("organization", orgID, "menu")
		if err != nil {
			return nil, err
		}
		for _, account := range accounts {
			ledgers, ledgerErr := s.platform.ListWalletLedger(account.ID, "menu")
			if ledgerErr != nil {
				return nil, ledgerErr
			}
			for _, item := range ledgers {
				if entry, ok := mapWalletLedgerHistory(item); ok {
					entries = append(entries, entry)
				}
			}
		}
	}

	if s.studio != nil {
		intents, err := s.studio.ListChargeIntents(orgID, 0)
		if err != nil {
			return nil, err
		}
		for _, item := range intents {
			entries = append(entries, mapChargeIntentHistory(item))
		}
	}

	sort.Slice(entries, func(i, j int) bool {
		return entries[i].OccurredAt > entries[j].OccurredAt
	})
	if limit > 0 && len(entries) > limit {
		entries = entries[:limit]
	}
	return &WalletHistoryResult{Items: entries}, nil
}

func (s *Service) CommercialOfferings(orgID string) (*CommercialOfferingsResult, error) {
	if s.platform == nil {
		return nil, errors.New("platform client unavailable")
	}
	offerings, err := s.platform.GetCatalogOfferings("menu")
	if err != nil {
		return nil, err
	}
	var summary *platform.WalletSummary
	if strings.TrimSpace(orgID) != "" {
		summary, _ = s.platform.GetWalletSummary("organization", orgID, "menu")
	}
	return &CommercialOfferingsResult{
		ProductCode:   "menu",
		Offerings:     offerings,
		WalletSummary: summary,
	}, nil
}

func (s *Service) CreateCommercialOrder(userID, orgID string, input CreateCommercialOrderInput) (*CommercialOrderView, error) {
	if s.platform == nil {
		return nil, errors.New("platform client unavailable")
	}
	if s.commercial == nil {
		return nil, errors.New("commercial repository unavailable")
	}
	offerings, err := s.platform.GetCatalogOfferings("menu")
	if err != nil {
		return nil, err
	}
	orderBundle, err := s.resolveOrderBundle(offerings, strings.TrimSpace(input.SKUCode), strings.TrimSpace(input.PackageCode))
	if err != nil {
		return nil, err
	}
	metadata, err := s.buildCommercialOrderMetadata(input.Metadata, orderBundle)
	if err != nil {
		return nil, err
	}
	quantity := input.Quantity
	if quantity <= 0 {
		quantity = 1
	}
	now := time.Now().UTC()
	order := &models.CommercialOrder{
		UserID:            userID,
		OrganizationID:    orgID,
		ProductCode:       "menu",
		SKUCode:           orderBundle.SKU.Code,
		PackageCode:       orderBundle.Package.Code,
		PackageType:       orderBundle.Package.PackageType,
		Currency:          orderBundle.Currency,
		Quantity:          quantity,
		UnitAmount:        orderBundle.UnitAmount,
		TotalAmount:       orderBundle.UnitAmount * quantity,
		Status:            "pending_payment",
		PaymentStatus:     "pending",
		FulfillmentStatus: "pending",
		Metadata:          metadata,
		CreatedAt:         now,
		UpdatedAt:         now,
	}
	if err := s.commercial.CreateOrder(order); err != nil {
		return nil, err
	}
	return &CommercialOrderView{Order: order}, nil
}

func (s *Service) ListCommercialOrders(orgID string, limit, offset int) (*CommercialOrdersResult, error) {
	if s.commercial == nil {
		return nil, errors.New("commercial repository unavailable")
	}
	items, err := s.commercial.ListOrders(orgID, limit, offset)
	if err != nil {
		return nil, err
	}
	out := make([]CommercialOrderView, 0, len(items))
	for i := range items {
		view, err := s.buildCommercialOrderView(&items[i])
		if err != nil {
			return nil, err
		}
		out = append(out, *view)
	}
	return &CommercialOrdersResult{Items: out}, nil
}

func (s *Service) GetCommercialOrder(orgID, orderID string) (*CommercialOrderView, error) {
	if s.commercial == nil {
		return nil, errors.New("commercial repository unavailable")
	}
	order, err := s.commercial.FindOrderByID(orgID, orderID)
	if err != nil {
		return nil, err
	}
	return s.buildCommercialOrderView(order)
}

func (s *Service) ConfirmCommercialOrderPayment(userID, orgID, orderID string, input ConfirmCommercialOrderPaymentInput) (*CommercialOrderView, error) {
	if s.commercial == nil {
		return nil, errors.New("commercial repository unavailable")
	}
	order, err := s.commercial.FindOrderByID(orgID, orderID)
	if err != nil {
		return nil, err
	}
	if order.Status == "fulfilled" {
		return s.buildCommercialOrderView(order)
	}
	if order.PaymentStatus == "succeeded" && order.FulfillmentStatus == "succeeded" {
		return s.buildCommercialOrderView(order)
	}
	now := time.Now().UTC()
	paymentAssetCode := defaultString(strings.TrimSpace(input.PaymentAssetCode), paymentAssetCodeFromMetadata(decodeMap(order.Metadata)))
	if paymentAssetCode == "" {
		paymentAssetCode = menuPaymentAssetCode
	}
	var paymentAssetType string
	var paymentCurrency string
	var paymentAmount int64
	payment, _ := s.commercial.FindLatestPaymentByOrderID(order.ID)
	if payment == nil || payment.Status != "succeeded" {
		paymentAssetType, paymentCurrency, paymentAmount, err = resolveCommercialPaymentCharge(order, paymentAssetCode)
		if err != nil {
			return nil, err
		}
		walletLedger, _, err := s.platform.PostWalletLedger(platform.PostWalletLedgerInput{
			BillingSubjectType: "organization",
			BillingSubjectID:   orgID,
			AssetCode:          paymentAssetCode,
			AssetType:          paymentAssetType,
			Direction:          "debit",
			Amount:             paymentAmount,
			Reason:             "commercial_order_payment",
			ReferenceType:      "commercial_order",
			ReferenceID:        order.ID,
			Metadata:           order.Metadata,
		})
		if err != nil {
			return nil, err
		}
		paymentMetadata, err := encodeMap(mergeMaps(
			decodeMap(order.Metadata),
			decodeMap(input.Metadata),
			map[string]any{
				"source":                "menu_commercial_payment_confirm",
				"order_id":              order.ID,
				"payment_asset_code":    paymentAssetCode,
				"wallet_ledger_id":      walletLedger.ID,
				"wallet_ledger_reason":  walletLedger.Reason,
				"wallet_reference_type": walletLedger.ReferenceType,
			},
		))
		if err != nil {
			return nil, err
		}
		payment = &models.CommercialPayment{
			OrderID:           order.ID,
			UserID:            userID,
			OrganizationID:    orgID,
			Amount:            paymentAmount,
			Currency:          paymentCurrency,
			PaymentMethod:     defaultString(strings.TrimSpace(input.PaymentMethod), "wallet_balance"),
			ProviderCode:      defaultString(strings.TrimSpace(input.ProviderCode), "platform_wallet"),
			ExternalPaymentID: defaultString(strings.TrimSpace(input.ExternalPaymentID), walletLedger.ID),
			Status:            "succeeded",
			Metadata:          paymentMetadata,
			PaidAt:            &now,
			CreatedAt:         now,
			UpdatedAt:         now,
		}
		if err := s.commercial.CreatePayment(payment); err != nil {
			return nil, err
		}
	}
	if order.PaymentStatus != "succeeded" {
		order.PaymentStatus = "succeeded"
		order.Status = "paid"
		order.PaidAt = &now
		order.UpdatedAt = now
		if err := s.commercial.SaveOrder(order); err != nil {
			return nil, err
		}
	}
	assignMetadata, err := encodeMap(mergeMaps(
		decodeMap(order.Metadata),
		decodeMap(input.Metadata),
		map[string]any{
			"source":             "menu_commercial_order_fulfillment",
			"order_id":           order.ID,
			"payment_id":         payment.ID,
			"sku_code":           order.SKUCode,
			"payment_asset_code": paymentAssetCode,
		},
	))
	if err != nil {
		return nil, err
	}
	fulfillment, _ := s.commercial.FindLatestFulfillmentByOrderID(order.ID)
	var assignResult *AssignCommercialPackageResult
	if fulfillment == nil || fulfillment.Status != "succeeded" {
		assignResult, err = s.AssignCommercialPackage(userID, orgID, AssignCommercialPackageInput{
			PackageCode: order.PackageCode,
			TargetOrgID: orgID,
			ReferenceID: order.ID,
			Metadata:    assignMetadata,
		})
		if err != nil {
			order.Status = "payment_succeeded_fulfillment_failed"
			order.FulfillmentStatus = "failed"
			order.UpdatedAt = time.Now().UTC()
			_ = s.commercial.SaveOrder(order)
			return nil, err
		}
		fulfillmentMetadata, err := encodeMap(mergeMaps(
			decodeMap(order.Metadata),
			map[string]any{
				"source":           "menu_commercial_order_fulfillment",
				"order_id":         order.ID,
				"payment_id":       payment.ID,
				"fulfillment_mode": assignResult.FulfillmentMode,
			},
		))
		if err != nil {
			return nil, err
		}
		fulfillment = &models.CommercialFulfillment{
			OrderID:           order.ID,
			UserID:            userID,
			OrganizationID:    orgID,
			PackageCode:       order.PackageCode,
			FulfillmentMode:   assignResult.FulfillmentMode,
			Status:            "succeeded",
			AssetCode:         assignResult.AssetCode,
			Amount:            assignResult.Amount,
			AllowancePolicyID: assignResult.AllowancePolicyID,
			CycleKey:          assignResult.CycleKey,
			Metadata:          fulfillmentMetadata,
			ExpiresAt:         assignResult.ExpiresAt,
			FulfilledAt:       &now,
			CreatedAt:         now,
			UpdatedAt:         now,
		}
		if assignResult.WalletAccount != nil {
			fulfillment.WalletAccountID = assignResult.WalletAccount.ID
		}
		if assignResult.WalletBucket != nil {
			fulfillment.WalletBucketID = assignResult.WalletBucket.ID
		}
		if assignResult.WalletLedger != nil {
			fulfillment.WalletLedgerID = assignResult.WalletLedger.ID
		}
		if err := s.commercial.CreateFulfillment(fulfillment); err != nil {
			return nil, err
		}
	} else {
		assignResult = &AssignCommercialPackageResult{
			FulfillmentMode:   fulfillment.FulfillmentMode,
			AssetCode:         fulfillment.AssetCode,
			Amount:            fulfillment.Amount,
			AllowancePolicyID: fulfillment.AllowancePolicyID,
			CycleKey:          fulfillment.CycleKey,
			ExpiresAt:         fulfillment.ExpiresAt,
		}
	}
	order.Status = "fulfilled"
	order.FulfillmentStatus = "succeeded"
	order.FulfilledAt = &now
	order.UpdatedAt = now
	if err := s.commercial.SaveOrder(order); err != nil {
		return nil, err
	}
	walletSummary := assignResult.WalletSummary
	if walletSummary == nil {
		walletSummary, _ = s.platform.GetWalletSummary("organization", orgID, order.ProductCode)
	}
	return &CommercialOrderView{
		Order:         order,
		Payment:       payment,
		Fulfillment:   fulfillment,
		WalletSummary: walletSummary,
	}, nil
}

func (s *Service) AssignCommercialPackage(actorUserID, orgID string, input AssignCommercialPackageInput) (*AssignCommercialPackageResult, error) {
	if s.platform == nil {
		return nil, errors.New("platform client unavailable")
	}
	packageCode := strings.TrimSpace(input.PackageCode)
	if packageCode == "" {
		return nil, errors.New("package_code is required")
	}
	targetOrgID := defaultString(strings.TrimSpace(input.TargetOrgID), orgID)
	if targetOrgID == "" {
		return nil, errors.New("target organization is required")
	}
	offerings, err := s.platform.GetCatalogOfferings("menu")
	if err != nil {
		return nil, err
	}
	pkg := findCommercialPackage(offerings.Packages, packageCode)
	if pkg == nil {
		return nil, fmt.Errorf("package not found: %s", packageCode)
	}
	if pkg.Status != "active" {
		return nil, fmt.Errorf("package is not active: %s", packageCode)
	}
	inputMetadata, err := decodeMapStrict(input.Metadata)
	if err != nil {
		return nil, err
	}
	referenceID := defaultString(strings.TrimSpace(input.ReferenceID), fmt.Sprintf("menu:package_assign:%s:%d", packageCode, time.Now().UnixNano()))
	baseMetadata := map[string]any{
		"source":               "menu_admin_assign_package",
		"package_code":         pkg.Code,
		"package_type":         pkg.PackageType,
		"target_org_id":        targetOrgID,
		"assigned_by_user_id":  actorUserID,
		"assigned_from_org_id": orgID,
	}
	quotaPolicies, err := s.platform.ListQuotaGrantPolicies("menu", pkg.Code)
	if err != nil {
		return nil, err
	}
	capabilityPolicies, err := s.platform.ListPackageCapabilityPolicies("menu", pkg.Code)
	if err != nil {
		return nil, err
	}
	if len(quotaPolicies) == 0 {
		return nil, fmt.Errorf("quota grant policy not found for package: %s", packageCode)
	}
	var grantedUnits int64
	for _, policy := range quotaPolicies {
		if policy.Status != "active" || policy.Units <= 0 {
			continue
		}
		if err := s.platform.GrantQuota(platform.GrantQuotaInput{
			BillingSubjectType: "organization",
			BillingSubjectID:   targetOrgID,
			BillableItemCode:   policy.BillableItemCode,
			Units:              policy.Units,
			Reason:             "commercial_package_assignment",
			ReferenceID:        referenceID,
		}); err != nil {
			return nil, err
		}
		grantedUnits += policy.Units
	}
	for _, policy := range capabilityPolicies {
		if policy.Status != "active" || strings.TrimSpace(policy.GrantValue) == "" {
			continue
		}
		metadataJSON, err := encodeMap(mergeMaps(
			baseMetadata,
			inputMetadata,
			map[string]any{"capability_policy_id": policy.ID},
		))
		if err != nil {
			return nil, err
		}
		if _, err := s.platform.GrantCapability(platform.GrantCapabilityInput{
			ProductCode:        "menu",
			BillingSubjectType: "organization",
			BillingSubjectID:   targetOrgID,
			CapabilityCode:     policy.CapabilityCode,
			GrantValue:         policy.GrantValue,
			SourceType:         "commercial_package",
			SourceID:           referenceID,
			Metadata:           metadataJSON,
		}); err != nil {
			return nil, err
		}
	}
	summary, _ := s.platform.GetWalletSummary("organization", targetOrgID, "menu")
	return &AssignCommercialPackageResult{
		TargetOrgID:       targetOrgID,
		PackageCode:       pkg.Code,
		PackageType:       pkg.PackageType,
		FulfillmentMode:   "entitlement_grant",
		Amount:            grantedUnits,
		GrantedQuotaUnits: grantedUnits,
		WalletSummary:     summary,
	}, nil
}

func (s *Service) SimulateCommercialConsumption(actorUserID, orgID string, input SimulateCommercialConsumptionInput) (*SimulateCommercialConsumptionResult, error) {
	if s.platform == nil {
		return nil, errors.New("platform client unavailable")
	}
	targetOrgID := defaultString(strings.TrimSpace(input.TargetOrgID), orgID)
	if targetOrgID == "" {
		return nil, errors.New("target organization is required")
	}
	offerings, err := s.platform.GetCatalogOfferings("menu")
	if err != nil {
		return nil, err
	}
	billableItemCode := strings.TrimSpace(input.BillableItemCode)
	if billableItemCode == "" {
		item := pickDefaultBillableItem(offerings.BillableItems)
		if item == nil {
			return nil, errors.New("no active billable item found")
		}
		billableItemCode = item.Code
	}
	units := input.Units
	if units <= 0 {
		units = 1
	}
	resourceType := defaultString(strings.TrimSpace(input.ResourceType), "credits")
	referenceID := defaultString(strings.TrimSpace(input.ReferenceID), fmt.Sprintf("menu:consume_probe:%d", time.Now().UnixNano()))
	reservationKey := defaultString(strings.TrimSpace(input.ReservationKey), fmt.Sprintf("menu:reserve_probe:%s", referenceID))
	finalizationID := defaultString(strings.TrimSpace(input.FinalizationID), fmt.Sprintf("menu:finalize_probe:%s", referenceID))
	eventID := defaultString(strings.TrimSpace(input.EventID), fmt.Sprintf("menu:event_probe:%s", referenceID))
	inputMetadata, err := decodeMapStrict(input.Metadata)
	if err != nil {
		return nil, err
	}
	beforeWallet, _ := s.platform.GetWalletSummary("organization", targetOrgID, "menu")
	reservationMetadata, err := encodeMap(mergeMaps(inputMetadata, map[string]any{
		"source":             "menu_admin_consume_probe",
		"target_org_id":      targetOrgID,
		"actor_user_id":      actorUserID,
		"actor_org_id":       orgID,
		"billable_item_code": billableItemCode,
		"reference_id":       referenceID,
	}))
	if err != nil {
		return nil, err
	}
	reservation, err := s.platform.ReserveResources(platform.ReserveInput{
		ResourceType:       resourceType,
		BillingSubjectType: "organization",
		BillingSubjectID:   targetOrgID,
		BillableItemCode:   billableItemCode,
		ReservationKey:     reservationKey,
		Units:              units,
		ReferenceID:        referenceID,
		Metadata:           reservationMetadata,
	})
	if err != nil {
		return nil, err
	}
	finalizeMetadata, err := encodeMap(mergeMaps(inputMetadata, map[string]any{
		"source":             "menu_admin_consume_probe",
		"target_org_id":      targetOrgID,
		"actor_user_id":      actorUserID,
		"actor_org_id":       orgID,
		"billable_item_code": billableItemCode,
		"reference_id":       referenceID,
		"reservation_id":     reservation.ID,
	}))
	if err != nil {
		return nil, err
	}
	result, finalizeErr := s.platform.FinalizeMetering(platform.FinalizeInput{
		FinalizationID: finalizationID,
		ReservationID:  reservation.ID,
		IngestEventInput: platform.IngestEventInput{
			EventID:            eventID,
			SourceType:         "menu_admin_probe",
			SourceID:           referenceID,
			SourceAction:       "consume_probe",
			ProductCode:        "menu",
			OrgID:              targetOrgID,
			UserID:             actorUserID,
			BillableItemCode:   billableItemCode,
			BillingSubjectType: "organization",
			BillingSubjectID:   targetOrgID,
			UsageUnits:         units,
			Unit:               "action",
			Dimensions:         finalizeMetadata,
			OccurredAt:         time.Now().UTC().Format(time.RFC3339),
		},
	})
	if finalizeErr != nil {
		_, _ = s.platform.ReleaseReservation(reservation.ID)
		return nil, finalizeErr
	}
	afterWallet, _ := s.platform.GetWalletSummary("organization", targetOrgID, "menu")
	return &SimulateCommercialConsumptionResult{
		TargetOrgID:      targetOrgID,
		ProductCode:      "menu",
		ResourceType:     resourceType,
		BillableItemCode: billableItemCode,
		Units:            units,
		Reservation:      reservation,
		Settlement:       result.Settlement,
		BeforeWallet:     beforeWallet,
		AfterWallet:      afterWallet,
	}, nil
}

func (s *Service) UpdateProfile(userID, orgID string, input UpdateProfileInput) (*auth.UserSummary, error) {
	if input.Name != "" {
		if _, err := s.platform.UpdateUserProfile(userID, platform.UpdateUserProfileInput{FullName: input.Name}); err != nil {
			return nil, err
		}
	}
	if input.RestaurantName != "" {
		if err := s.platform.UpdateOrganizationProfile(orgID, platform.UpdateOrganizationProfileInput{Name: input.RestaurantName}); err != nil {
			return nil, err
		}
	}
	if input.LanguagePreference != "" {
		if _, err := s.repo.UpsertPreference(userID, orgID, input.LanguagePreference); err != nil {
			return nil, err
		}
	}
	return s.Profile(userID, orgID)
}

func mapActivity(item models.Activity) ActivityItem {
	return ActivityItem{
		ID:           item.ID,
		ActionType:   item.ActionType,
		ActionName:   item.ActionName,
		Status:       item.Status,
		CreditsUsed:  item.CreditsUsed,
		CreatedAt:    item.CreatedAt.UTC().Format("2006-01-02T15:04:05Z"),
		ResultURL:    item.ResultURL,
		EventID:      item.EventID,
		JobID:        item.JobID,
		ErrorMessage: item.ErrorMessage,
	}
}

func mapRewardHistory(item platform.RewardLedger) WalletHistoryEntry {
	category := "reward"
	title := "Reward credit issued"
	if item.RewardType == "commission_redeem" {
		category = "redeem"
		title = "Commission redeemed to credits"
	}
	return WalletHistoryEntry{
		ID:            item.ID,
		Category:      category,
		Title:         title,
		Direction:     "credit",
		Amount:        item.Amount,
		AssetCode:     item.AssetCode,
		Status:        item.Status,
		OccurredAt:    item.CreatedAt.UTC().Format(time.RFC3339),
		ReferenceType: item.ReferenceType,
		ReferenceID:   item.ReferenceID,
		Metadata:      decodeMap(item.Metadata),
	}
}

func mapCommissionHistory(item platform.CommissionLedger) WalletHistoryEntry {
	title := "Referral commission earned"
	switch item.Status {
	case "redeemed":
		title = "Referral commission redeemed"
	case "reversed":
		title = "Referral commission reversed"
	}
	return WalletHistoryEntry{
		ID:            item.ID,
		Category:      "commission",
		Title:         title,
		Direction:     "info",
		Amount:        item.Amount,
		Currency:      item.Currency,
		Status:        item.Status,
		OccurredAt:    item.CreatedAt.UTC().Format(time.RFC3339),
		ReferenceType: item.ReferenceType,
		ReferenceID:   item.ReferenceID,
		Metadata:      decodeMap(item.Metadata),
	}
}

func mapWalletLedgerHistory(item platform.WalletLedger) (WalletHistoryEntry, bool) {
	if item.AssetCode == "MENU_MONTHLY_ALLOWANCE" {
		return WalletHistoryEntry{}, false
	}
	switch item.Reason {
	case "reward_issue", "metering_settlement":
		return WalletHistoryEntry{}, false
	case "asset_expire":
		return WalletHistoryEntry{
			ID:            item.ID,
			Category:      "expiration",
			Title:         "Credits expired",
			Direction:     "debit",
			Amount:        item.Amount,
			AssetCode:     item.AssetCode,
			Status:        item.Status,
			OccurredAt:    item.CreatedAt.UTC().Format(time.RFC3339),
			ReferenceType: item.ReferenceType,
			ReferenceID:   item.ReferenceID,
			Metadata:      decodeMap(item.Metadata),
		}, true
	default:
		title := "Balance adjustment"
		category := "wallet_adjustment"
		if item.Direction == "credit" {
			title = "Credits recharged"
			category = "recharge"
		}
		return WalletHistoryEntry{
			ID:            item.ID,
			Category:      category,
			Title:         title,
			Direction:     normalizeDirection(item.Direction),
			Amount:        item.Amount,
			AssetCode:     item.AssetCode,
			Status:        item.Status,
			OccurredAt:    item.CreatedAt.UTC().Format(time.RFC3339),
			ReferenceType: item.ReferenceType,
			ReferenceID:   item.ReferenceID,
			Metadata:      decodeMap(item.Metadata),
		}, true
	}
}

func mapChargeIntentHistory(item models.StudioChargeIntent) WalletHistoryEntry {
	metadata := decodeMap(item.Metadata)
	settlement, _ := metadata["settlement"].(map[string]any)
	title := "Studio generation charge"
	switch item.Status {
	case "released":
		title = "Studio reservation released"
	case "failed_need_reconcile":
		title = "Studio charge needs reconciliation"
	}
	occurredAt := item.UpdatedAt.UTC().Format(time.RFC3339)
	if item.FinalizedAt != nil {
		occurredAt = item.FinalizedAt.UTC().Format(time.RFC3339)
	} else if item.ReleasedAt != nil {
		occurredAt = item.ReleasedAt.UTC().Format(time.RFC3339)
	} else if item.ReservedAt != nil {
		occurredAt = item.ReservedAt.UTC().Format(time.RFC3339)
	}
	quotaConsumed := int64MapValue(settlement, "quota_consumed")
	creditsConsumed := int64MapValue(settlement, "credits_consumed")
	walletDebited := int64MapValue(settlement, "wallet_debited")
	assetCode := stringMapValue(settlement, "wallet_asset_code")
	displayAmount := int64MapValue(settlement, "net_amount")
	if quotaConsumed > 0 && displayAmount == 0 {
		displayAmount = quotaConsumed
		if assetCode == "" {
			assetCode = menuQuotaBillableItemCode
		}
	} else if creditsConsumed > 0 && displayAmount == 0 {
		displayAmount = creditsConsumed
		if assetCode == "" {
			assetCode = "MENU_CREDIT"
		}
	} else if walletDebited > 0 && displayAmount == 0 {
		displayAmount = walletDebited
	}
	description := buildChargeHistoryDescription(assetCode, quotaConsumed, creditsConsumed, walletDebited)
	return WalletHistoryEntry{
		ID:               item.ID,
		Category:         "charge",
		Title:            title,
		Description:      description,
		Direction:        "debit",
		Amount:           displayAmount,
		AssetCode:        assetCode,
		Currency:         stringMapValue(settlement, "currency"),
		Status:           item.Status,
		OccurredAt:       occurredAt,
		ReferenceType:    "studio_job",
		ReferenceID:      item.JobID,
		JobID:            item.JobID,
		EventID:          item.EventID,
		SettlementID:     item.SettlementID,
		BillableItemCode: item.BillableItemCode,
		ChargeMode:       item.ChargeMode,
		QuotaConsumed:    quotaConsumed,
		CreditsConsumed:  creditsConsumed,
		WalletDebited:    walletDebited,
		FlowStatus:       item.Status,
		Metadata:         metadata,
	}
}

func buildChargeHistoryDescription(assetCode string, quotaConsumed, creditsConsumed, walletDebited int64) string {
	switch assetCode {
	case menuQuotaBillableItemCode:
		if quotaConsumed > 0 {
			return fmt.Sprintf("Consumed %d quota from menu generation package", quotaConsumed)
		}
	case "MENU_CREDIT":
		if creditsConsumed > 0 {
			return fmt.Sprintf("Consumed %d credits from permanent balance", creditsConsumed)
		}
	case "MENU_PROMO_CREDIT":
		if creditsConsumed > 0 {
			return fmt.Sprintf("Consumed %d promo credits", creditsConsumed)
		}
	case "MENU_CASH":
		if walletDebited > 0 {
			return fmt.Sprintf("Debited %.2f CNY from wallet balance", float64(walletDebited)/100)
		}
	}
	if quotaConsumed > 0 {
		return fmt.Sprintf("Consumed %d quota", quotaConsumed)
	}
	if creditsConsumed > 0 {
		return fmt.Sprintf("Consumed %d credits", creditsConsumed)
	}
	if walletDebited > 0 {
		return fmt.Sprintf("Debited %d from wallet", walletDebited)
	}
	return ""
}

func decodeMap(raw string) map[string]any {
	if strings.TrimSpace(raw) == "" {
		return map[string]any{}
	}
	out := map[string]any{}
	_ = json.Unmarshal([]byte(raw), &out)
	return out
}

func stringMapValue(values map[string]any, key string) string {
	if values == nil {
		return ""
	}
	value, ok := values[key]
	if !ok {
		return ""
	}
	str, _ := value.(string)
	return str
}

func int64MapValue(values map[string]any, key string) int64 {
	if values == nil {
		return 0
	}
	value, ok := values[key]
	if !ok {
		return 0
	}
	switch typed := value.(type) {
	case int64:
		return typed
	case int:
		return int64(typed)
	case float64:
		return int64(typed)
	default:
		return 0
	}
}

func normalizeDirection(direction string) string {
	switch direction {
	case "credit":
		return "credit"
	case "debit":
		return "debit"
	default:
		return "info"
	}
}

func decodeMapStrict(raw string) (map[string]any, error) {
	if strings.TrimSpace(raw) == "" {
		return map[string]any{}, nil
	}
	out := map[string]any{}
	if err := json.Unmarshal([]byte(raw), &out); err != nil {
		return nil, fmt.Errorf("invalid metadata json: %w", err)
	}
	return out, nil
}

func mergeMaps(items ...map[string]any) map[string]any {
	out := map[string]any{}
	for _, item := range items {
		maps.Copy(out, item)
	}
	return out
}

func (s *Service) resolveOrderBundle(offerings *platform.OfferingsView, skuCode, packageCode string) (*commercialOrderBundle, error) {
	if offerings == nil {
		return nil, errors.New("offerings unavailable")
	}
	var matchedPackage *platform.CommercialPackage
	if packageCode != "" {
		matchedPackage = findCommercialPackage(offerings.Packages, packageCode)
		if matchedPackage == nil {
			return nil, fmt.Errorf("package not found: %s", packageCode)
		}
	}
	var matchedSKU *platform.SKU
	if skuCode != "" {
		matchedSKU = findSKU(offerings.SKUs, skuCode)
		if matchedSKU == nil {
			return nil, fmt.Errorf("sku not found: %s", skuCode)
		}
	}
	if matchedPackage == nil && matchedSKU == nil {
		return nil, errors.New("sku_code or package_code is required")
	}
	if matchedPackage == nil && matchedSKU != nil {
		skuMetadata := decodeMap(matchedSKU.Metadata)
		pkgCode, _ := skuMetadata["package_code"].(string)
		matchedPackage = findCommercialPackage(offerings.Packages, pkgCode)
		if matchedPackage == nil {
			return nil, fmt.Errorf("package not found for sku: %s", matchedSKU.Code)
		}
	}
	if matchedSKU == nil && matchedPackage != nil {
		pkgMetadata := decodeMap(matchedPackage.Metadata)
		skuCodeFromPkg, _ := pkgMetadata["sku_code"].(string)
		matchedSKU = findSKU(offerings.SKUs, skuCodeFromPkg)
		if matchedSKU == nil {
			return nil, fmt.Errorf("sku not found for package: %s", matchedPackage.Code)
		}
	}
	if matchedPackage.Status != "active" || matchedSKU.Status != "active" {
		return nil, errors.New("commercial sku/package is not active")
	}
	rateCard := findBestRateCardForOrder(offerings.RateCards, matchedSKU, matchedPackage.Code)
	currency := matchedSKU.Currency
	unitAmount := matchedSKU.ListPrice
	if rateCard != nil {
		rateConfig := decodeMap(rateCard.PriceConfig)
		if amount := int64MapValue(rateConfig, "unit_amount"); amount > 0 {
			unitAmount = amount
		}
		if strings.TrimSpace(rateCard.Currency) != "" {
			currency = rateCard.Currency
		}
	}
	return &commercialOrderBundle{
		SKU:        matchedSKU,
		Package:    matchedPackage,
		UnitAmount: unitAmount,
		Currency:   currency,
	}, nil
}

func (s *Service) buildCommercialOrderMetadata(raw string, bundle *commercialOrderBundle) (string, error) {
	inputMetadata, err := decodeMapStrict(raw)
	if err != nil {
		return "", err
	}
	return encodeMap(mergeMaps(inputMetadata, map[string]any{
		"sku_code":           bundle.SKU.Code,
		"package_code":       bundle.Package.Code,
		"package_type":       bundle.Package.PackageType,
		"product_code":       "menu",
		"payment_asset_code": menuPaymentAssetCode,
	}))
}

func (s *Service) buildCommercialOrderView(order *models.CommercialOrder) (*CommercialOrderView, error) {
	view := &CommercialOrderView{Order: order}
	if order == nil || s.commercial == nil {
		return view, nil
	}
	if payment, err := s.commercial.FindLatestPaymentByOrderID(order.ID); err == nil {
		view.Payment = payment
	}
	if fulfillment, err := s.commercial.FindLatestFulfillmentByOrderID(order.ID); err == nil {
		view.Fulfillment = fulfillment
	}
	if s.platform != nil {
		summary, _ := s.platform.GetWalletSummary("organization", order.OrganizationID, order.ProductCode)
		view.WalletSummary = summary
	}
	return view, nil
}

func encodeMap(value map[string]any) (string, error) {
	if len(value) == 0 {
		return "", nil
	}
	data, err := json.Marshal(value)
	if err != nil {
		return "", err
	}
	return string(data), nil
}

func findCommercialPackage(items []platform.CommercialPackage, packageCode string) *platform.CommercialPackage {
	for i := range items {
		if items[i].Code == packageCode {
			return &items[i]
		}
	}
	return nil
}

func findSKU(items []platform.SKU, skuCode string) *platform.SKU {
	for i := range items {
		if items[i].Code == skuCode {
			return &items[i]
		}
	}
	return nil
}

func findAssetDefinition(items []platform.AssetDefinition, assetCode string) *platform.AssetDefinition {
	for i := range items {
		if items[i].AssetCode == assetCode {
			return &items[i]
		}
	}
	return nil
}

func findBestRateCardForOrder(items []platform.RateCard, sku *platform.SKU, packageCode string) *platform.RateCard {
	var best *platform.RateCard
	for i := range items {
		item := &items[i]
		if item.Status != "active" {
			continue
		}
		matchBySKU := item.TargetType == "sku" && item.TargetID == sku.ID
		matchByPackage := stringMapValue(decodeMap(item.Metadata), "package_code") == packageCode
		if !matchBySKU && !matchByPackage {
			continue
		}
		if best == nil || item.Version > best.Version {
			best = item
		}
	}
	return best
}

func pickDefaultBillableItem(items []platform.BillableItem) *platform.BillableItem {
	for i := range items {
		if items[i].Status == "active" && strings.Contains(items[i].Code, "single") {
			return &items[i]
		}
	}
	for i := range items {
		if items[i].Status == "active" {
			return &items[i]
		}
	}
	return nil
}

func buildMonthlyCycleKey(now time.Time) string {
	return now.UTC().Format("2006-01")
}

func buildExpiryFromMonths(months int, now time.Time) *time.Time {
	if months <= 0 {
		return nil
	}
	value := now.UTC().AddDate(0, months, 0)
	return &value
}

func assetTypeValue(item *platform.AssetDefinition) string {
	if item == nil {
		return ""
	}
	return item.AssetType
}

func paymentAssetCodeFromMetadata(values map[string]any) string {
	if values == nil {
		return ""
	}
	return stringMapValue(values, "payment_asset_code")
}

func resolveCommercialPaymentCharge(order *models.CommercialOrder, paymentAssetCode string) (assetType string, currency string, amount int64, err error) {
	switch paymentAssetCode {
	case "", menuPaymentAssetCode:
		return "cash_balance", defaultString(order.Currency, "CNY"), order.TotalAmount, nil
	case menuCreditsPaymentAssetCode:
		return "wallet_credit", menuCreditsPaymentAssetCode, convertCashAmountToCredits(order.TotalAmount), nil
	case "MENU_PROMO_CREDIT":
		return "reward_credit", "MENU_PROMO_CREDIT", convertCashAmountToCredits(order.TotalAmount), nil
	default:
		return "", "", 0, fmt.Errorf("unsupported payment asset code: %s", paymentAssetCode)
	}
}

func convertCashAmountToCredits(cents int64) int64 {
	if cents <= 0 {
		return 0
	}
	return (cents*menuCreditsPerRMB + 99) / 100
}

func defaultString(value, fallback string) string {
	if value == "" {
		return fallback
	}
	return value
}

func intMapValue(values map[string]any, key string) int {
	if values == nil {
		return 0
	}
	value, ok := values[key]
	if !ok {
		return 0
	}
	switch typed := value.(type) {
	case int:
		return typed
	case int64:
		return int(typed)
	case float64:
		return int(typed)
	default:
		return 0
	}
}
