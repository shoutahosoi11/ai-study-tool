package persistence

import (
	"strings"
	"testing"
)

func TestStripeSubscriptionUserUpdatesAreScopedByResolvedUserID(t *testing.T) {
	tests := []struct {
		name  string
		query string
	}{
		{name: "update", query: updateStripeSubscriptionUserSQL},
		{name: "cancel", query: cancelStripeSubscriptionUserSQL},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			normalized := strings.ToLower(strings.Join(strings.Fields(tt.query), " "))
			if !strings.Contains(normalized, "where id = $1") {
				t.Fatalf("stripe subscription update must be scoped by resolved user id: %s", normalized)
			}
			if strings.Contains(normalized, "where stripe_customer_id") || strings.Contains(normalized, "or stripe_subscription_id") {
				t.Fatalf("stripe subscription update must not update by provider identifiers: %s", normalized)
			}
		})
	}
}
