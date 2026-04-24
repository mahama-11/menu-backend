package channel

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"menu-service/internal/config"
	"menu-service/internal/platform"
)

func TestCurrentBinding_EnrichesPartnerAndProgram(t *testing.T) {
	client, shutdown := newChannelTestClient(t)
	defer shutdown()
	service := NewService(client)

	items, err := service.CurrentBinding("org-bound")
	if err != nil {
		t.Fatalf("CurrentBinding() error = %v", err)
	}
	if len(items) != 1 {
		t.Fatalf("len(items) = %d, want 1", len(items))
	}
	if items[0].Partner == nil || items[0].Partner.Code != "CHANNEL_A" {
		t.Fatalf("unexpected partner enrichment: %+v", items[0])
	}
	if items[0].Program == nil || items[0].Program.ProgramCode != "CHANNEL_PROGRAM_A" {
		t.Fatalf("unexpected program enrichment: %+v", items[0])
	}
}

func TestOverview_AndSettlements_ForChannelOrg(t *testing.T) {
	client, shutdown := newChannelTestClient(t)
	defer shutdown()
	service := NewService(client)

	overview, err := service.Overview("org-channel")
	if err != nil {
		t.Fatalf("Overview() error = %v", err)
	}
	if len(overview.Partners) != 1 || overview.Partners[0].Code != "CHANNEL_A" {
		t.Fatalf("unexpected partners: %+v", overview.Partners)
	}
	if overview.TotalCommission != 180 || overview.PendingCommission != 60 || overview.SettledCommission != 120 {
		t.Fatalf("unexpected commission summary: %+v", overview)
	}
	if overview.PendingClawback != 20 || overview.AppliedClawback != 5 {
		t.Fatalf("unexpected clawback summary: %+v", overview)
	}
	if overview.SettlementCount != 1 || len(overview.RecentSettlements) != 1 {
		t.Fatalf("unexpected settlement summary: %+v", overview)
	}

	commissions, err := service.ListCommissions("org-channel", "")
	if err != nil {
		t.Fatalf("ListCommissions() error = %v", err)
	}
	if len(commissions) != 2 {
		t.Fatalf("len(commissions) = %d, want 2", len(commissions))
	}

	settlements, err := service.ListSettlements("org-channel", "")
	if err != nil {
		t.Fatalf("ListSettlements() error = %v", err)
	}
	if len(settlements) != 1 {
		t.Fatalf("len(settlements) = %d, want 1", len(settlements))
	}
	if settlements[0].Batch == nil || settlements[0].Batch.BatchNo != "batch-1" {
		t.Fatalf("unexpected batch enrichment: %+v", settlements[0])
	}

	adjustments, err := service.ListAdjustments("org-channel", "")
	if err != nil {
		t.Fatalf("ListAdjustments() error = %v", err)
	}
	if len(adjustments) != 1 || adjustments[0].Item.AdjustmentAmount != 15 {
		t.Fatalf("unexpected adjustments: %+v", adjustments)
	}

	preview, err := service.Preview("org-preview", PreviewInput{
		BillableItemCode:   "ai_generation",
		AppliesTo:          "usage_charge",
		SourceChargeID:     "preview-charge-1",
		Currency:           "CNY",
		PaidAmount:         10000,
		NetCollectedAmount: 10000,
		PaymentFeeAmount:   500,
		TaxAmount:          1000,
	})
	if err != nil {
		t.Fatalf("Preview() error = %v", err)
	}
	if preview.Result == nil || !preview.Result.Matched || preview.Result.PolicyVersionID != "version-1" {
		t.Fatalf("unexpected preview: %+v", preview)
	}
	if preview.Program == nil || preview.Program.ProgramCode != "CHANNEL_PROGRAM_A" {
		t.Fatalf("unexpected preview program: %+v", preview)
	}
}

func newChannelTestClient(t *testing.T) (*platform.Client, func()) {
	t.Helper()
	mux := http.NewServeMux()
	mux.HandleFunc("/internal/v1/incentives/channel-partners", func(w http.ResponseWriter, r *http.Request) {
		writePlatformSuccess(w, map[string]any{
			"items": []map[string]any{
				{
					"id":                      "partner-1",
					"code":                    "CHANNEL_A",
					"name":                    "Channel A",
					"partner_type":            "channel",
					"settlement_subject_type": "organization",
					"settlement_subject_id":   "org-channel",
					"status":                  "active",
					"risk_level":              "low",
					"created_at":              time.Now().UTC().Format(time.RFC3339),
					"updated_at":              time.Now().UTC().Format(time.RFC3339),
				},
			},
		})
	})
	mux.HandleFunc("/internal/v1/incentives/channel-programs", func(w http.ResponseWriter, r *http.Request) {
		writePlatformSuccess(w, map[string]any{
			"items": []map[string]any{
				{
					"id":                       "program-1",
					"product_code":             "menu",
					"program_code":             "CHANNEL_PROGRAM_A",
					"name":                     "Channel Revenue Share",
					"program_type":             "channel_revenue_share",
					"status":                   "active",
					"default_settlement_cycle": "monthly",
					"created_at":               time.Now().UTC().Format(time.RFC3339),
					"updated_at":               time.Now().UTC().Format(time.RFC3339),
				},
			},
		})
	})
	mux.HandleFunc("/internal/v1/incentives/channel-bindings", func(w http.ResponseWriter, r *http.Request) {
		orgID := r.URL.Query().Get("org_id")
		items := []map[string]any{}
		if orgID == "org-bound" {
			items = append(items, map[string]any{
				"id":                 "binding-1",
				"product_code":       "menu",
				"org_id":             "org-bound",
				"channel_partner_id": "partner-1",
				"channel_program_id": "program-1",
				"binding_source":     "signup_code",
				"binding_scope":      "product_org",
				"status":             "active",
				"source_code":        "CHANNEL_A",
				"created_at":         time.Now().UTC().Format(time.RFC3339),
				"updated_at":         time.Now().UTC().Format(time.RFC3339),
			})
		}
		writePlatformSuccess(w, map[string]any{"items": items})
	})
	mux.HandleFunc("/internal/v1/incentives/channel-commissions", func(w http.ResponseWriter, r *http.Request) {
		writePlatformSuccess(w, map[string]any{
			"items": []map[string]any{
				{
					"id":                 "ledger-1",
					"channel_partner_id": "partner-1",
					"channel_program_id": "program-1",
					"commission_amount":  120,
					"settleable_amount":  120,
					"status":             "settled",
					"currency":           "CNY",
					"binding_id":         "binding-1",
					"created_at":         time.Now().UTC().Add(-time.Hour).Format(time.RFC3339),
					"updated_at":         time.Now().UTC().Format(time.RFC3339),
				},
				{
					"id":                 "ledger-2",
					"channel_partner_id": "partner-1",
					"channel_program_id": "program-1",
					"commission_amount":  60,
					"settleable_amount":  60,
					"status":             "pending",
					"currency":           "CNY",
					"binding_id":         "binding-2",
					"created_at":         time.Now().UTC().Format(time.RFC3339),
					"updated_at":         time.Now().UTC().Format(time.RFC3339),
				},
			},
		})
	})
	mux.HandleFunc("/internal/v1/incentives/channel-clawbacks", func(w http.ResponseWriter, r *http.Request) {
		writePlatformSuccess(w, map[string]any{
			"items": []map[string]any{
				{"id": "claw-1", "channel_partner_id": "partner-1", "clawback_amount": 20, "status": "pending", "currency": "CNY", "created_at": time.Now().UTC().Format(time.RFC3339), "updated_at": time.Now().UTC().Format(time.RFC3339)},
				{"id": "claw-2", "channel_partner_id": "partner-1", "clawback_amount": 5, "status": "applied", "currency": "CNY", "created_at": time.Now().UTC().Format(time.RFC3339), "updated_at": time.Now().UTC().Format(time.RFC3339)},
			},
		})
	})
	mux.HandleFunc("/internal/v1/incentives/channel-settlement-items", func(w http.ResponseWriter, r *http.Request) {
		writePlatformSuccess(w, map[string]any{
			"items": []map[string]any{
				{
					"id":                  "item-1",
					"settlement_batch_id": "batch-1",
					"channel_partner_id":  "partner-1",
					"commission_amount":   120,
					"clawback_amount":     5,
					"net_amount":          115,
					"status":              "completed",
					"currency":            "CNY",
					"created_at":          time.Now().UTC().Format(time.RFC3339),
					"updated_at":          time.Now().UTC().Format(time.RFC3339),
				},
			},
		})
	})
	mux.HandleFunc("/internal/v1/incentives/channel-settlement-batches/batch-1", func(w http.ResponseWriter, r *http.Request) {
		writePlatformSuccess(w, map[string]any{
			"batch": map[string]any{
				"id":                      "batch-1",
				"batch_no":                "batch-1",
				"product_code":            "menu",
				"channel_program_id":      "program-1",
				"settlement_cycle":        "monthly",
				"status":                  "closed",
				"gross_commission_amount": 120,
				"gross_clawback_amount":   5,
				"net_settleable_amount":   115,
				"period_start":            time.Now().UTC().Add(-24 * time.Hour).Format(time.RFC3339),
				"period_end":              time.Now().UTC().Format(time.RFC3339),
				"created_at":              time.Now().UTC().Format(time.RFC3339),
				"updated_at":              time.Now().UTC().Format(time.RFC3339),
			},
			"items": []map[string]any{},
		})
	})
	mux.HandleFunc("/internal/v1/incentives/channel-adjustments", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			writePlatformSuccess(w, map[string]any{
				"items": []map[string]any{
					{
						"id":                 "adjustment-1",
						"product_code":       "menu",
						"channel_partner_id": "partner-1",
						"channel_program_id": "program-1",
						"adjustment_type":    "manual_credit",
						"currency":           "CNY",
						"adjustment_amount":  15,
						"reason_code":        "PROMOTION_BONUS",
						"status":             "pending",
						"created_at":         time.Now().UTC().Format(time.RFC3339),
						"updated_at":         time.Now().UTC().Format(time.RFC3339),
					},
				},
			})
		case http.MethodPost:
			writePlatformSuccess(w, map[string]any{
				"id":                 "adjustment-created-1",
				"product_code":       "menu",
				"channel_partner_id": "partner-1",
				"channel_program_id": "program-1",
				"adjustment_type":    "manual_credit",
				"currency":           "CNY",
				"adjustment_amount":  20,
				"reason_code":        "PROMOTION_BONUS",
				"status":             "pending",
				"created_at":         time.Now().UTC().Format(time.RFC3339),
				"updated_at":         time.Now().UTC().Format(time.RFC3339),
			})
		default:
			w.WriteHeader(http.StatusMethodNotAllowed)
		}
	})
	mux.HandleFunc("/internal/v1/incentives/channel-policy-resolution-preview", func(w http.ResponseWriter, r *http.Request) {
		writePlatformSuccess(w, map[string]any{
			"matched":               true,
			"mode":                  "policy_version",
			"binding_id":            "binding-preview-1",
			"channel_id":            "partner-1",
			"channel_program_id":    "program-1",
			"policy_id":             "policy-1",
			"policy_version_id":     "version-1",
			"assignment_id":         "assignment-1",
			"assignment_level":      "product_default_assignment",
			"matched_rule_code":     "product_default_assignment.billable_item",
			"commissionable_amount": 8500,
			"commission_amount":     3400,
			"holdback_amount":       0,
			"settleable_amount":     3400,
			"status":                "earned",
			"snapshot": map[string]any{
				"id":                          "snapshot-1",
				"product_code":                "menu",
				"org_id":                      "org-preview",
				"source_charge_id":            "preview-charge-1",
				"currency":                    "CNY",
				"net_collected_amount":        10000,
				"payment_fee_amount":          500,
				"tax_amount":                  1000,
				"recognized_cost_amount":      1500,
				"distributable_profit_amount": 8500,
				"commission_recognition_at":   time.Now().UTC().Format(time.RFC3339),
				"created_at":                  time.Now().UTC().Format(time.RFC3339),
				"updated_at":                  time.Now().UTC().Format(time.RFC3339),
			},
		})
	})

	server := httptest.NewServer(mux)
	client := platform.New(config.PlatformConfig{
		BaseURL:               server.URL,
		Timeout:               5 * time.Second,
		InternalServiceSecret: "test-secret",
	})
	return client, server.Close
}

func writePlatformSuccess(w http.ResponseWriter, data any) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]any{
		"code":    0,
		"message": "ok",
		"data":    data,
	})
}
