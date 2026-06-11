package user

import "context"

func (s *Service) WithContext(ctx context.Context) *Service {
	if s == nil || ctx == nil {
		return s
	}
	clone := *s
	if s.platform != nil {
		clone.platform = s.platform.WithContext(ctx)
	}
	if s.auth != nil {
		clone.auth = s.auth.WithContext(ctx)
	}
	return &clone
}
