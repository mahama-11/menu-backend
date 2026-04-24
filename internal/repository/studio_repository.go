package repository

import (
	"fmt"
	"time"

	"menu-service/internal/models"

	"gorm.io/gorm"
)

type StudioRepository struct {
	db *gorm.DB
}

func NewStudioRepository(db *gorm.DB) *StudioRepository {
	return &StudioRepository{db: db}
}

func (r *StudioRepository) CreateAsset(item *models.StudioAsset) error {
	if item.ID == "" {
		item.ID = buildStudioID("asset")
	}
	return r.db.Create(item).Error
}

func (r *StudioRepository) ListAssets(orgID, userID, assetType, status string) ([]models.StudioAsset, error) {
	var items []models.StudioAsset
	q := r.db.Where("organization_id = ?", orgID).Order("created_at desc")
	if userID != "" {
		q = q.Where("user_id = ?", userID)
	}
	if assetType != "" {
		q = q.Where("asset_type = ?", assetType)
	}
	if status != "" {
		q = q.Where("status = ?", status)
	}
	err := q.Find(&items).Error
	return items, err
}

func (r *StudioRepository) ListAssetsPaginated(orgID, userID, assetType, status, query string, limit, offset int) ([]models.StudioAsset, int64, error) {
	var (
		items []models.StudioAsset
		total int64
	)
	q := r.db.Model(&models.StudioAsset{}).Where("organization_id = ?", orgID)
	if userID != "" {
		q = q.Where("user_id = ?", userID)
	}
	if assetType != "" {
		q = q.Where("asset_type = ?", assetType)
	}
	if status != "" {
		q = q.Where("status = ?", status)
	}
	if query != "" {
		like := "%" + query + "%"
		q = q.Where("file_name LIKE ? OR source_url LIKE ? OR metadata LIKE ?", like, like, like)
	}
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	if limit > 0 {
		q = q.Limit(limit)
	}
	if offset > 0 {
		q = q.Offset(offset)
	}
	if err := q.Order("created_at desc").Find(&items).Error; err != nil {
		return nil, 0, err
	}
	return items, total, nil
}

func (r *StudioRepository) FindAssetByID(orgID, assetID string) (*models.StudioAsset, error) {
	var item models.StudioAsset
	if err := r.db.Where("organization_id = ? AND id = ?", orgID, assetID).First(&item).Error; err != nil {
		return nil, err
	}
	return &item, nil
}

func (r *StudioRepository) FindAssetByIDGlobal(assetID string) (*models.StudioAsset, error) {
	var item models.StudioAsset
	if err := r.db.Where("id = ?", assetID).First(&item).Error; err != nil {
		return nil, err
	}
	return &item, nil
}

func (r *StudioRepository) CreateStylePreset(item *models.StylePreset) error {
	if item.ID == "" {
		item.ID = buildStudioID("style")
	}
	return r.db.Create(item).Error
}

func (r *StudioRepository) SaveStylePreset(item *models.StylePreset) error {
	return r.db.Save(item).Error
}

func (r *StudioRepository) ListStylePresets(orgID, visibility, status string) ([]models.StylePreset, error) {
	var items []models.StylePreset
	q := r.db.Where("organization_id = ?", orgID).Order("created_at desc")
	if visibility != "" {
		q = q.Where("visibility = ?", visibility)
	}
	if status != "" {
		q = q.Where("status = ?", status)
	}
	err := q.Find(&items).Error
	return items, err
}

func (r *StudioRepository) FindStylePresetByID(orgID, styleID string) (*models.StylePreset, error) {
	var item models.StylePreset
	if err := r.db.Where("organization_id = ? AND id = ?", orgID, styleID).First(&item).Error; err != nil {
		return nil, err
	}
	return &item, nil
}

func (r *StudioRepository) CreateGenerationJob(item *models.GenerationJob) error {
	if item.ID == "" {
		item.ID = buildStudioID("job")
	}
	return r.db.Create(item).Error
}

func (r *StudioRepository) FindGenerationJobByIdempotencyKey(orgID, userID, key string) (*models.GenerationJob, error) {
	if key == "" {
		return nil, gorm.ErrRecordNotFound
	}
	var item models.GenerationJob
	if err := r.db.Where("organization_id = ? AND user_id = ? AND idempotency_key = ?", orgID, userID, key).First(&item).Error; err != nil {
		return nil, err
	}
	return &item, nil
}

func (r *StudioRepository) SaveGenerationJob(item *models.GenerationJob) error {
	return r.db.Save(item).Error
}

func (r *StudioRepository) ListGenerationJobs(orgID, userID, status string) ([]models.GenerationJob, error) {
	var items []models.GenerationJob
	q := r.db.Where("organization_id = ?", orgID).Where("batch_root_id = '' OR batch_root_id IS NULL").Order("created_at desc")
	if userID != "" {
		q = q.Where("user_id = ?", userID)
	}
	if status != "" {
		q = q.Where("status = ?", status)
	}
	err := q.Find(&items).Error
	return items, err
}

func (r *StudioRepository) ListGenerationJobsPaginated(orgID, userID, status string, limit, offset int) ([]models.GenerationJob, int64, error) {
	var (
		items []models.GenerationJob
		total int64
	)
	q := r.db.Model(&models.GenerationJob{}).
		Where("organization_id = ?", orgID).
		Where("batch_root_id = '' OR batch_root_id IS NULL")
	if userID != "" {
		q = q.Where("user_id = ?", userID)
	}
	if status != "" {
		q = q.Where("status = ?", status)
	}
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	if limit > 0 {
		q = q.Limit(limit)
	}
	if offset > 0 {
		q = q.Offset(offset)
	}
	if err := q.Order("created_at desc").Find(&items).Error; err != nil {
		return nil, 0, err
	}
	return items, total, nil
}

func (r *StudioRepository) FindGenerationJobByID(orgID, jobID string) (*models.GenerationJob, error) {
	var item models.GenerationJob
	if err := r.db.Where("organization_id = ? AND id = ?", orgID, jobID).First(&item).Error; err != nil {
		return nil, err
	}
	return &item, nil
}

func (r *StudioRepository) FindGenerationJobByIDGlobal(jobID string) (*models.GenerationJob, error) {
	var item models.GenerationJob
	if err := r.db.Where("id = ?", jobID).First(&item).Error; err != nil {
		return nil, err
	}
	return &item, nil
}

func (r *StudioRepository) ListChildGenerationJobs(rootJobID string) ([]models.GenerationJob, error) {
	var items []models.GenerationJob
	err := r.db.Where("batch_root_id = ?", rootJobID).Order("created_at asc").Find(&items).Error
	return items, err
}

func (r *StudioRepository) CountActiveJobsForUser(orgID, userID string) (int64, error) {
	var count int64
	err := r.db.Model(&models.GenerationJob{}).
		Where("organization_id = ? AND user_id = ? AND status IN ?", orgID, userID, []string{"queued", "processing"}).
		Where("batch_root_id != '' OR mode != ?", "batch").
		Where("NOT EXISTS (SELECT 1 FROM menu_studio_charge_intents sci WHERE sci.job_id = menu_generation_jobs.id AND sci.status = ?)", "failed_need_reconcile").
		Count(&count).Error
	return count, err
}

func (r *StudioRepository) CountActiveJobsForOrg(orgID string) (int64, error) {
	var count int64
	err := r.db.Model(&models.GenerationJob{}).
		Where("organization_id = ? AND status IN ?", orgID, []string{"queued", "processing"}).
		Where("batch_root_id != '' OR mode != ?", "batch").
		Where("NOT EXISTS (SELECT 1 FROM menu_studio_charge_intents sci WHERE sci.job_id = menu_generation_jobs.id AND sci.status = ?)", "failed_need_reconcile").
		Count(&count).Error
	return count, err
}

func (r *StudioRepository) ListDispatchableJobs(limit int) ([]models.GenerationJob, error) {
	var items []models.GenerationJob
	q := r.db.Where("status = ?", "queued").Order("created_at asc")
	if limit > 0 {
		q = q.Limit(limit)
	}
	err := q.Find(&items).Error
	return items, err
}

func (r *StudioRepository) ListRecoverableJobs(now time.Time, limit int) ([]models.GenerationJob, error) {
	var items []models.GenerationJob
	q := r.db.Where("status IN ?", []string{"queued", "processing"}).
		Where("(timeout_at IS NOT NULL AND timeout_at <= ?) OR (next_retry_at IS NOT NULL AND next_retry_at <= ?)", now, now).
		Order("updated_at asc")
	if limit > 0 {
		q = q.Limit(limit)
	}
	err := q.Find(&items).Error
	return items, err
}

func (r *StudioRepository) CreateGenerationVariant(item *models.GenerationVariant) error {
	if item.ID == "" {
		item.ID = buildStudioID("variant")
	}
	return r.db.Create(item).Error
}

func (r *StudioRepository) CreateChargeIntent(item *models.StudioChargeIntent) error {
	if item.ID == "" {
		item.ID = buildStudioID("intent")
	}
	return r.db.Create(item).Error
}

func (r *StudioRepository) SaveChargeIntent(item *models.StudioChargeIntent) error {
	return r.db.Save(item).Error
}

func (r *StudioRepository) FindChargeIntentByJobID(jobID string) (*models.StudioChargeIntent, error) {
	var item models.StudioChargeIntent
	if err := r.db.Where("job_id = ?", jobID).First(&item).Error; err != nil {
		return nil, err
	}
	return &item, nil
}

func (r *StudioRepository) ListChargeIntents(orgID string, limit int) ([]models.StudioChargeIntent, error) {
	var items []models.StudioChargeIntent
	q := r.db.Where("organization_id = ?", orgID).Order("updated_at desc, created_at desc")
	if limit > 0 {
		q = q.Limit(limit)
	}
	if err := q.Find(&items).Error; err != nil {
		return nil, err
	}
	return items, nil
}

func (r *StudioRepository) SaveGenerationVariant(item *models.GenerationVariant) error {
	return r.db.Save(item).Error
}

func (r *StudioRepository) ListGenerationVariants(jobID string) ([]models.GenerationVariant, error) {
	var items []models.GenerationVariant
	err := r.db.Where("job_id = ?", jobID).Order("variant_index asc, created_at asc").Find(&items).Error
	return items, err
}

func (r *StudioRepository) FindGenerationVariant(jobID, variantID string) (*models.GenerationVariant, error) {
	var item models.GenerationVariant
	if err := r.db.Where("job_id = ? AND id = ?", jobID, variantID).First(&item).Error; err != nil {
		return nil, err
	}
	return &item, nil
}

func (r *StudioRepository) FindGenerationVariantByIndex(jobID string, index int) (*models.GenerationVariant, error) {
	var item models.GenerationVariant
	if err := r.db.Where("job_id = ? AND variant_index = ?", jobID, index).First(&item).Error; err != nil {
		return nil, err
	}
	return &item, nil
}

func (r *StudioRepository) FindGenerationVariantByAssetID(assetID string) (*models.GenerationVariant, error) {
	var item models.GenerationVariant
	if err := r.db.Where("asset_id = ?", assetID).Order("created_at desc").First(&item).Error; err != nil {
		return nil, err
	}
	return &item, nil
}

func (r *StudioRepository) FindLatestJobUsingAsset(orgID, assetID string) (*models.GenerationJob, error) {
	var item models.GenerationJob
	like := "%" + assetID + "%"
	if err := r.db.Where("organization_id = ? AND source_asset_ids LIKE ?", orgID, like).Order("created_at desc").First(&item).Error; err != nil {
		return nil, err
	}
	return &item, nil
}

func (r *StudioRepository) ClearSelectedVariants(jobID string) error {
	return r.db.Model(&models.GenerationVariant{}).Where("job_id = ?", jobID).Update("is_selected", false).Error
}

func buildStudioID(prefix string) string {
	return fmt.Sprintf("%s_%d", prefix, time.Now().UnixNano())
}
