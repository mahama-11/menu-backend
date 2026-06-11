package platform

import (
	"bytes"
	"context"
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

	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/trace"
)

type Client struct {
	baseURL     string
	secret      string
	serviceName string
	http        *http.Client
	ctx         context.Context
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

func (c *Client) WithContext(ctx context.Context) *Client {
	if c == nil || ctx == nil {
		return c
	}
	clone := *c
	clone.ctx = ctx
	return &clone
}

func (c *Client) InternalURL(path string) string {
	return fmt.Sprintf("%s/internal/v1/%s", c.baseURL, strings.TrimLeft(path, "/"))
}

func (c *Client) PublicURL(path string) string {
	return fmt.Sprintf("%s/api/v1/%s", c.baseURL, strings.TrimLeft(path, "/"))
}

func DefaultTimeout() time.Duration { return 5 * time.Second }

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

func (c *Client) GetCatalogOfferings(productCode string) (*OfferingsView, error) {
	return doGet[OfferingsView](c, withQuery("/catalog/offerings", map[string]string{
		"product_code": productCode,
	}))
}

func (c *Client) ListQuotaGrantPolicies(productCode, packageCode string) ([]QuotaGrantPolicy, error) {
	out, err := doGet[platformItemsResponse[QuotaGrantPolicy]](c, withQuery("/controls/quota/policies", map[string]string{
		"product_code": productCode,
		"package_code": packageCode,
	}))
	if err != nil {
		return nil, err
	}
	return out.Items, nil
}

func (c *Client) ListPackageCapabilityPolicies(productCode, packageCode string) ([]PackageCapabilityPolicy, error) {
	out, err := doGet[platformItemsResponse[PackageCapabilityPolicy]](c, withQuery("/controls/capability/policies", map[string]string{
		"product_code": productCode,
		"package_code": packageCode,
	}))
	if err != nil {
		return nil, err
	}
	return out.Items, nil
}

func (c *Client) GrantQuota(input GrantQuotaInput) error {
	_, err := doPost[GrantQuotaInput, map[string]any](c, "/controls/quota/grants", input)
	return err
}

func (c *Client) ActivatePackage(input ActivatePackageInput) (*PackageActivationResult, error) {
	return doPost[ActivatePackageInput, PackageActivationResult](c, "/controls/package-activations", input)
}

func (c *Client) GetQuotaBalance(subjectType, subjectID, billableItemCode string) (*QuotaBalance, error) {
	return doGet[QuotaBalance](c, withQuery("/controls/quota/balance", map[string]string{
		"billing_subject_type": subjectType,
		"billing_subject_id":   subjectID,
		"billable_item_code":   billableItemCode,
	}))
}

func (c *Client) GrantCapability(input GrantCapabilityInput) (*CapabilityGrant, error) {
	return doPost[GrantCapabilityInput, CapabilityGrant](c, "/controls/capability/grants", input)
}

func (c *Client) ResolveCapability(productCode, billingSubjectType, billingSubjectID, capabilityCode string) (*ResolveCapabilityResult, error) {
	return doGet[ResolveCapabilityResult](c, withQuery("/controls/capability/resolve", map[string]string{
		"product_code":         productCode,
		"billing_subject_type": billingSubjectType,
		"billing_subject_id":   billingSubjectID,
		"capability_code":      capabilityCode,
	}))
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
	ctx := c.requestContext()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.InternalURL(path), nil)
	if err != nil {
		return nil, nil, err
	}
	for key, value := range c.buildHeaders(ctx, http.MethodGet, path, nil) {
		req.Header.Set(key, value)
	}
	propagation.TraceContext{}.Inject(ctx, propagation.HeaderCarrier(req.Header))
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

func (c *Client) ListWalletAccounts(subjectType, subjectID, productCode string) ([]WalletAccount, error) {
	path := withQuery("/wallet/accounts", map[string]string{
		"billing_subject_type": subjectType,
		"billing_subject_id":   subjectID,
		"product_code":         productCode,
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

func (c *Client) ListWalletLedger(walletAccountID, productCode string) ([]WalletLedger, error) {
	path := withQuery("/wallet/ledger", map[string]string{
		"wallet_account_id": walletAccountID,
		"product_code":      productCode,
	})
	out, err := doGet[platformItemsResponse[WalletLedger]](c, path)
	if err != nil {
		return nil, err
	}
	return out.Items, nil
}

func (c *Client) PostWalletLedger(input PostWalletLedgerInput) (*WalletLedger, *WalletAccount, error) {
	out, err := doPost[PostWalletLedgerInput, postWalletLedgerResult](c, "/wallet/ledger", input)
	if err != nil {
		return nil, nil, err
	}
	return out.Ledger, out.Account, nil
}

func (c *Client) GrantCycleAllowance(input GrantCycleAllowanceInput) (*WalletBucket, *WalletAccount, error) {
	out, err := doPost[GrantCycleAllowanceInput, grantCycleAllowanceResult](c, "/wallet/cycle-allowances", input)
	if err != nil {
		return nil, nil, err
	}
	return out.Bucket, out.Account, nil
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

func (c *Client) InternalTemplateCatalog(productCode string) (*PlatformTemplateCatalogResult, error) {
	params := url.Values{}
	params.Set("product_code", productCode)
	params.Set("published_only", "true")
	path := "template-ops/catalog?" + params.Encode()
	return doGet[PlatformTemplateCatalogResult](c, path)
}

func (c *Client) InternalTemplateCatalogDetail(templateRef string) (*PlatformTemplateCatalogDetail, error) {
	return doGet[PlatformTemplateCatalogDetail](c, "template-ops/catalog/"+url.PathEscape(templateRef))
}

func doRequest[T any](c *Client, method, path string, payload any) (*T, error) {
	body, err := encodePayload(payload)
	if err != nil {
		return nil, err
	}
	ctx := c.requestContext()
	req, err := http.NewRequestWithContext(ctx, method, c.InternalURL(path), bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	for key, value := range c.buildHeaders(ctx, method, path, body) {
		req.Header.Set(key, value)
	}
	if len(body) > 0 {
		req.Header.Set("Content-Type", "application/json")
	}
	propagation.TraceContext{}.Inject(ctx, propagation.HeaderCarrier(req.Header))
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
	ctx := c.requestContext()
	req, err := http.NewRequestWithContext(ctx, method, c.PublicURL(path), bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/json")
	if len(body) > 0 {
		req.Header.Set("Content-Type", "application/json")
	}
	setCorrelationHeaders(req.Header, ctx, c.serviceName)
	propagation.TraceContext{}.Inject(ctx, propagation.HeaderCarrier(req.Header))
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

func (c *Client) buildHeaders(ctx context.Context, method, path string, body []byte) map[string]string {
	timestamp := strconv.FormatInt(time.Now().Unix(), 10)
	signature := sign(c.secret, c.serviceName, method, path, timestamp, body)
	headers := map[string]string{
		"Accept":                    "application/json",
		"X-Internal-Service":        c.serviceName,
		"X-Internal-Timestamp":      timestamp,
		"X-Internal-Signature":      signature,
		"X-Internal-Service-Secret": c.secret,
	}
	requestID, traceID := correlationIDs(ctx, c.serviceName)
	headers["X-Request-ID"] = requestID
	headers["X-Trace-ID"] = traceID
	return headers
}

func setCorrelationHeaders(header http.Header, ctx context.Context, serviceName string) {
	requestID, traceID := correlationIDs(ctx, serviceName)
	header.Set("X-Request-ID", requestID)
	header.Set("X-Trace-ID", traceID)
}

func correlationIDs(ctx context.Context, serviceName string) (string, string) {
	requestID := stringContextValue(ctx, "request_id")
	if requestID == "" {
		requestID = stringContextValue(ctx, "requestID")
	}
	if requestID == "" {
		requestID = buildRequestID(serviceName)
	}
	traceID := stringContextValue(ctx, "trace_id")
	if traceID == "" {
		traceID = stringContextValue(ctx, "traceID")
	}
	if spanCtx := trace.SpanContextFromContext(ctx); spanCtx.IsValid() {
		traceID = spanCtx.TraceID().String()
	}
	if traceID == "" {
		traceID = buildRequestID("trace")
	}
	return requestID, traceID
}

func (c *Client) requestContext() context.Context {
	if c != nil && c.ctx != nil {
		return c.ctx
	}
	return context.Background()
}

func stringContextValue(ctx context.Context, key string) string {
	if ctx == nil {
		return ""
	}
	value, _ := ctx.Value(key).(string)
	return value
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
