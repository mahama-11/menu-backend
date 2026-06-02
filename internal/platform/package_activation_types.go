package platform

import "encoding/json"

type ActivatePackageInput struct {
	ProductCode        string          `json:"product_code"`
	PackageCode        string          `json:"package_code"`
	BillingSubjectType string          `json:"billing_subject_type"`
	BillingSubjectID   string          `json:"billing_subject_id"`
	ActivationReason   string          `json:"activation_reason,omitempty"`
	ReferenceID        string          `json:"reference_id"`
	Metadata           json.RawMessage `json:"metadata,omitempty"`
}

type PackageActivationResult struct {
	ProductCode        string            `json:"product_code"`
	PackageCode        string            `json:"package_code"`
	BillingSubjectType string            `json:"billing_subject_type"`
	BillingSubjectID   string            `json:"billing_subject_id"`
	ActivationReason   string            `json:"activation_reason"`
	ReferenceID        string            `json:"reference_id"`
	QuotaGrants        []map[string]any  `json:"quota_grants"`
	CapabilityGrants   []CapabilityGrant `json:"capability_grants"`
	GrantedQuotaUnits  int64             `json:"granted_quota_units"`
	Idempotent         bool              `json:"idempotent"`
}
