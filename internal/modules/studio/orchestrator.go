package studio

import (
	"context"
	"errors"
	"time"

	"menu-service/internal/config"
	"menu-service/internal/models"
	"menu-service/pkg/logger"
)

func (s *Service) UseRuntime(queue JobQueue, registry *ProviderRegistry) {
	if queue != nil {
		s.queue = queue
	}
	if registry != nil {
		s.registry = registry
	}
}

func (s *Service) HandleDispatchTask(_ context.Context, jobID string) error {
	return s.dispatchByID(jobID, time.Now())
}

func (s *Service) HandleTimeoutTask(_ context.Context, jobID string) error {
	return s.handleTimeoutByID(jobID, time.Now())
}

func (s *Service) CancelGenerationJob(orgID, jobID string) (*GenerationJobSummary, error) {
	job, err := s.repo.FindGenerationJobByID(orgID, jobID)
	if err != nil {
		return nil, err
	}
	if job.Status == "completed" || job.Status == "failed" || job.Status == "canceled" {
		return s.GetGenerationJob(orgID, jobID)
	}
	if provider, getErr := s.registry.Get(job.Provider); getErr == nil && job.ProviderJobID != "" {
		_ = provider.Cancel(context.Background(), job.ProviderJobID)
	}
	now := time.Now()
	job.Status = "canceled"
	job.Stage = "canceled"
	job.StageMessage = "Job canceled"
	job.CanceledAt = &now
	job.TimeoutAt = nil
	job.NextRetryAt = nil
	if err := s.repo.SaveGenerationJob(job); err != nil {
		return nil, err
	}
	_ = s.releaseChargeIntent(job)
	if job.BatchRootID != "" {
		_ = s.refreshBatchRoot(orgID, job.BatchRootID)
	}
	return s.GetGenerationJob(orgID, jobID)
}

func (s *Service) dispatchJob(job *models.GenerationJob, now time.Time) error {
	if job.Status != "queued" {
		return nil
	}
	if ok, message, err := s.canDispatch(job.OrganizationID, job.UserID); err != nil {
		return err
	} else if !ok {
		job.Stage = "queued"
		job.StageMessage = message
		job.QueuePosition++
		saveErr := s.repo.SaveGenerationJob(job)
		if saveErr == nil && s.queue != nil {
			_ = s.queue.EnqueueDispatch(job.ID, s.cfg.RetryBackoff)
		}
		return saveErr
	}
	provider, err := s.registry.Get(job.Provider)
	if err != nil {
		return s.rescheduleOrFail(job, "PROVIDER_NOT_FOUND", err.Error(), now)
	}
	job.Stage = "dispatching"
	job.StageMessage = "Dispatching to provider"
	job.Status = "processing"
	job.AttemptCount++
	timeoutAt := now.Add(s.cfg.ExecutionTimeout)
	job.TimeoutAt = &timeoutAt
	job.HeartbeatAt = &now
	if saveErr := s.repo.SaveGenerationJob(job); saveErr != nil {
		return saveErr
	}
	submission, err := provider.Submit(context.Background(), ProviderJobRequest{
		JobID:          job.ID,
		Mode:           job.Mode,
		OrganizationID: job.OrganizationID,
		UserID:         job.UserID,
		SourceAssetIDs: decodeStringSlice(job.SourceAssetIDs),
		StylePresetID:  job.StylePresetID,
		Provider:       job.Provider,
		Prompt:         decodeExecutionProfile(job.PromptSnapshot),
		Params:         decodeMap(job.ParamsSnapshot),
		RequestedCount: job.RequestedVariants,
		Metadata:       decodeMap(job.Metadata),
	})
	if err != nil {
		return s.rescheduleOrFail(job, "PROVIDER_SUBMIT_FAILED", err.Error(), now)
	}
	job.ProviderJobID = submission.ProviderJobID
	job.Stage = defaultString(submission.Stage, "provider_accepted")
	job.StageMessage = defaultString(submission.StageMessage, "Accepted by provider")
	job.QueuePosition = 0
	job.EtaSeconds = submission.EtaSeconds
	job.NextRetryAt = nil
	if err := s.repo.SaveGenerationJob(job); err != nil {
		return err
	}
	if s.queue != nil && s.cfg.ExecutionTimeout > 0 {
		if err := s.queue.EnqueueTimeout(job.ID, s.cfg.ExecutionTimeout); err != nil {
			logger.With("job_id", job.ID, "error", err).Error("studio.timeout.enqueue_failed")
		}
	}
	return nil
}

func (s *Service) rescheduleOrFail(job *models.GenerationJob, errorCode, message string, now time.Time) error {
	job.ErrorCode = errorCode
	job.ErrorMessage = message
	job.TimeoutAt = nil
	job.HeartbeatAt = nil
	if job.AttemptCount >= job.MaxAttempts {
		job.Status = "failed"
		job.Stage = "failed"
		job.StageMessage = message
		completedAt := now
		job.CompletedAt = &completedAt
		if err := s.repo.SaveGenerationJob(job); err != nil {
			return err
		}
		_ = s.releaseChargeIntent(job)
		return nil
	}
	retryAt := now.Add(s.cfg.RetryBackoff)
	job.Status = "queued"
	job.Stage = "retry_scheduled"
	job.StageMessage = "Retry scheduled after provider failure"
	job.NextRetryAt = &retryAt
	if err := s.repo.SaveGenerationJob(job); err != nil {
		return err
	}
	if s.queue != nil {
		return s.queue.EnqueueDispatch(job.ID, s.cfg.RetryBackoff)
	}
	return nil
}

func (s *Service) canDispatch(orgID, userID string) (bool, string, error) {
	if s.cfg.MaxConcurrentPerUser > 0 {
		count, err := s.repo.CountActiveJobsForUser(orgID, userID)
		if err != nil {
			return false, "", err
		}
		if count > int64(s.cfg.MaxConcurrentPerUser) {
			return false, "Waiting for user concurrency slot", nil
		}
	}
	if s.cfg.MaxConcurrentPerOrg > 0 {
		count, err := s.repo.CountActiveJobsForOrg(orgID)
		if err != nil {
			return false, "", err
		}
		if count > int64(s.cfg.MaxConcurrentPerOrg) {
			return false, "Waiting for organization concurrency slot", nil
		}
	}
	return true, "", nil
}

func (s *Service) refreshBatchRoot(orgID, rootJobID string) error {
	root, err := s.repo.FindGenerationJobByID(orgID, rootJobID)
	if err != nil {
		return err
	}
	children, err := s.repo.ListChildGenerationJobs(rootJobID)
	if err != nil {
		return err
	}
	if len(children) == 0 {
		return nil
	}
	completed := 0
	failed := 0
	canceled := 0
	progressSum := 0
	for _, child := range children {
		progressSum += child.Progress
		switch child.Status {
		case "completed":
			completed++
		case "failed":
			failed++
		case "canceled":
			canceled++
		}
	}
	root.Progress = progressSum / len(children)
	root.ChildJobCount = len(children)
	switch {
	case completed == len(children):
		root.Status = "completed"
		root.Stage = "completed"
		root.StageMessage = "All batch items completed"
		now := time.Now()
		root.CompletedAt = &now
	case failed+canceled == len(children):
		root.Status = "failed"
		root.Stage = "failed"
		root.StageMessage = "All batch items failed or were canceled"
		now := time.Now()
		root.CompletedAt = &now
	default:
		root.Status = "processing"
		root.Stage = "running"
		root.StageMessage = "Batch is still processing"
	}
	return s.repo.SaveGenerationJob(root)
}

func defaultStudioConfig(cfg config.StudioConfig) config.StudioConfig {
	if cfg.WorkerConcurrency <= 0 {
		cfg.WorkerConcurrency = 8
	}
	if cfg.QueueName == "" {
		cfg.QueueName = "studio:default"
	}
	if cfg.ExecutionTimeout <= 0 {
		cfg.ExecutionTimeout = 5 * time.Minute
	}
	if cfg.RetryBackoff <= 0 {
		cfg.RetryBackoff = 15 * time.Second
	}
	if cfg.MaxAttempts <= 0 {
		cfg.MaxAttempts = 3
	}
	if cfg.DefaultVariantCount <= 0 {
		cfg.DefaultVariantCount = 4
	}
	if cfg.DefaultProvider == "" {
		cfg.DefaultProvider = "manual"
	}
	return cfg
}

var ErrJobCanceled = errors.New("generation job canceled")

func (s *Service) dispatchByID(jobID string, now time.Time) error {
	job, err := s.repo.FindGenerationJobByIDGlobal(jobID)
	if err != nil {
		return err
	}
	return s.dispatchJob(job, now)
}

func (s *Service) handleTimeoutByID(jobID string, now time.Time) error {
	job, err := s.repo.FindGenerationJobByIDGlobal(jobID)
	if err != nil {
		return err
	}
	if job.Status == "completed" || job.Status == "failed" || job.Status == "canceled" {
		return nil
	}
	if job.TimeoutAt == nil || job.TimeoutAt.After(now) {
		return nil
	}
	return s.rescheduleOrFail(job, "JOB_TIMEOUT", "Job timed out while waiting for provider completion", now)
}
