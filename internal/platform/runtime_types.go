package platform

import "time"

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
