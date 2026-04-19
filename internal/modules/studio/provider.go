package studio

import (
	"context"
	"fmt"
	"sync"
	"time"
)

type ProviderJobRequest struct {
	JobID          string
	Mode           string
	OrganizationID string
	UserID         string
	SourceAssetIDs []string
	StylePresetID  string
	Provider       string
	Prompt         StyleExecutionProfile
	Params         map[string]any
	RequestedCount int
	Metadata       map[string]any
}

type ProviderSubmission struct {
	ProviderJobID string
	Stage         string
	StageMessage  string
	EtaSeconds    int
}

type GenerationProvider interface {
	Name() string
	Submit(ctx context.Context, req ProviderJobRequest) (*ProviderSubmission, error)
	Cancel(ctx context.Context, providerJobID string) error
}

type ProviderRegistry struct {
	mu        sync.RWMutex
	providers map[string]GenerationProvider
}

func NewProviderRegistry() *ProviderRegistry {
	registry := &ProviderRegistry{providers: map[string]GenerationProvider{}}
	registry.Register(newManualProvider("manual"))
	registry.Register(newManualProvider("mock"))
	return registry
}

func (r *ProviderRegistry) Register(provider GenerationProvider) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.providers[provider.Name()] = provider
}

func (r *ProviderRegistry) Get(name string) (GenerationProvider, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	provider, ok := r.providers[name]
	if !ok {
		return nil, fmt.Errorf("provider %s not registered", name)
	}
	return provider, nil
}

type manualProvider struct {
	name string
}

func newManualProvider(name string) GenerationProvider {
	return &manualProvider{name: name}
}

func (p *manualProvider) Name() string { return p.name }

func (p *manualProvider) Submit(_ context.Context, req ProviderJobRequest) (*ProviderSubmission, error) {
	return &ProviderSubmission{
		ProviderJobID: fmt.Sprintf("%s-%s-%d", p.name, req.JobID, time.Now().UnixNano()),
		Stage:         "provider_accepted",
		StageMessage:  "Accepted by provider and waiting for processing results",
		EtaSeconds:    30,
	}, nil
}

func (p *manualProvider) Cancel(_ context.Context, _ string) error {
	return nil
}
