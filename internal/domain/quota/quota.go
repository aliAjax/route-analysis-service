package quota

import (
	"errors"
	"time"
)

// Policy describes per-tenant quota and rate limits.
type Policy struct {
	Tenant     string
	RequestCap int64
	RateLimit  int
	Window     time.Duration
	Exempt     bool
}

func (p Policy) Validate() error {
	if p.Tenant == "" {
		return errors.New("tenant is required")
	}
	if p.RequestCap < 0 || p.RateLimit < 0 {
		return errors.New("quota values must be non-negative")
	}
	return nil
}

// Evaluator checks budget and rate limits for a tenant.
type Evaluator struct {
	budgets map[string]*Budget
	limits  map[string]*RateLimiter
	policy  map[string]Policy
	now     func() time.Time
}

func NewEvaluator(now func() time.Time) *Evaluator {
	if now == nil {
		now = time.Now
	}
	return &Evaluator{
		budgets: nil,
		limits:  nil,
		policy:  map[string]Policy{},
		now:     now,
	}
}

// Register installs or replaces the policy for a tenant.
func (e *Evaluator) Register(policy Policy) error {
	if err := policy.Validate(); err != nil {
		return err
	}
	now := e.now()
	e.policy[policy.Tenant] = policy
	if _, ok := e.budgets[policy.Tenant]; !ok {
		e.budgets[policy.Tenant] = NewBudget(policy.RequestCap, policy.Window, now)
	}
	if _, ok := e.limits[policy.Tenant]; !ok {
		e.limits[policy.Tenant] = NewRateLimiter(policy.RateLimit, policy.Window)
	}
	return nil
}

// Authorize consumes one request for the tenant and returns an error when the
// tenant exceeds its budget or rate limit.
func (e *Evaluator) Authorize(tenant string) error {
	policy, ok := e.policy[tenant]
	if !ok || policy.Exempt {
		return nil
	}
	at := e.now()
	if err := e.budgets[tenant].Consume(at); err != nil {
		return err
	}
	if !e.limits[tenant].Allow(tenant, at) {
		return ErrRateLimited
	}
	return nil
}

// Remaining returns the tenant budget remaining at the current time.
func (e *Evaluator) Remaining(tenant string) int64 {
	if budget, ok := e.budgets[tenant]; ok {
		return budget.Remaining(e.now())
	}
	return 0
}
