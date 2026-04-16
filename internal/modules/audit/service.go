package audit

import (
	"encoding/json"
	"reflect"
	"sort"
	"time"

	"menu-service/internal/models"
	"menu-service/internal/repository"

	"github.com/gin-gonic/gin"
)

type Service struct {
	repo *repository.AuditRepository
}

type RecordInput struct {
	Action         string
	TargetType     string
	TargetID       string
	Status         string
	Details        string
	BeforeSnapshot any
	AfterSnapshot  any
}

func NewService(repo *repository.AuditRepository) *Service {
	return &Service{repo: repo}
}

func (s *Service) RecordFromGin(c *gin.Context, input RecordInput) error {
	beforeSnapshot, _ := encodeSnapshot(input.BeforeSnapshot)
	afterSnapshot, _ := encodeSnapshot(input.AfterSnapshot)
	item := &models.AuditLog{
		ID:             buildID("audit"),
		RequestID:      c.GetString("requestID"),
		TraceID:        c.GetString("traceID"),
		ActorUserID:    c.GetString("userID"),
		ActorOrgID:     c.GetString("orgID"),
		Action:         input.Action,
		TargetType:     input.TargetType,
		TargetID:       input.TargetID,
		Status:         defaultString(input.Status, "success"),
		Route:          c.FullPath(),
		Method:         c.Request.Method,
		Details:        input.Details,
		BeforeSnapshot: beforeSnapshot,
		AfterSnapshot:  afterSnapshot,
		DiffSummary:    buildDiffSummary(input.BeforeSnapshot, input.AfterSnapshot),
		CreatedAt:      time.Now(),
	}
	return s.repo.Create(item)
}

func encodeSnapshot(value any) (string, error) {
	if value == nil {
		return "", nil
	}
	data, err := json.Marshal(value)
	if err != nil {
		return "", err
	}
	return string(data), nil
}

func buildDiffSummary(before, after any) string {
	if before == nil && after == nil {
		return ""
	}
	if before == nil {
		keys := snapshotKeys(after)
		return "created:" + joinKeys(keys)
	}
	if after == nil {
		keys := snapshotKeys(before)
		return "deleted:" + joinKeys(keys)
	}
	beforeMap := toMap(before)
	afterMap := toMap(after)
	keySet := map[string]struct{}{}
	for key := range beforeMap {
		keySet[key] = struct{}{}
	}
	for key := range afterMap {
		keySet[key] = struct{}{}
	}
	changed := make([]string, 0, len(keySet))
	for key := range keySet {
		if !reflect.DeepEqual(beforeMap[key], afterMap[key]) {
			changed = append(changed, key)
		}
	}
	sort.Strings(changed)
	return "changed:" + joinKeys(changed)
}

func snapshotKeys(value any) []string {
	keys := make([]string, 0)
	for key := range toMap(value) {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}

func toMap(value any) map[string]any {
	if value == nil {
		return map[string]any{}
	}
	data, err := json.Marshal(value)
	if err != nil {
		return map[string]any{}
	}
	out := map[string]any{}
	if err := json.Unmarshal(data, &out); err != nil {
		return map[string]any{}
	}
	return out
}

func joinKeys(keys []string) string {
	if len(keys) == 0 {
		return ""
	}
	data, _ := json.Marshal(keys)
	return string(data)
}

func defaultString(value, fallback string) string {
	if value == "" {
		return fallback
	}
	return value
}

func buildID(prefix string) string {
	return prefix + "_" + time.Now().Format("20060102150405.000000000")
}
