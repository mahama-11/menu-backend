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
	if err := db.AutoMigrate(&models.StudioAsset{}, &models.SharePost{}); err != nil {
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
