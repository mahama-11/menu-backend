package share

import (
	"fmt"
	"testing"

	"menu-service/internal/config"
	"menu-service/internal/models"
	"menu-service/internal/repository"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestCreatePost_BuildsShareURL(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(fmt.Sprintf("file:%s?mode=memory&cache=shared", t.Name())), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := db.AutoMigrate(&models.StudioAsset{}, &models.SharePost{}, &models.SharePostLike{}, &models.SharePostFavorite{}); err != nil {
		t.Fatalf("auto migrate: %v", err)
	}
	studioRepo := repository.NewStudioRepository(db)
	if err := studioRepo.CreateAsset(&models.StudioAsset{
		ID:             "asset-1",
		UserID:         "user-1",
		OrganizationID: "org-1",
		AssetType:      "generated",
		SourceType:     "generated",
		Status:         "ready",
		FileName:       "dish.png",
		SourceURL:      "https://cdn.example.com/dish.png",
		PreviewURL:     "https://cdn.example.com/dish.png",
	}); err != nil {
		t.Fatalf("create asset: %v", err)
	}
	service := NewService(repository.NewShareRepository(db), studioRepo, config.AppConfig{
		FrontendBaseURL: "https://menu.example.com",
	})
	item, err := service.CreatePost("user-1", "org-1", CreatePostInput{
		AssetID:    "asset-1",
		Title:      "Dish",
		Visibility: "public",
	})
	if err != nil {
		t.Fatalf("CreatePost() error = %v", err)
	}
	if item.ShareURL == "" || item.Visibility != "public" || item.Status != "published" {
		t.Fatalf("unexpected share post: %+v", item)
	}
}

func TestShareEngagementFlow_PublicDetailViewLikeFavorite(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(fmt.Sprintf("file:%s?mode=memory&cache=shared", t.Name())), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := db.AutoMigrate(&models.StudioAsset{}, &models.SharePost{}, &models.SharePostLike{}, &models.SharePostFavorite{}); err != nil {
		t.Fatalf("auto migrate: %v", err)
	}

	studioRepo := repository.NewStudioRepository(db)
	if err := studioRepo.CreateAsset(&models.StudioAsset{
		ID:             "asset-1",
		UserID:         "user-1",
		OrganizationID: "org-1",
		AssetType:      "generated",
		SourceType:     "generated",
		Status:         "ready",
		FileName:       "dish.png",
		MimeType:       "image/png",
		SourceURL:      "https://cdn.example.com/dish.png",
		PreviewURL:     "https://cdn.example.com/dish-preview.png",
		Width:          1200,
		Height:         900,
	}); err != nil {
		t.Fatalf("create asset: %v", err)
	}

	service := NewService(repository.NewShareRepository(db), studioRepo, config.AppConfig{
		FrontendBaseURL: "https://menu.example.com",
	})

	post, err := service.CreatePost("user-1", "org-1", CreatePostInput{
		AssetID:    "asset-1",
		Title:      "Dish",
		Visibility: "public",
	})
	if err != nil {
		t.Fatalf("CreatePost() error = %v", err)
	}

	detail, err := service.GetPublicPost(post.ShareURL[len("https://menu.example.com/share/"):])
	if err != nil {
		t.Fatalf("GetPublicPost() error = %v", err)
	}
	if detail.Asset.AssetID != "asset-1" || detail.Asset.PreviewURL == "" {
		t.Fatalf("unexpected public detail: %+v", detail)
	}

	viewSummary, err := service.RecordPublicView(post.ShareURL[len("https://menu.example.com/share/"):])
	if err != nil {
		t.Fatalf("RecordPublicView() error = %v", err)
	}
	if viewSummary.ViewCount != 1 {
		t.Fatalf("expected view count 1, got %+v", viewSummary)
	}

	likeSummary, err := service.SetLike("user-2", "org-1", post.ShareID, true)
	if err != nil {
		t.Fatalf("SetLike() error = %v", err)
	}
	if !likeSummary.ViewerLiked || likeSummary.LikeCount != 1 {
		t.Fatalf("unexpected like summary: %+v", likeSummary)
	}

	favoriteSummary, err := service.SetFavorite("user-2", "org-1", post.ShareID, true)
	if err != nil {
		t.Fatalf("SetFavorite() error = %v", err)
	}
	if !favoriteSummary.ViewerFavorited || favoriteSummary.FavoriteCount != 1 {
		t.Fatalf("unexpected favorite summary: %+v", favoriteSummary)
	}

	engagement, err := service.GetEngagement("user-2", "org-1", post.ShareID)
	if err != nil {
		t.Fatalf("GetEngagement() error = %v", err)
	}
	if !engagement.ViewerLiked || !engagement.ViewerFavorited || engagement.ViewCount != 1 {
		t.Fatalf("unexpected engagement state: %+v", engagement)
	}

	unlike, err := service.SetLike("user-2", "org-1", post.ShareID, false)
	if err != nil {
		t.Fatalf("SetLike(false) error = %v", err)
	}
	if unlike.ViewerLiked || unlike.LikeCount != 0 {
		t.Fatalf("unexpected unlike summary: %+v", unlike)
	}
}
