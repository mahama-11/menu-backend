package studio

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"menu-service/internal/config"

	"github.com/hibiken/asynq"
)

const (
	taskTypeDispatch = "studio:dispatch"
	taskTypeTimeout  = "studio:timeout"
)

type taskPayload struct {
	JobID string `json:"job_id"`
}

type AsynqRuntime struct {
	client    *asynq.Client
	server    *asynq.Server
	mux       *asynq.ServeMux
	queueName string
}

func NewAsynqRuntime(redisCfg config.RedisConfig, studioCfg config.StudioConfig, service *Service) (*AsynqRuntime, error) {
	if !redisCfg.Enabled {
		return nil, fmt.Errorf("studio queue requires redis.enabled=true")
	}
	studioCfg = defaultStudioConfig(studioCfg)
	redisOpt := asynq.RedisClientOpt{
		Addr:     fmt.Sprintf("%s:%d", redisCfg.Host, redisCfg.Port),
		Password: redisCfg.Password,
		DB:       redisCfg.DB,
	}
	queueName := studioCfg.QueueName
	if queueName == "" {
		queueName = "studio:default"
	}
	mux := asynq.NewServeMux()
	mux.HandleFunc(taskTypeDispatch, func(ctx context.Context, task *asynq.Task) error {
		payload, err := decodeTaskPayload(task)
		if err != nil {
			return err
		}
		return service.HandleDispatchTask(ctx, payload.JobID)
	})
	mux.HandleFunc(taskTypeTimeout, func(ctx context.Context, task *asynq.Task) error {
		payload, err := decodeTaskPayload(task)
		if err != nil {
			return err
		}
		return service.HandleTimeoutTask(ctx, payload.JobID)
	})
	server := asynq.NewServer(redisOpt, asynq.Config{
		Concurrency: studioCfg.WorkerConcurrency,
		Queues: map[string]int{
			queueName: 1,
		},
		RetryDelayFunc: func(n int, _ error, _ *asynq.Task) time.Duration {
			return studioCfg.RetryBackoff
		},
	})
	return &AsynqRuntime{
		client:    asynq.NewClient(redisOpt),
		server:    server,
		mux:       mux,
		queueName: queueName,
	}, nil
}

func (r *AsynqRuntime) EnqueueDispatch(jobID string, delay time.Duration) error {
	return r.enqueue(taskTypeDispatch, jobID, delay)
}

func (r *AsynqRuntime) EnqueueTimeout(jobID string, delay time.Duration) error {
	return r.enqueue(taskTypeTimeout, jobID, delay)
}

func (r *AsynqRuntime) Start() error {
	go func() {
		_ = r.server.Run(r.mux)
	}()
	return nil
}

func (r *AsynqRuntime) Shutdown() {
	if r.server != nil {
		r.server.Shutdown()
	}
	if r.client != nil {
		_ = r.client.Close()
	}
}

func (r *AsynqRuntime) enqueue(taskType, jobID string, delay time.Duration) error {
	payload, _ := json.Marshal(taskPayload{JobID: jobID})
	task := asynq.NewTask(taskType, payload)
	options := []asynq.Option{
		asynq.Queue(r.queueName),
		asynq.MaxRetry(0),
	}
	if delay > 0 {
		options = append(options, asynq.ProcessIn(delay))
	}
	_, err := r.client.Enqueue(task, options...)
	return err
}

func decodeTaskPayload(task *asynq.Task) (*taskPayload, error) {
	var payload taskPayload
	if err := json.Unmarshal(task.Payload(), &payload); err != nil {
		return nil, err
	}
	return &payload, nil
}
