package studio

import (
	"time"
)

func (s *Service) CancelGenerationJob(orgID, jobID string) (*GenerationJobSummary, error) {
	job, err := s.repo.FindGenerationJobByID(orgID, jobID)
	if err != nil {
		return nil, err
	}
	if job.Status == "completed" || job.Status == "failed" || job.Status == "canceled" {
		return s.GetGenerationJob(orgID, jobID)
	}
	if s.platform != nil && job.RuntimeJobID != "" {
		_, _ = s.platform.CancelRuntimeJob(job.RuntimeJobID)
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
