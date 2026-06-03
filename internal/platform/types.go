package platform

import "time"

type envelope[T any] struct {
	Code      int    `json:"code"`
	Message   string `json:"message"`
	ErrorCode string `json:"error_code"`
	ErrorHint string `json:"error_hint"`
	RequestID string `json:"request_id"`
	Timestamp int64  `json:"timestamp"`
	Data      T      `json:"data"`
	Error     string `json:"error"`
	Errors    []struct {
		Field   string `json:"field"`
		Message string `json:"message"`
		Value   string `json:"value,omitempty"`
	} `json:"errors,omitempty"`
}

type PlatformAccessData struct {
	UserID      string   `json:"user_id"`
	OrgID       string   `json:"org_id"`
	OrgRole     string   `json:"org_role"`
	Permissions []string `json:"permissions"`
}

type AuthRegisterInput struct {
	FullName string `json:"full_name"`
	Email    string `json:"email"`
	Company  string `json:"company"`
	Password string `json:"password"`
	Avatar   string `json:"avatar,omitempty"`
}

type AuthLoginInput struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type PlatformOrganizationLite struct {
	ID   string `json:"id"`
	Name string `json:"name"`
	Role string `json:"role"`
}

type PlatformUserProfile struct {
	ID              string                     `json:"id"`
	Email           string                     `json:"email"`
	FullName        string                     `json:"full_name"`
	AvatarURL       string                     `json:"avatar_url"`
	Role            string                     `json:"role"`
	OrgRole         string                     `json:"org_role"`
	OrgID           string                     `json:"org_id"`
	LastActiveOrgID string                     `json:"last_active_org_id"`
	PlanID          string                     `json:"plan_id"`
	Status          string                     `json:"status"`
	Permissions     []string                   `json:"permissions"`
	Orgs            []PlatformOrganizationLite `json:"orgs"`
}

type PlatformTemplateCatalogItem struct {
	TemplateRef    string         `json:"template_ref"`
	ProductCode    string         `json:"product_code"`
	TemplateID     string         `json:"template_id"`
	Slug           string         `json:"slug"`
	Name           string         `json:"name"`
	Summary        string         `json:"summary"`
	Status         string         `json:"status"`
	CoverAssetURL  string         `json:"cover_asset_url"`
	CoverAssetID   string         `json:"cover_asset_id"`
	RecommendScore int            `json:"recommend_score"`
	Tags           []string       `json:"tags"`
	Platforms      []string       `json:"platforms"`
	Series         string         `json:"series"`
	CapabilityType string         `json:"capability_type"`
	Modality       string         `json:"modality"`
	Scope          string         `json:"scope"`
	ManagedSource  string         `json:"managed_source"`
	Raw            map[string]any `json:"raw"`
}

type PlatformTemplateCatalogResult struct {
	Items  []PlatformTemplateCatalogItem `json:"items"`
	Total  int                           `json:"total"`
	Limit  int                           `json:"limit"`
	Offset int                           `json:"offset"`
}

type PlatformTemplateCatalogDetail struct {
	Item      PlatformTemplateCatalogItem `json:"item"`
	Product   string                      `json:"product"`
	DetailRaw map[string]any              `json:"detail_raw"`
}

type QuotaBalance struct {
	BillingSubjectType string `json:"billing_subject_type"`
	BillingSubjectID   string `json:"billing_subject_id"`
	BillableItemCode   string `json:"billable_item_code"`
	Granted            int64  `json:"granted"`
	Consumed           int64  `json:"consumed"`
	Reserved           int64  `json:"reserved"`
	Available          int64  `json:"available"`
}

type UpdateUserProfileInput struct {
	FullName  string `json:"full_name,omitempty"`
	AvatarURL string `json:"avatar_url,omitempty"`
}

type UpdateOrganizationProfileInput struct {
	Name         string `json:"name,omitempty"`
	BillingEmail string `json:"billing_email,omitempty"`
}

type PlatformAuthResult struct {
	AccessToken string              `json:"access_token"`
	User        PlatformUserProfile `json:"user"`
}

type ReserveInput struct {
	ResourceType       string `json:"resource_type"`
	BillingSubjectType string `json:"billing_subject_type"`
	BillingSubjectID   string `json:"billing_subject_id"`
	BillableItemCode   string `json:"billable_item_code,omitempty"`
	ReservationKey     string `json:"reservation_key,omitempty"`
	Units              int64  `json:"units"`
	ReferenceID        string `json:"reference_id,omitempty"`
	Metadata           string `json:"metadata,omitempty"`
}

type ResourceReservation struct {
	ID                 string     `json:"id"`
	ResourceType       string     `json:"resource_type"`
	BillingSubjectType string     `json:"billing_subject_type"`
	BillingSubjectID   string     `json:"billing_subject_id"`
	BillableItemCode   string     `json:"billable_item_code"`
	ReservationKey     *string    `json:"reservation_key,omitempty"`
	FinalizationID     *string    `json:"finalization_id,omitempty"`
	Units              int64      `json:"units"`
	Status             string     `json:"status"`
	ReferenceID        string     `json:"reference_id"`
	Metadata           string     `json:"metadata"`
	ExpiresAt          *time.Time `json:"expires_at"`
	CommittedAt        *time.Time `json:"committed_at"`
	ReleasedAt         *time.Time `json:"released_at"`
	CreatedAt          time.Time  `json:"created_at"`
	UpdatedAt          time.Time  `json:"updated_at"`
}

type IngestEventInput struct {
	EventID               string `json:"event_id"`
	RequestID             string `json:"request_id,omitempty"`
	TraceID               string `json:"trace_id,omitempty"`
	SourceType            string `json:"source_type,omitempty"`
	SourceID              string `json:"source_id,omitempty"`
	SourceAction          string `json:"source_action,omitempty"`
	ProductCode           string `json:"product_code"`
	OrgID                 string `json:"org_id,omitempty"`
	UserID                string `json:"user_id,omitempty"`
	BillableItemCode      string `json:"billable_item_code"`
	ChargeGroupID         string `json:"charge_group_id,omitempty"`
	ParentEventID         string `json:"parent_event_id,omitempty"`
	EventRole             string `json:"event_role,omitempty"`
	BillingSubjectType    string `json:"billing_subject_type,omitempty"`
	BillingSubjectID      string `json:"billing_subject_id,omitempty"`
	UsageUnits            int64  `json:"usage_units,omitempty"`
	Unit                  string `json:"unit,omitempty"`
	Billable              *bool  `json:"billable,omitempty"`
	BillingProfileKey     string `json:"billing_profile_key,omitempty"`
	CurrencyContext       string `json:"currency_context,omitempty"`
	Dimensions            string `json:"dimensions,omitempty"`
	OccurredAt            string `json:"occurred_at,omitempty"`
	DiscountType          string `json:"discount_type,omitempty"`
	DiscountAmount        int64  `json:"discount_amount,omitempty"`
	CampaignCode          string `json:"campaign_code,omitempty"`
	RewardAmount          int64  `json:"reward_amount,omitempty"`
	RewardAssetCode       string `json:"reward_asset_code,omitempty"`
	RewardSubjectType     string `json:"reward_subject_type,omitempty"`
	RewardSubjectID       string `json:"reward_subject_id,omitempty"`
	ReferralCode          string `json:"referral_code,omitempty"`
	CommissionAmount      int64  `json:"commission_amount,omitempty"`
	CommissionType        string `json:"commission_type,omitempty"`
	CommissionSubjectType string `json:"commission_subject_type,omitempty"`
	CommissionSubjectID   string `json:"commission_subject_id,omitempty"`
}

type FinalizeInput struct {
	FinalizationID string `json:"finalization_id"`
	ReservationID  string `json:"reservation_id"`
	IngestEventInput
}

type FinalizeResult struct {
	Reservation *ResourceReservation `json:"reservation"`
	Event       map[string]any       `json:"event"`
	Settlement  *SettlementRecord    `json:"settlement,omitempty"`
}

type ReverseSettlementInput struct {
	Reason   string `json:"reason,omitempty"`
	Metadata string `json:"metadata,omitempty"`
}

type SettlementRecord struct {
	ID                 string    `json:"id"`
	EventID            string    `json:"event_id"`
	RequestID          string    `json:"request_id"`
	TraceID            string    `json:"trace_id"`
	BillingSubjectType string    `json:"billing_subject_type"`
	BillingSubjectID   string    `json:"billing_subject_id"`
	ProductCode        string    `json:"product_code"`
	BillableItemCode   string    `json:"billable_item_code"`
	BillingProfileID   string    `json:"billing_profile_id"`
	CommercialEntityID string    `json:"commercial_entity_id"`
	MerchantAccountID  string    `json:"merchant_account_id"`
	SettlementMode     string    `json:"settlement_mode"`
	Currency           string    `json:"currency"`
	GrossAmount        int64     `json:"gross_amount"`
	DiscountAmount     int64     `json:"discount_amount"`
	NetAmount          int64     `json:"net_amount"`
	QuotaConsumed      int64     `json:"quota_consumed"`
	CreditsConsumed    int64     `json:"credits_consumed"`
	WalletAssetCode    string    `json:"wallet_asset_code"`
	WalletDebited      int64     `json:"wallet_debited"`
	BillingAmount      int64     `json:"billing_amount"`
	RewardAmount       int64     `json:"reward_amount"`
	CommissionAmount   int64     `json:"commission_amount"`
	Status             string    `json:"status"`
	Snapshot           string    `json:"snapshot"`
	CreatedAt          time.Time `json:"created_at"`
	UpdatedAt          time.Time `json:"updated_at"`
}

type DiscountLedger struct {
	ID                 string    `json:"id"`
	ProductCode        string    `json:"product_code"`
	CampaignCode       string    `json:"campaign_code"`
	DiscountType       string    `json:"discount_type"`
	BillingSubjectType string    `json:"billing_subject_type"`
	BillingSubjectID   string    `json:"billing_subject_id"`
	Currency           string    `json:"currency"`
	Amount             int64     `json:"amount"`
	Status             string    `json:"status"`
	ReferenceType      string    `json:"reference_type"`
	ReferenceID        string    `json:"reference_id"`
	Metadata           string    `json:"metadata"`
	CreatedAt          time.Time `json:"created_at"`
	UpdatedAt          time.Time `json:"updated_at"`
}

type WalletAccount struct {
	ID                 string    `json:"id"`
	BillingSubjectType string    `json:"billing_subject_type"`
	BillingSubjectID   string    `json:"billing_subject_id"`
	AssetCode          string    `json:"asset_code"`
	AssetType          string    `json:"asset_type"`
	Balance            int64     `json:"balance"`
	Status             string    `json:"status"`
	Metadata           string    `json:"metadata"`
	CreatedAt          time.Time `json:"created_at"`
	UpdatedAt          time.Time `json:"updated_at"`
}

type WalletAssetSummary struct {
	AssetCode        string     `json:"asset_code"`
	AssetType        string     `json:"asset_type"`
	LifecycleType    string     `json:"lifecycle_type"`
	AccountBalance   int64      `json:"account_balance"`
	AvailableBalance int64      `json:"available_balance"`
	ExpiringBalance  int64      `json:"expiring_balance"`
	NextExpiresAt    *time.Time `json:"next_expires_at,omitempty"`
}

type WalletSummary struct {
	BillingSubjectType string               `json:"billing_subject_type"`
	BillingSubjectID   string               `json:"billing_subject_id"`
	ProductCode        string               `json:"product_code"`
	TotalBalance       int64                `json:"total_balance"`
	PermanentBalance   int64                `json:"permanent_balance"`
	RewardBalance      int64                `json:"reward_balance"`
	AllowanceBalance   int64                `json:"allowance_balance"`
	Assets             []WalletAssetSummary `json:"assets"`
}

type AssetDefinition struct {
	AssetCode         string    `json:"asset_code"`
	ProductCode       string    `json:"product_code"`
	AssetType         string    `json:"asset_type"`
	LifecycleType     string    `json:"lifecycle_type"`
	DefaultExpireDays int       `json:"default_expire_days"`
	ResetCycle        string    `json:"reset_cycle"`
	Status            string    `json:"status"`
	Description       string    `json:"description"`
	Metadata          string    `json:"metadata"`
	CreatedAt         time.Time `json:"created_at"`
	UpdatedAt         time.Time `json:"updated_at"`
}

type Product struct {
	ID        string    `json:"id"`
	Code      string    `json:"code"`
	Name      string    `json:"name"`
	Status    string    `json:"status"`
	OwnerTeam string    `json:"owner_team"`
	Metadata  string    `json:"metadata"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type SKU struct {
	ID          string    `json:"id"`
	ProductID   string    `json:"product_id"`
	Code        string    `json:"code"`
	Name        string    `json:"name"`
	SKUType     string    `json:"sku_type"`
	BillingMode string    `json:"billing_mode"`
	Currency    string    `json:"currency"`
	ListPrice   int64     `json:"list_price"`
	Status      string    `json:"status"`
	Metadata    string    `json:"metadata"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

type CommercialPackage struct {
	ID          string    `json:"id"`
	ProductID   string    `json:"product_id"`
	Code        string    `json:"code"`
	Name        string    `json:"name"`
	PackageType string    `json:"package_type"`
	Status      string    `json:"status"`
	Metadata    string    `json:"metadata"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

type BillableItem struct {
	ID              string    `json:"id"`
	ProductID       string    `json:"product_id"`
	Code            string    `json:"code"`
	Name            string    `json:"name"`
	MeterUnit       string    `json:"meter_unit"`
	BillingScope    string    `json:"billing_scope"`
	SettlementMode  string    `json:"settlement_mode"`
	PricingBehavior string    `json:"pricing_behavior"`
	Status          string    `json:"status"`
	Metadata        string    `json:"metadata"`
	CreatedAt       time.Time `json:"created_at"`
	UpdatedAt       time.Time `json:"updated_at"`
}

type RateCard struct {
	ID            string     `json:"id"`
	ProductID     string     `json:"product_id"`
	Code          string     `json:"code"`
	TargetType    string     `json:"target_type"`
	TargetID      string     `json:"target_id"`
	PriceModel    string     `json:"price_model"`
	Currency      string     `json:"currency"`
	PriceConfig   string     `json:"price_config"`
	EffectiveFrom *time.Time `json:"effective_from,omitempty"`
	EffectiveTo   *time.Time `json:"effective_to,omitempty"`
	Version       int        `json:"version"`
	Status        string     `json:"status"`
	Metadata      string     `json:"metadata"`
	CreatedAt     time.Time  `json:"created_at"`
	UpdatedAt     time.Time  `json:"updated_at"`
}

type AllowancePolicy struct {
	ID                 string     `json:"id"`
	ProductCode        string     `json:"product_code"`
	BillingSubjectType string     `json:"billing_subject_type"`
	BillingSubjectID   string     `json:"billing_subject_id"`
	AssetCode          string     `json:"asset_code"`
	Amount             int64      `json:"amount"`
	ResetCycle         string     `json:"reset_cycle"`
	Status             string     `json:"status"`
	EffectiveFrom      *time.Time `json:"effective_from,omitempty"`
	EffectiveTo        *time.Time `json:"effective_to,omitempty"`
	Metadata           string     `json:"metadata"`
	CreatedAt          time.Time  `json:"created_at"`
	UpdatedAt          time.Time  `json:"updated_at"`
}

type QuotaGrantPolicy struct {
	ID               string    `json:"id"`
	ProductCode      string    `json:"product_code"`
	PackageCode      string    `json:"package_code"`
	BillableItemCode string    `json:"billable_item_code"`
	GrantMode        string    `json:"grant_mode"`
	Units            int64     `json:"units"`
	ResetCycle       string    `json:"reset_cycle"`
	Status           string    `json:"status"`
	Metadata         string    `json:"metadata"`
	CreatedAt        time.Time `json:"created_at"`
	UpdatedAt        time.Time `json:"updated_at"`
}

type PackageCapabilityPolicy struct {
	ID             string    `json:"id"`
	ProductCode    string    `json:"product_code"`
	PackageCode    string    `json:"package_code"`
	CapabilityCode string    `json:"capability_code"`
	GrantValue     string    `json:"grant_value"`
	Status         string    `json:"status"`
	Metadata       string    `json:"metadata"`
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`
}

type CapabilityGrant struct {
	ID                 string     `json:"id"`
	ProductCode        string     `json:"product_code"`
	BillingSubjectType string     `json:"billing_subject_type"`
	BillingSubjectID   string     `json:"billing_subject_id"`
	CapabilityCode     string     `json:"capability_code"`
	GrantValue         string     `json:"grant_value"`
	SourceType         string     `json:"source_type"`
	SourceID           string     `json:"source_id"`
	Status             string     `json:"status"`
	ExpiresAt          *time.Time `json:"expires_at,omitempty"`
	Metadata           string     `json:"metadata"`
	CreatedAt          time.Time  `json:"created_at"`
	UpdatedAt          time.Time  `json:"updated_at"`
}

type ResolveCapabilityResult struct {
	ProductCode        string           `json:"product_code"`
	BillingSubjectType string           `json:"billing_subject_type"`
	BillingSubjectID   string           `json:"billing_subject_id"`
	CapabilityCode     string           `json:"capability_code"`
	GrantValue         string           `json:"grant_value"`
	Grant              *CapabilityGrant `json:"grant,omitempty"`
}

type OfferingsView struct {
	Product           *Product            `json:"product"`
	SKUs              []SKU               `json:"skus"`
	Packages          []CommercialPackage `json:"packages"`
	BillableItems     []BillableItem      `json:"billable_items"`
	RateCards         []RateCard          `json:"rate_cards"`
	AssetDefinitions  []AssetDefinition   `json:"asset_definitions"`
	AllowancePolicies []AllowancePolicy   `json:"allowance_policies"`
}

type WalletBucket struct {
	ID                 string     `json:"id"`
	WalletAccountID    string     `json:"wallet_account_id"`
	BillingSubjectType string     `json:"billing_subject_type"`
	BillingSubjectID   string     `json:"billing_subject_id"`
	AssetCode          string     `json:"asset_code"`
	AssetType          string     `json:"asset_type"`
	LifecycleType      string     `json:"lifecycle_type"`
	SourceType         string     `json:"source_type"`
	SourceID           string     `json:"source_id"`
	CycleKey           string     `json:"cycle_key"`
	Balance            int64      `json:"balance"`
	ExpiresAt          *time.Time `json:"expires_at,omitempty"`
	Status             string     `json:"status"`
	Metadata           string     `json:"metadata"`
	CreatedAt          time.Time  `json:"created_at"`
	UpdatedAt          time.Time  `json:"updated_at"`
}

type PostWalletLedgerInput struct {
	BillingSubjectType string `json:"billing_subject_type"`
	BillingSubjectID   string `json:"billing_subject_id"`
	AssetCode          string `json:"asset_code"`
	AssetType          string `json:"asset_type,omitempty"`
	Direction          string `json:"direction"`
	Amount             int64  `json:"amount"`
	Reason             string `json:"reason,omitempty"`
	ReferenceType      string `json:"reference_type,omitempty"`
	ReferenceID        string `json:"reference_id,omitempty"`
	Status             string `json:"status,omitempty"`
	Metadata           string `json:"metadata,omitempty"`
	ExpiresAt          string `json:"expires_at,omitempty"`
	CycleKey           string `json:"cycle_key,omitempty"`
}

type GrantCycleAllowanceInput struct {
	BillingSubjectType string `json:"billing_subject_type"`
	BillingSubjectID   string `json:"billing_subject_id"`
	AssetCode          string `json:"asset_code"`
	CycleKey           string `json:"cycle_key"`
	Amount             int64  `json:"amount"`
	Metadata           string `json:"metadata,omitempty"`
}

type GrantQuotaInput struct {
	BillingSubjectType string `json:"billing_subject_type"`
	BillingSubjectID   string `json:"billing_subject_id"`
	BillableItemCode   string `json:"billable_item_code"`
	Units              int64  `json:"units"`
	Reason             string `json:"reason,omitempty"`
	ReferenceID        string `json:"reference_id,omitempty"`
}

type GrantCapabilityInput struct {
	ProductCode        string `json:"product_code"`
	BillingSubjectType string `json:"billing_subject_type"`
	BillingSubjectID   string `json:"billing_subject_id"`
	CapabilityCode     string `json:"capability_code"`
	GrantValue         string `json:"grant_value"`
	SourceType         string `json:"source_type,omitempty"`
	SourceID           string `json:"source_id,omitempty"`
	Metadata           string `json:"metadata,omitempty"`
}

type postWalletLedgerResult struct {
	Ledger  *WalletLedger  `json:"ledger"`
	Account *WalletAccount `json:"account"`
}

type grantCycleAllowanceResult struct {
	Bucket  *WalletBucket  `json:"bucket"`
	Account *WalletAccount `json:"account"`
}

type WalletLedger struct {
	ID                 string    `json:"id"`
	WalletAccountID    string    `json:"wallet_account_id"`
	BillingSubjectType string    `json:"billing_subject_type"`
	BillingSubjectID   string    `json:"billing_subject_id"`
	AssetCode          string    `json:"asset_code"`
	Direction          string    `json:"direction"`
	Amount             int64     `json:"amount"`
	Reason             string    `json:"reason"`
	ReferenceType      string    `json:"reference_type"`
	ReferenceID        string    `json:"reference_id"`
	Status             string    `json:"status"`
	Metadata           string    `json:"metadata"`
	CreatedAt          time.Time `json:"created_at"`
}

type RewardLedger struct {
	ID                     string    `json:"id"`
	ProductCode            string    `json:"product_code"`
	CampaignCode           string    `json:"campaign_code"`
	RewardType             string    `json:"reward_type"`
	BeneficiarySubjectType string    `json:"beneficiary_subject_type"`
	BeneficiarySubjectID   string    `json:"beneficiary_subject_id"`
	AssetCode              string    `json:"asset_code"`
	Amount                 int64     `json:"amount"`
	Status                 string    `json:"status"`
	ReferenceType          string    `json:"reference_type"`
	ReferenceID            string    `json:"reference_id"`
	Metadata               string    `json:"metadata"`
	CreatedAt              time.Time `json:"created_at"`
	UpdatedAt              time.Time `json:"updated_at"`
}

type CreateRewardInput struct {
	ProductCode            string `json:"product_code,omitempty"`
	CampaignCode           string `json:"campaign_code,omitempty"`
	RewardType             string `json:"reward_type"`
	BeneficiarySubjectType string `json:"beneficiary_subject_type"`
	BeneficiarySubjectID   string `json:"beneficiary_subject_id"`
	AssetCode              string `json:"asset_code,omitempty"`
	Amount                 int64  `json:"amount"`
	Status                 string `json:"status,omitempty"`
	ReferenceType          string `json:"reference_type,omitempty"`
	ReferenceID            string `json:"reference_id,omitempty"`
	Metadata               string `json:"metadata,omitempty"`
}

type CreateAssetDefinitionInput struct {
	AssetCode         string `json:"asset_code"`
	ProductCode       string `json:"product_code,omitempty"`
	AssetType         string `json:"asset_type"`
	LifecycleType     string `json:"lifecycle_type"`
	DefaultExpireDays int    `json:"default_expire_days,omitempty"`
	ResetCycle        string `json:"reset_cycle,omitempty"`
	Status            string `json:"status,omitempty"`
	Description       string `json:"description,omitempty"`
	Metadata          string `json:"metadata,omitempty"`
}

type CommissionLedger struct {
	ID                     string     `json:"id"`
	ProductCode            string     `json:"product_code"`
	CommissionType         string     `json:"commission_type"`
	BeneficiarySubjectType string     `json:"beneficiary_subject_type"`
	BeneficiarySubjectID   string     `json:"beneficiary_subject_id"`
	SettlementSubjectType  string     `json:"settlement_subject_type"`
	SettlementSubjectID    string     `json:"settlement_subject_id"`
	Currency               string     `json:"currency"`
	Amount                 int64      `json:"amount"`
	Status                 string     `json:"status"`
	ReferenceType          string     `json:"reference_type"`
	ReferenceID            string     `json:"reference_id"`
	RedeemedRewardID       string     `json:"redeemed_reward_id"`
	RedeemedAt             *time.Time `json:"redeemed_at,omitempty"`
	Metadata               string     `json:"metadata"`
	CreatedAt              time.Time  `json:"created_at"`
	UpdatedAt              time.Time  `json:"updated_at"`
}

type ReferralProgram struct {
	ID                    string     `json:"id"`
	ProductCode           string     `json:"product_code"`
	ProgramCode           string     `json:"program_code"`
	Name                  string     `json:"name"`
	Status                string     `json:"status"`
	TriggerType           string     `json:"trigger_type"`
	CommissionPolicy      string     `json:"commission_policy"`
	CommissionCurrency    string     `json:"commission_currency"`
	CommissionFixedAmount int64      `json:"commission_fixed_amount"`
	CommissionRateBps     int64      `json:"commission_rate_bps"`
	SettlementDelayDays   int        `json:"settlement_delay_days"`
	AllowRepeat           bool       `json:"allow_repeat"`
	EffectiveFrom         *time.Time `json:"effective_from,omitempty"`
	EffectiveTo           *time.Time `json:"effective_to,omitempty"`
	Metadata              string     `json:"metadata"`
	CreatedAt             time.Time  `json:"created_at"`
	UpdatedAt             time.Time  `json:"updated_at"`
}

type ReferralCode struct {
	ID                  string    `json:"id"`
	ProgramID           string    `json:"program_id"`
	ProductCode         string    `json:"product_code"`
	Code                string    `json:"code"`
	PromoterSubjectType string    `json:"promoter_subject_type"`
	PromoterSubjectID   string    `json:"promoter_subject_id"`
	Status              string    `json:"status"`
	Metadata            string    `json:"metadata"`
	CreatedAt           time.Time `json:"created_at"`
	UpdatedAt           time.Time `json:"updated_at"`
}

type ReferralConversion struct {
	ID                    string    `json:"id"`
	ProgramID             string    `json:"program_id"`
	ReferralCodeID        string    `json:"referral_code_id"`
	ProductCode           string    `json:"product_code"`
	TriggerType           string    `json:"trigger_type"`
	PromoterSubjectType   string    `json:"promoter_subject_type"`
	PromoterSubjectID     string    `json:"promoter_subject_id"`
	ReferredSubjectType   string    `json:"referred_subject_type"`
	ReferredSubjectID     string    `json:"referred_subject_id"`
	SettlementSubjectType string    `json:"settlement_subject_type"`
	SettlementSubjectID   string    `json:"settlement_subject_id"`
	ReferenceType         string    `json:"reference_type"`
	ReferenceID           string    `json:"reference_id"`
	CommissionCurrency    string    `json:"commission_currency"`
	CommissionAmount      int64     `json:"commission_amount"`
	CommissionLedgerID    string    `json:"commission_ledger_id"`
	Status                string    `json:"status"`
	Metadata              string    `json:"metadata"`
	CreatedAt             time.Time `json:"created_at"`
	UpdatedAt             time.Time `json:"updated_at"`
}

type CreateReferralCodeInput struct {
	ProgramCode         string `json:"program_code"`
	Code                string `json:"code,omitempty"`
	PromoterSubjectType string `json:"promoter_subject_type"`
	PromoterSubjectID   string `json:"promoter_subject_id"`
	Status              string `json:"status,omitempty"`
	Metadata            string `json:"metadata,omitempty"`
}

type CreateReferralProgramInput struct {
	ProductCode           string `json:"product_code"`
	ProgramCode           string `json:"program_code"`
	Name                  string `json:"name"`
	TriggerType           string `json:"trigger_type"`
	CommissionPolicy      string `json:"commission_policy"`
	CommissionCurrency    string `json:"commission_currency,omitempty"`
	CommissionFixedAmount int64  `json:"commission_fixed_amount,omitempty"`
	CommissionRateBps     int64  `json:"commission_rate_bps,omitempty"`
	SettlementDelayDays   int    `json:"settlement_delay_days,omitempty"`
	AllowRepeat           bool   `json:"allow_repeat"`
	Status                string `json:"status,omitempty"`
	Metadata              string `json:"metadata,omitempty"`
}

type CreateReferralConversionInput struct {
	ReferralCode          string `json:"referral_code"`
	ProductCode           string `json:"product_code"`
	TriggerType           string `json:"trigger_type"`
	ReferredSubjectType   string `json:"referred_subject_type"`
	ReferredSubjectID     string `json:"referred_subject_id"`
	SettlementSubjectType string `json:"settlement_subject_type,omitempty"`
	SettlementSubjectID   string `json:"settlement_subject_id,omitempty"`
	ReferenceType         string `json:"reference_type"`
	ReferenceID           string `json:"reference_id"`
	CommissionBaseAmount  int64  `json:"commission_base_amount,omitempty"`
	CommissionCurrency    string `json:"commission_currency,omitempty"`
	Metadata              string `json:"metadata,omitempty"`
}

type ResolvedReferralCode struct {
	Code                  string         `json:"code"`
	ProductCode           string         `json:"product_code"`
	ProgramID             string         `json:"program_id"`
	ProgramCode           string         `json:"program_code"`
	ProgramName           string         `json:"program_name"`
	TriggerType           string         `json:"trigger_type"`
	CommissionPolicy      string         `json:"commission_policy"`
	CommissionCurrency    string         `json:"commission_currency"`
	CommissionFixedAmount int64          `json:"commission_fixed_amount"`
	CommissionRateBps     int64          `json:"commission_rate_bps"`
	SettlementDelayDays   int            `json:"settlement_delay_days"`
	AllowRepeat           bool           `json:"allow_repeat"`
	RewardPolicyDesc      string         `json:"reward_policy_desc"`
	PromoterSubjectType   string         `json:"promoter_subject_type"`
	PromoterSubjectID     string         `json:"promoter_subject_id"`
	Status                string         `json:"status"`
	Metadata              map[string]any `json:"metadata,omitempty"`
}

type ResolveRouteInput struct {
	OrganizationID    string `json:"organization_id,omitempty"`
	BillingProfileKey string `json:"billing_profile_key,omitempty"`
	Channel           string `json:"channel,omitempty"`
	Currency          string `json:"currency,omitempty"`
	Region            string `json:"region,omitempty"`
	MerchantRouteHint string `json:"merchant_route_hint,omitempty"`
	PaymentScene      string `json:"payment_scene,omitempty"`
	OrderType         string `json:"order_type,omitempty"`
}

type ResolveRouteResult struct {
	BillingProfileID    string `json:"billing_profile_id"`
	BillingProfileCode  string `json:"billing_profile_code"`
	CommercialEntityID  string `json:"commercial_entity_id"`
	MerchantAccountID   string `json:"merchant_account_id"`
	SettlementAccountID string `json:"settlement_account_id"`
	RoutingPolicyID     string `json:"routing_policy_id"`
	ResolutionReason    string `json:"resolution_reason"`
	RouteSnapshot       string `json:"route_snapshot"`
}

type RedeemCommissionsInput struct {
	ProductCode            string   `json:"product_code"`
	BeneficiarySubjectType string   `json:"beneficiary_subject_type"`
	BeneficiarySubjectID   string   `json:"beneficiary_subject_id"`
	AssetCode              string   `json:"asset_code"`
	CommissionIDs          []string `json:"commission_ids,omitempty"`
	Metadata               string   `json:"metadata,omitempty"`
}

type RedeemCommissionsResult struct {
	RewardLedgerID string             `json:"reward_ledger_id"`
	AssetCode      string             `json:"asset_code"`
	TotalAmount    int64              `json:"total_amount"`
	Commissions    []CommissionLedger `json:"commissions"`
}
