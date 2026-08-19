package r2

import (
	"testing"
	"time"

	"github.com/example/route-analysis-service/internal/domain/quota"
)

func TestEvaluatorRegisterDoesNotPanic(t *testing.T) {
	evaluator := quota.NewEvaluator(func() time.Time { return time.Unix(1000, 0) })
	policy := quota.Policy{Tenant: "t1", RequestCap: 5, RateLimit: 3, Window: time.Minute}
	if err := evaluator.Register(policy); err != nil {
		t.Fatal(err)
	}
	if got := evaluator.Remaining("t1"); got != 5 {
		t.Fatalf("expected 5 remaining, got %d", got)
	}
}

func TestZeroBudgetIsNotNil(t *testing.T) {
	budget := quota.NewBudget(0, time.Minute, time.Unix(1000, 0))
	if budget == nil {
		t.Fatal("NewBudget(0, ...) returned nil budget")
	}
	if err := budget.Consume(time.Unix(1000, 0)); err == nil {
		t.Fatal("expected rate limited for zero budget")
	}
}

func TestNilBudgetConsumeReturnsError(t *testing.T) {
	var budget *quota.Budget
	if err := budget.Consume(time.Unix(1000, 0)); err == nil {
		t.Fatal("expected error when consuming nil budget")
	}
}

func TestRateLimiterAllowDoesNotPanic(t *testing.T) {
	limiter := quota.NewRateLimiter(3, time.Minute)
	at := time.Unix(1000, 0)
	for i := 0; i < 3; i++ {
		if !limiter.Allow("tenant-a", at) {
			t.Fatalf("request %d should pass", i)
		}
	}
	if limiter.Allow("tenant-a", at) {
		t.Fatal("expected fourth request to be rate limited")
	}
}
