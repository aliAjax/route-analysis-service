package application

import (
	"context"

	"github.com/example/route-analysis-service/internal/domain/quota"
)

// RegisterQuotaPolicy installs a tenant quota policy.
func (s *Service) RegisterQuotaPolicy(ctx context.Context, policy quota.Policy) error {
	return s.quota.Register(policy)
}

// AuthorizeRequest consumes one request for a tenant.
func (s *Service) AuthorizeRequest(ctx context.Context, tenant string) error {
	return s.quota.Authorize(tenant)
}

// QuotaStatus returns a tenant's remaining budget.
func (s *Service) QuotaStatus(ctx context.Context, tenant string) map[string]any {
	return map[string]any{
		"tenant":    tenant,
		"remaining": s.quota.Remaining(tenant),
	}
}
