package platform

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"menu-service/internal/config"
)

type Client struct {
	baseURL     string
	secret      string
	serviceName string
	http        *http.Client
}

func New(cfg config.PlatformConfig) *Client {
	return &Client{
		baseURL:     strings.TrimRight(cfg.BaseURL, "/"),
		secret:      cfg.InternalServiceSecret,
		serviceName: defaultString(cfg.ServiceName, "v-menu-backend"),
		http:        &http.Client{Timeout: cfg.Timeout},
	}
}

func (c *Client) BaseURL() string          { return c.baseURL }
func (c *Client) InternalSecret() string   { return c.secret }
func (c *Client) HTTPClient() *http.Client { return c.http }
func (c *Client) ServiceName() string      { return c.serviceName }

func (c *Client) InternalURL(path string) string {
	return fmt.Sprintf("%s/internal/v1/%s", c.baseURL, strings.TrimLeft(path, "/"))
}

func (c *Client) PublicURL(path string) string {
	return fmt.Sprintf("%s/api/v1/%s", c.baseURL, strings.TrimLeft(path, "/"))
}

func DefaultTimeout() time.Duration { return 5 * time.Second }

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

type RuntimeProviderDefinition struct {
	ID            string    `json:"id"`
	Code          string    `json:"code"`
	Name          string    `json:"name"`
	ProviderType  string    `json:"provider_type"`
	Mode          string    `json:"mode"`
	CredentialRef string    `json:"credential_ref"`
	Capabilities  string    `json:"capabilities"`
	Status        string    `json:"status"`
	Metadata      string    `json:"metadata"`
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`
}

type CreateRuntimeJobInput struct {
	ProductCode     string `json:"product_code"`
	TaskType        string `json:"task_type"`
	ProviderCode    string `json:"provider_code,omitempty"`
	ProviderMode    string `json:"provider_mode"`
	OrganizationID  string `json:"organization_id"`
	UserID          string `json:"user_id,omitempty"`
	SourceType      string `json:"source_type"`
	SourceID        string `json:"source_id"`
	IdempotencyKey  string `json:"idempotency_key,omitempty"`
	ChargeSessionID string `json:"charge_session_id,omitempty"`
	InputManifest   string `json:"input_manifest,omitempty"`
	RouteSnapshot   string `json:"route_snapshot,omitempty"`
	Metadata        string `json:"metadata,omitempty"`
	Priority        int    `json:"priority,omitempty"`
	MaxAttempts     int    `json:"max_attempts,omitempty"`
	TimeoutSeconds  int    `json:"timeout_seconds,omitempty"`
}

type UpdateRuntimeJobInput struct {
	Status         string `json:"status,omitempty"`
	Stage          string `json:"stage,omitempty"`
	StageMessage   string `json:"stage_message,omitempty"`
	ProviderJobID  string `json:"provider_job_id,omitempty"`
	ErrorClass     string `json:"error_class,omitempty"`
	ErrorCode      string `json:"error_code,omitempty"`
	ErrorMessage   string `json:"error_message,omitempty"`
	OutputManifest string `json:"output_manifest,omitempty"`
	RouteSnapshot  string `json:"route_snapshot,omitempty"`
	Metadata       string `json:"metadata,omitempty"`
	AttemptCount   *int   `json:"attempt_count,omitempty"`
	NextRetryAt    string `json:"next_retry_at,omitempty"`
}

type RuntimeJob struct {
	ID              string     `json:"id"`
	ProductCode     string     `json:"product_code"`
	TaskType        string     `json:"task_type"`
	ProviderCode    string     `json:"provider_code"`
	ProviderMode    string     `json:"provider_mode"`
	ProviderJobID   string     `json:"provider_job_id"`
	OrganizationID  string     `json:"organization_id"`
	UserID          string     `json:"user_id"`
	SourceType      string     `json:"source_type"`
	SourceID        string     `json:"source_id"`
	IdempotencyKey  *string    `json:"idempotency_key,omitempty"`
	ChargeSessionID string     `json:"charge_session_id"`
	Status          string     `json:"status"`
	Stage           string     `json:"stage"`
	StageMessage    string     `json:"stage_message"`
	ErrorClass      string     `json:"error_class"`
	ErrorCode       string     `json:"error_code"`
	ErrorMessage    string     `json:"error_message"`
	InputManifest   string     `json:"input_manifest"`
	OutputManifest  string     `json:"output_manifest"`
	RouteSnapshot   string     `json:"route_snapshot"`
	Metadata        string     `json:"metadata"`
	Priority        int        `json:"priority"`
	AttemptCount    int        `json:"attempt_count"`
	MaxAttempts     int        `json:"max_attempts"`
	TimeoutAt       *time.Time `json:"timeout_at"`
	NextRetryAt     *time.Time `json:"next_retry_at"`
	CompletedAt     *time.Time `json:"completed_at"`
	CanceledAt      *time.Time `json:"canceled_at"`
	CreatedAt       time.Time  `json:"created_at"`
	UpdatedAt       time.Time  `json:"updated_at"`
}

type RuntimeAttempt struct {
	ID               string     `json:"id"`
	RuntimeJobID     string     `json:"runtime_job_id"`
	AttemptNo        int        `json:"attempt_no"`
	Status           string     `json:"status"`
	ErrorClass       string     `json:"error_class"`
	ErrorCode        string     `json:"error_code"`
	ErrorMessage     string     `json:"error_message"`
	ProviderCode     string     `json:"provider_code"`
	ProviderMode     string     `json:"provider_mode"`
	ProviderRequest  string     `json:"provider_request"`
	ProviderResponse string     `json:"provider_response"`
	StartedAt        *time.Time `json:"started_at"`
	EndedAt          *time.Time `json:"ended_at"`
	CreatedAt        time.Time  `json:"created_at"`
	UpdatedAt        time.Time  `json:"updated_at"`
}

type RuntimeJobDetail struct {
	Job      *RuntimeJob      `json:"job"`
	Attempts []RuntimeAttempt `json:"attempts"`
}

type CreateChargeSessionInput struct {
	SourceType         string `json:"source_type"`
	SourceID           string `json:"source_id"`
	ProductCode        string `json:"product_code"`
	OrganizationID     string `json:"organization_id"`
	UserID             string `json:"user_id,omitempty"`
	BillingSubjectType string `json:"billing_subject_type"`
	BillingSubjectID   string `json:"billing_subject_id"`
	BillableItemCode   string `json:"billable_item_code"`
	ResourceType       string `json:"resource_type"`
	ReservationKey     string `json:"reservation_key,omitempty"`
	EstimatedUnits     int64  `json:"estimated_units,omitempty"`
	RouteSnapshot      string `json:"route_snapshot,omitempty"`
	Metadata           string `json:"metadata,omitempty"`
}

type UpdateChargeSessionInput struct {
	Status         string `json:"status,omitempty"`
	ReservationID  string `json:"reservation_id,omitempty"`
	FinalizationID string `json:"finalization_id,omitempty"`
	EventID        string `json:"event_id,omitempty"`
	SettlementID   string `json:"settlement_id,omitempty"`
	FinalUnits     *int64 `json:"final_units,omitempty"`
	RouteSnapshot  string `json:"route_snapshot,omitempty"`
	Metadata       string `json:"metadata,omitempty"`
}

type ChargeSession struct {
	ID                 string     `json:"id"`
	SourceType         string     `json:"source_type"`
	SourceID           string     `json:"source_id"`
	ProductCode        string     `json:"product_code"`
	OrganizationID     string     `json:"organization_id"`
	UserID             string     `json:"user_id"`
	BillingSubjectType string     `json:"billing_subject_type"`
	BillingSubjectID   string     `json:"billing_subject_id"`
	BillableItemCode   string     `json:"billable_item_code"`
	ResourceType       string     `json:"resource_type"`
	Status             string     `json:"status"`
	ReservationKey     string     `json:"reservation_key"`
	ReservationID      string     `json:"reservation_id"`
	FinalizationID     string     `json:"finalization_id"`
	EventID            string     `json:"event_id"`
	SettlementID       string     `json:"settlement_id"`
	EstimatedUnits     int64      `json:"estimated_units"`
	FinalUnits         int64      `json:"final_units"`
	RouteSnapshot      string     `json:"route_snapshot"`
	Metadata           string     `json:"metadata"`
	ReservedAt         *time.Time `json:"reserved_at"`
	FinalizedAt        *time.Time `json:"finalized_at"`
	ReleasedAt         *time.Time `json:"released_at"`
	CreatedAt          time.Time  `json:"created_at"`
	UpdatedAt          time.Time  `json:"updated_at"`
}

type UploadAssetInput struct {
	ProductCode string `json:"product_code"`
	Category    string `json:"category"`
	FileName    string `json:"file_name"`
	MimeType    string `json:"mime_type"`
	Payload     string `json:"payload"`
}

type StoredAsset struct {
	StorageKey string `json:"storage_key"`
	MimeType   string `json:"mime_type"`
	FileSize   int64  `json:"file_size"`
}

type platformItemsResponse[T any] struct {
	Items []T `json:"items"`
}

type platformError struct {
	Status    int
	Code      int
	Message   string
	ErrorCode string
	ErrorHint string
	Err       string
	RequestID string
	Fields    []string
}

func (e *platformError) Error() string {
	if e == nil {
		return ""
	}
	if e.Err != "" {
		if len(e.Fields) > 0 {
			return fmt.Sprintf("platform request failed: status=%d code=%d message=%s error_code=%s error=%s fields=%s request_id=%s", e.Status, e.Code, e.Message, e.ErrorCode, e.Err, strings.Join(e.Fields, ","), e.RequestID)
		}
		return fmt.Sprintf("platform request failed: status=%d code=%d message=%s error_code=%s error=%s request_id=%s", e.Status, e.Code, e.Message, e.ErrorCode, e.Err, e.RequestID)
	}
	if len(e.Fields) > 0 {
		return fmt.Sprintf("platform request failed: status=%d code=%d message=%s error_code=%s fields=%s request_id=%s", e.Status, e.Code, e.Message, e.ErrorCode, strings.Join(e.Fields, ","), e.RequestID)
	}
	return fmt.Sprintf("platform request failed: status=%d code=%d message=%s error_code=%s request_id=%s", e.Status, e.Code, e.Message, e.ErrorCode, e.RequestID)
}

func (c *Client) GetAccessContext(userID, orgID string) (*PlatformAccessData, error) {
	return doGet[PlatformAccessData](c, fmt.Sprintf("/access/users/%s/orgs/%s", userID, orgID))
}

func (c *Client) GetUserProfile(userID, orgID string) (*PlatformUserProfile, error) {
	path := fmt.Sprintf("/users/%s/profile", userID)
	if orgID != "" {
		path = withQuery(path, map[string]string{"org_id": orgID})
	}
	return doGet[PlatformUserProfile](c, path)
}

func (c *Client) UpdateUserProfile(userID string, input UpdateUserProfileInput) (*PlatformUserProfile, error) {
	return doPut[UpdateUserProfileInput, PlatformUserProfile](c, fmt.Sprintf("/users/%s/profile", userID), input)
}

func (c *Client) UpdateOrganizationProfile(orgID string, input UpdateOrganizationProfileInput) error {
	_, err := doPut[UpdateOrganizationProfileInput, map[string]any](c, fmt.Sprintf("/orgs/%s/profile", orgID), input)
	return err
}

func (c *Client) Register(input AuthRegisterInput) (*PlatformAuthResult, error) {
	return doPublicPost[AuthRegisterInput, PlatformAuthResult](c, "/auth/register", input)
}

func (c *Client) Login(input AuthLoginInput) (*PlatformAuthResult, error) {
	return doPublicPost[AuthLoginInput, PlatformAuthResult](c, "/auth/login", input)
}

func (c *Client) ReserveResources(input ReserveInput) (*ResourceReservation, error) {
	return doPost[ReserveInput, ResourceReservation](c, "/controls/reservations", input)
}

func (c *Client) CommitReservation(reservationID string) (*ResourceReservation, error) {
	return doPost[any, ResourceReservation](c, fmt.Sprintf("/controls/reservations/%s/commit", reservationID), nil)
}

func (c *Client) ReleaseReservation(reservationID string) (*ResourceReservation, error) {
	return doPost[any, ResourceReservation](c, fmt.Sprintf("/controls/reservations/%s/release", reservationID), nil)
}

func (c *Client) IngestMeteringEvent(input IngestEventInput) error {
	_, err := doPost[IngestEventInput, map[string]any](c, "/metering/events", input)
	return err
}

func (c *Client) FinalizeMetering(input FinalizeInput) (*FinalizeResult, error) {
	return doPost[FinalizeInput, FinalizeResult](c, "/metering/finalizations", input)
}

func (c *Client) GetSettlement(eventID string) (*SettlementRecord, error) {
	return doGet[SettlementRecord](c, fmt.Sprintf("/metering/settlements/%s", eventID))
}

func (c *Client) ListSettlements(subjectType, subjectID, productCode string) ([]SettlementRecord, error) {
	path := withQuery("/metering/settlements", map[string]string{
		"billing_subject_type": subjectType,
		"billing_subject_id":   subjectID,
		"product_code":         productCode,
	})
	out, err := doGet[platformItemsResponse[SettlementRecord]](c, path)
	if err != nil {
		return nil, err
	}
	return out.Items, nil
}

func (c *Client) ReverseSettlement(eventID string, input ReverseSettlementInput) (*SettlementRecord, error) {
	return doPost[ReverseSettlementInput, SettlementRecord](c, fmt.Sprintf("/metering/settlements/%s/reverse", eventID), input)
}

func (c *Client) ListDiscounts(subjectType, subjectID, productCode string) ([]DiscountLedger, error) {
	path := withQuery("/metering/discounts", map[string]string{
		"billing_subject_type": subjectType,
		"billing_subject_id":   subjectID,
		"product_code":         productCode,
	})
	out, err := doGet[platformItemsResponse[DiscountLedger]](c, path)
	if err != nil {
		return nil, err
	}
	return out.Items, nil
}

func (c *Client) ResolveCommercialRoute(input ResolveRouteInput) (*ResolveRouteResult, error) {
	return doPost[ResolveRouteInput, ResolveRouteResult](c, "/commercial/route/resolve", input)
}

func (c *Client) CreateRuntimeJob(input CreateRuntimeJobInput) (*RuntimeJob, error) {
	return doPost[CreateRuntimeJobInput, RuntimeJob](c, "/runtime/jobs", input)
}

func (c *Client) GetRuntimeJob(runtimeJobID string) (*RuntimeJobDetail, error) {
	return doGet[RuntimeJobDetail](c, fmt.Sprintf("/runtime/jobs/%s", runtimeJobID))
}

func (c *Client) UpdateRuntimeJob(runtimeJobID string, input UpdateRuntimeJobInput) (*RuntimeJob, error) {
	return doPut[UpdateRuntimeJobInput, RuntimeJob](c, fmt.Sprintf("/runtime/jobs/%s", runtimeJobID), input)
}

func (c *Client) CancelRuntimeJob(runtimeJobID string) (*RuntimeJob, error) {
	return doPost[any, RuntimeJob](c, fmt.Sprintf("/runtime/jobs/%s/cancel", runtimeJobID), nil)
}

func (c *Client) CreateChargeSession(input CreateChargeSessionInput) (*ChargeSession, error) {
	return doPost[CreateChargeSessionInput, ChargeSession](c, "/runtime/charge-sessions", input)
}

func (c *Client) GetChargeSession(chargeSessionID string) (*ChargeSession, error) {
	return doGet[ChargeSession](c, fmt.Sprintf("/runtime/charge-sessions/%s", chargeSessionID))
}

func (c *Client) UpdateChargeSession(chargeSessionID string, input UpdateChargeSessionInput) (*ChargeSession, error) {
	return doPut[UpdateChargeSessionInput, ChargeSession](c, fmt.Sprintf("/runtime/charge-sessions/%s", chargeSessionID), input)
}

func (c *Client) UploadAsset(input UploadAssetInput) (*StoredAsset, error) {
	return doPost[UploadAssetInput, StoredAsset](c, "/storage/assets", input)
}

func (c *Client) DownloadAsset(storageKey string) (io.ReadCloser, http.Header, error) {
	path := withQuery("/storage/assets/content", map[string]string{"storage_key": storageKey})
	req, err := http.NewRequest(http.MethodGet, c.InternalURL(path), nil)
	if err != nil {
		return nil, nil, err
	}
	for key, value := range c.buildHeaders(http.MethodGet, path, nil) {
		req.Header.Set(key, value)
	}
	resp, err := c.http.Do(req)
	if err != nil {
		return nil, nil, err
	}
	if resp.StatusCode >= 400 {
		defer resp.Body.Close()
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 2048))
		return nil, nil, fmt.Errorf("platform asset download failed: status=%d body=%s", resp.StatusCode, strings.TrimSpace(string(body)))
	}
	return resp.Body, resp.Header, nil
}

func (c *Client) ListWalletAccounts(subjectType, subjectID string) ([]WalletAccount, error) {
	path := withQuery("/wallet/accounts", map[string]string{
		"billing_subject_type": subjectType,
		"billing_subject_id":   subjectID,
	})
	out, err := doGet[platformItemsResponse[WalletAccount]](c, path)
	if err != nil {
		return nil, err
	}
	return out.Items, nil
}

func (c *Client) GetWalletSummary(subjectType, subjectID, productCode string) (*WalletSummary, error) {
	path := withQuery("/wallet/summary", map[string]string{
		"billing_subject_type": subjectType,
		"billing_subject_id":   subjectID,
		"product_code":         productCode,
	})
	return doGet[WalletSummary](c, path)
}

func (c *Client) ListAssetDefinitions(productCode, lifecycleType, status string) ([]AssetDefinition, error) {
	path := withQuery("/wallet/assets", map[string]string{
		"product_code":   productCode,
		"lifecycle_type": lifecycleType,
		"status":         status,
	})
	out, err := doGet[platformItemsResponse[AssetDefinition]](c, path)
	if err != nil {
		return nil, err
	}
	return out.Items, nil
}

func (c *Client) CreateAssetDefinition(input CreateAssetDefinitionInput) (*AssetDefinition, error) {
	return doPost[CreateAssetDefinitionInput, AssetDefinition](c, "/wallet/assets", input)
}

func (c *Client) ListWalletLedger(walletAccountID string) ([]WalletLedger, error) {
	path := withQuery("/wallet/ledger", map[string]string{
		"wallet_account_id": walletAccountID,
	})
	out, err := doGet[platformItemsResponse[WalletLedger]](c, path)
	if err != nil {
		return nil, err
	}
	return out.Items, nil
}

func (c *Client) ListRewards(productCode, beneficiaryType, beneficiaryID string) ([]RewardLedger, error) {
	path := withQuery("/incentives/rewards", map[string]string{
		"product_code":             productCode,
		"beneficiary_subject_type": beneficiaryType,
		"beneficiary_subject_id":   beneficiaryID,
	})
	out, err := doGet[platformItemsResponse[RewardLedger]](c, path)
	if err != nil {
		return nil, err
	}
	return out.Items, nil
}

func (c *Client) CreateReward(input CreateRewardInput) (*RewardLedger, error) {
	return doPost[CreateRewardInput, RewardLedger](c, "/incentives/rewards", input)
}

func (c *Client) ListCommissions(productCode, beneficiaryType, beneficiaryID, status string) ([]CommissionLedger, error) {
	path := withQuery("/incentives/commissions", map[string]string{
		"product_code":             productCode,
		"beneficiary_subject_type": beneficiaryType,
		"beneficiary_subject_id":   beneficiaryID,
		"status":                   status,
	})
	out, err := doGet[platformItemsResponse[CommissionLedger]](c, path)
	if err != nil {
		return nil, err
	}
	return out.Items, nil
}

func (c *Client) RedeemCommissions(input RedeemCommissionsInput) (*RedeemCommissionsResult, error) {
	return doPost[RedeemCommissionsInput, RedeemCommissionsResult](c, "/incentives/commissions/redeem", input)
}

func (c *Client) ListReferralPrograms(productCode, status string) ([]ReferralProgram, error) {
	path := withQuery("/incentives/referral-programs", map[string]string{
		"product_code": productCode,
		"status":       status,
	})
	out, err := doGet[platformItemsResponse[ReferralProgram]](c, path)
	if err != nil {
		return nil, err
	}
	return out.Items, nil
}

func (c *Client) CreateReferralProgram(input CreateReferralProgramInput) (*ReferralProgram, error) {
	return doPost[CreateReferralProgramInput, ReferralProgram](c, "/incentives/referral-programs", input)
}

func (c *Client) ListReferralCodes(programID, promoterType, promoterID, status string) ([]ReferralCode, error) {
	path := withQuery("/incentives/referral-codes", map[string]string{
		"program_id":            programID,
		"promoter_subject_type": promoterType,
		"promoter_subject_id":   promoterID,
		"status":                status,
	})
	out, err := doGet[platformItemsResponse[ReferralCode]](c, path)
	if err != nil {
		return nil, err
	}
	return out.Items, nil
}

func (c *Client) CreateReferralCode(input CreateReferralCodeInput) (*ReferralCode, error) {
	return doPost[CreateReferralCodeInput, ReferralCode](c, "/incentives/referral-codes", input)
}

func (c *Client) ResolveReferralCode(code, productCode string) (*ResolvedReferralCode, error) {
	path := withQuery(fmt.Sprintf("/incentives/referral-codes/%s/resolve", url.PathEscape(strings.TrimSpace(code))), map[string]string{
		"product_code": productCode,
	})
	return doGet[ResolvedReferralCode](c, path)
}

func (c *Client) ListReferralConversions(productCode, promoterType, promoterID, status string) ([]ReferralConversion, error) {
	path := withQuery("/incentives/referral-conversions", map[string]string{
		"product_code":          productCode,
		"promoter_subject_type": promoterType,
		"promoter_subject_id":   promoterID,
		"status":                status,
	})
	out, err := doGet[platformItemsResponse[ReferralConversion]](c, path)
	if err != nil {
		return nil, err
	}
	return out.Items, nil
}

func (c *Client) CreateReferralConversion(input CreateReferralConversionInput) (*ReferralConversion, error) {
	return doPost[CreateReferralConversionInput, ReferralConversion](c, "/incentives/referral-conversions", input)
}

func doGet[T any](c *Client, path string) (*T, error) {
	return doRequest[T](c, http.MethodGet, path, nil)
}

func doPost[Req any, Resp any](c *Client, path string, body Req) (*Resp, error) {
	return doRequest[Resp](c, http.MethodPost, path, body)
}

func doPublicPost[Req any, Resp any](c *Client, path string, body Req) (*Resp, error) {
	return doPublicRequest[Resp](c, http.MethodPost, path, body)
}

func doPut[Req any, Resp any](c *Client, path string, body Req) (*Resp, error) {
	return doRequest[Resp](c, http.MethodPut, path, body)
}

func doRequest[T any](c *Client, method, path string, payload any) (*T, error) {
	body, err := encodePayload(payload)
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequest(method, c.InternalURL(path), bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	for key, value := range c.buildHeaders(method, path, body) {
		req.Header.Set(key, value)
	}
	if len(body) > 0 {
		req.Header.Set("Content-Type", "application/json")
	}
	resp, err := c.http.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var out envelope[T]
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return nil, err
	}
	if resp.StatusCode >= 400 || out.Code != 0 {
		fields := make([]string, 0, len(out.Errors))
		for _, item := range out.Errors {
			fields = append(fields, item.Field)
		}
		return nil, &platformError{
			Status:    resp.StatusCode,
			Code:      out.Code,
			Message:   out.Message,
			ErrorCode: out.ErrorCode,
			ErrorHint: out.ErrorHint,
			Err:       out.Error,
			RequestID: out.RequestID,
			Fields:    fields,
		}
	}
	return &out.Data, nil
}

func doPublicRequest[T any](c *Client, method, path string, payload any) (*T, error) {
	body, err := encodePayload(payload)
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequest(method, c.PublicURL(path), bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/json")
	if len(body) > 0 {
		req.Header.Set("Content-Type", "application/json")
	}
	resp, err := c.http.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var out envelope[T]
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return nil, err
	}
	if resp.StatusCode >= 400 || out.Code != 0 {
		return nil, &platformError{
			Status:    resp.StatusCode,
			Code:      out.Code,
			Message:   out.Message,
			ErrorCode: out.ErrorCode,
			ErrorHint: out.ErrorHint,
			Err:       out.Error,
			RequestID: out.RequestID,
		}
	}
	return &out.Data, nil
}

func (c *Client) buildHeaders(method, path string, body []byte) map[string]string {
	timestamp := strconv.FormatInt(time.Now().Unix(), 10)
	signature := sign(c.secret, c.serviceName, method, path, timestamp, body)
	return map[string]string{
		"Accept":                    "application/json",
		"X-Internal-Service":        c.serviceName,
		"X-Internal-Timestamp":      timestamp,
		"X-Internal-Signature":      signature,
		"X-Internal-Service-Secret": c.secret,
		"X-Request-ID":              buildRequestID(c.serviceName),
		"X-Trace-ID":                buildRequestID("trace"),
	}
}

func encodePayload(payload any) ([]byte, error) {
	if payload == nil {
		return nil, nil
	}
	data, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}
	if string(data) == "null" {
		return nil, nil
	}
	return data, nil
}

func buildMessage(service, method, path, timestamp string, body []byte) string {
	bodyHash := sha256.Sum256(body)
	return fmt.Sprintf("%s\n%s\n%s\n%s\n%s", service, method, path, timestamp, hex.EncodeToString(bodyHash[:]))
}

func sign(secret, service, method, path, timestamp string, body []byte) string {
	message := buildMessage(service, method, path, timestamp, body)
	mac := hmac.New(sha256.New, []byte(secret))
	_, _ = mac.Write([]byte(message))
	return hex.EncodeToString(mac.Sum(nil))
}

func buildRequestID(prefix string) string {
	return fmt.Sprintf("%s-%d", prefix, time.Now().UnixNano())
}

func withQuery(path string, values map[string]string) string {
	q := url.Values{}
	for key, value := range values {
		if value != "" {
			q.Set(key, value)
		}
	}
	if len(q) == 0 {
		return path
	}
	return path + "?" + q.Encode()
}

func defaultString(value, fallback string) string {
	if value == "" {
		return fallback
	}
	return value
}

func IsConflict(err error) bool {
	var platformErr *platformError
	return errors.As(err, &platformErr) && platformErr.Status == http.StatusConflict
}

func IsNotFound(err error) bool {
	var platformErr *platformError
	return errors.As(err, &platformErr) && platformErr.Status == http.StatusNotFound
}

func IsUnauthorized(err error) bool {
	var platformErr *platformError
	return errors.As(err, &platformErr) && platformErr.Status == http.StatusUnauthorized
}

func ErrorCode(err error) string {
	var platformErr *platformError
	if errors.As(err, &platformErr) {
		return platformErr.ErrorCode
	}
	return ""
}

func ErrorHint(err error) string {
	var platformErr *platformError
	if errors.As(err, &platformErr) {
		return platformErr.ErrorHint
	}
	return ""
}

// HTTPStatus exposes the upstream HTTP status code for error mapping.
func HTTPStatus(err error) int {
	var platformErr *platformError
	if errors.As(err, &platformErr) {
		return platformErr.Status
	}
	return 0
}

// ResponseCode exposes the upstream JSON envelope code for error mapping.
func ResponseCode(err error) int {
	var platformErr *platformError
	if errors.As(err, &platformErr) {
		return platformErr.Code
	}
	return 0
}
