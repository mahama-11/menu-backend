package share

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"strings"
	"time"

	"menu-service/internal/config"
	"menu-service/internal/models"
	"menu-service/internal/repository"

	"gorm.io/gorm"
)

type Service struct {
	repo       *repository.ShareRepository
	studioRepo *repository.StudioRepository
	baseURL    string
}

type SharePostSummary struct {
	ShareID       string         `json:"share_id"`
	AssetID       string         `json:"asset_id"`
	JobID         string         `json:"job_id,omitempty"`
	VariantID     string         `json:"variant_id,omitempty"`
	Title         string         `json:"title,omitempty"`
	Caption       string         `json:"caption,omitempty"`
	Visibility    string         `json:"visibility"`
	Status        string         `json:"status"`
	ShareURL      string         `json:"share_url,omitempty"`
	ViewCount     int64          `json:"view_count"`
	LikeCount     int64          `json:"like_count"`
	FavoriteCount int64          `json:"favorite_count"`
	Metadata      map[string]any `json:"metadata,omitempty"`
	PublishedAt   *string        `json:"published_at,omitempty"`
	CreatedAt     string         `json:"created_at"`
	UpdatedAt     string         `json:"updated_at"`
}

type ShareAssetSummary struct {
	AssetID    string `json:"asset_id"`
	FileName   string `json:"file_name"`
	SourceURL  string `json:"source_url"`
	PreviewURL string `json:"preview_url,omitempty"`
	MimeType   string `json:"mime_type,omitempty"`
	Width      int    `json:"width,omitempty"`
	Height     int    `json:"height,omitempty"`
}

type SharePostDetail struct {
	SharePostSummary
	Asset ShareAssetSummary `json:"asset"`
}

type ShareEngagementSummary struct {
	ShareID         string `json:"share_id"`
	ViewCount       int64  `json:"view_count"`
	LikeCount       int64  `json:"like_count"`
	FavoriteCount   int64  `json:"favorite_count"`
	ViewerLiked     bool   `json:"viewer_liked"`
	ViewerFavorited bool   `json:"viewer_favorited"`
}

type CreatePostInput struct {
	AssetID    string         `json:"asset_id" binding:"required"`
	JobID      string         `json:"job_id"`
	VariantID  string         `json:"variant_id"`
	Title      string         `json:"title"`
	Caption    string         `json:"caption"`
	Visibility string         `json:"visibility" binding:"required,oneof=private organization public"`
	Metadata   map[string]any `json:"metadata"`
}

type SetEngagementInput struct {
	Active bool `json:"active"`
}

func NewService(repo *repository.ShareRepository, studioRepo *repository.StudioRepository, appCfg config.AppConfig) *Service {
	return &Service{
		repo:       repo,
		studioRepo: studioRepo,
		baseURL:    strings.TrimRight(appCfg.FrontendBaseURL, "/"),
	}
}

func (s *Service) CreatePost(userID, orgID string, input CreatePostInput) (*SharePostSummary, error) {
	asset, err := s.studioRepo.FindAssetByID(orgID, input.AssetID)
	if err != nil {
		return nil, err
	}
	now := time.Now()
	item := &models.SharePost{
		OrganizationID: orgID,
		UserID:         userID,
		AssetID:        asset.ID,
		JobID:          input.JobID,
		VariantID:      input.VariantID,
		Title:          strings.TrimSpace(input.Title),
		Caption:        strings.TrimSpace(input.Caption),
		Visibility:     input.Visibility,
		Status:         "published",
		ShareToken:     generateShareToken(),
		Metadata:       mustEncodeJSON(input.Metadata),
		PublishedAt:    &now,
	}
	item.ShareURL = s.buildShareURL(item.ShareToken)
	if err := s.repo.CreatePost(item); err != nil {
		return nil, err
	}
	return mapSharePost(item), nil
}

func (s *Service) ListPosts(userID, orgID, status string, limit int) ([]SharePostSummary, error) {
	items, err := s.repo.ListPosts(orgID, "", status, limit)
	if err != nil {
		return nil, err
	}
	out := make([]SharePostSummary, 0, len(items))
	for _, item := range items {
		if userID != "" && item.UserID != userID && item.Visibility == "private" {
			continue
		}
		out = append(out, *mapSharePost(&item))
	}
	return out, nil
}

func (s *Service) GetPost(orgID, shareID string) (*SharePostSummary, error) {
	item, err := s.repo.FindPostByID(orgID, shareID)
	if err != nil {
		return nil, err
	}
	return mapSharePost(item), nil
}

func (s *Service) ListPublicPosts(limit int, sort string) ([]SharePostSummary, error) {
	items, err := s.repo.ListPublicPosts(limit, sort)
	if err != nil {
		return nil, err
	}
	out := make([]SharePostSummary, 0, len(items))
	for _, item := range items {
		out = append(out, *mapSharePost(&item))
	}
	return out, nil
}

func (s *Service) ListFavoritePosts(userID, orgID string, limit int) ([]SharePostSummary, error) {
	items, err := s.repo.ListFavoritePosts(orgID, userID, limit)
	if err != nil {
		return nil, err
	}
	out := make([]SharePostSummary, 0, len(items))
	for _, item := range items {
		out = append(out, *mapSharePost(&item))
	}
	return out, nil
}

func (s *Service) GetPublicPost(token string) (*SharePostDetail, error) {
	item, err := s.repo.FindPostByToken(token)
	if err != nil {
		return nil, err
	}
	if item.Visibility != "public" || item.Status != "published" {
		return nil, gorm.ErrRecordNotFound
	}
	asset, err := s.studioRepo.FindAssetByID(item.OrganizationID, item.AssetID)
	if err != nil {
		return nil, err
	}
	return mapSharePostDetail(item, asset), nil
}

func (s *Service) RecordPublicView(token string) (*ShareEngagementSummary, error) {
	item, err := s.repo.FindPostByToken(token)
	if err != nil {
		return nil, err
	}
	if item.Visibility != "public" || item.Status != "published" {
		return nil, gorm.ErrRecordNotFound
	}
	if err := s.repo.IncrementViewCount(item.ID); err != nil {
		return nil, err
	}
	refreshed, err := s.repo.FindPostByID(item.OrganizationID, item.ID)
	if err != nil {
		return nil, err
	}
	return mapEngagementSummary(refreshed, false, false), nil
}

func (s *Service) GetEngagement(userID, orgID, shareID string) (*ShareEngagementSummary, error) {
	item, err := s.repo.FindPostByID(orgID, shareID)
	if err != nil {
		return nil, err
	}
	viewerLiked, viewerFavorited, err := s.repo.GetEngagementState(orgID, userID, shareID)
	if err != nil {
		return nil, err
	}
	return mapEngagementSummary(item, viewerLiked, viewerFavorited), nil
}

func (s *Service) SetLike(userID, orgID, shareID string, active bool) (*ShareEngagementSummary, error) {
	item, err := s.repo.FindPostByID(orgID, shareID)
	if err != nil {
		return nil, err
	}
	if item.Visibility != "public" || item.Status != "published" {
		return nil, gorm.ErrRecordNotFound
	}
	viewerLiked, err := s.repo.SetLike(orgID, userID, shareID, active)
	if err != nil {
		return nil, err
	}
	refreshed, err := s.repo.FindPostByID(orgID, shareID)
	if err != nil {
		return nil, err
	}
	_, viewerFavorited, err := s.repo.GetEngagementState(orgID, userID, shareID)
	if err != nil {
		return nil, err
	}
	return mapEngagementSummary(refreshed, viewerLiked, viewerFavorited), nil
}

func (s *Service) SetFavorite(userID, orgID, shareID string, active bool) (*ShareEngagementSummary, error) {
	item, err := s.repo.FindPostByID(orgID, shareID)
	if err != nil {
		return nil, err
	}
	if item.Visibility != "public" || item.Status != "published" {
		return nil, gorm.ErrRecordNotFound
	}
	viewerFavorited, err := s.repo.SetFavorite(orgID, userID, shareID, active)
	if err != nil {
		return nil, err
	}
	refreshed, err := s.repo.FindPostByID(orgID, shareID)
	if err != nil {
		return nil, err
	}
	viewerLiked, _, err := s.repo.GetEngagementState(orgID, userID, shareID)
	if err != nil {
		return nil, err
	}
	return mapEngagementSummary(refreshed, viewerLiked, viewerFavorited), nil
}

func (s *Service) buildShareURL(token string) string {
	base := s.baseURL
	if base == "" {
		base = "http://localhost:5173"
	}
	return base + "/share/" + token
}

func mapSharePost(item *models.SharePost) *SharePostSummary {
	out := &SharePostSummary{
		ShareID:       item.ID,
		AssetID:       item.AssetID,
		JobID:         item.JobID,
		VariantID:     item.VariantID,
		Title:         item.Title,
		Caption:       item.Caption,
		Visibility:    item.Visibility,
		Status:        item.Status,
		ShareURL:      item.ShareURL,
		ViewCount:     item.ViewCount,
		LikeCount:     item.LikeCount,
		FavoriteCount: item.FavoriteCount,
		Metadata:      decodeMap(item.Metadata),
		CreatedAt:     item.CreatedAt.UTC().Format(time.RFC3339),
		UpdatedAt:     item.UpdatedAt.UTC().Format(time.RFC3339),
	}
	if item.PublishedAt != nil {
		value := item.PublishedAt.UTC().Format(time.RFC3339)
		out.PublishedAt = &value
	}
	return out
}

func mapSharePostDetail(item *models.SharePost, asset *models.StudioAsset) *SharePostDetail {
	return &SharePostDetail{
		SharePostSummary: *mapSharePost(item),
		Asset: ShareAssetSummary{
			AssetID:    asset.ID,
			FileName:   asset.FileName,
			SourceURL:  asset.SourceURL,
			PreviewURL: asset.PreviewURL,
			MimeType:   asset.MimeType,
			Width:      asset.Width,
			Height:     asset.Height,
		},
	}
}

func mapEngagementSummary(item *models.SharePost, viewerLiked, viewerFavorited bool) *ShareEngagementSummary {
	return &ShareEngagementSummary{
		ShareID:         item.ID,
		ViewCount:       item.ViewCount,
		LikeCount:       item.LikeCount,
		FavoriteCount:   item.FavoriteCount,
		ViewerLiked:     viewerLiked,
		ViewerFavorited: viewerFavorited,
	}
}

func mustEncodeJSON(value any) string {
	if value == nil {
		return ""
	}
	data, _ := json.Marshal(value)
	return string(data)
}

func decodeMap(raw string) map[string]any {
	if raw == "" {
		return map[string]any{}
	}
	out := map[string]any{}
	_ = json.Unmarshal([]byte(raw), &out)
	return out
}

func generateShareToken() string {
	buffer := make([]byte, 12)
	if _, err := rand.Read(buffer); err != nil {
		return hex.EncodeToString([]byte(time.Now().Format("20060102150405.000000000")))
	}
	return hex.EncodeToString(buffer)
}
