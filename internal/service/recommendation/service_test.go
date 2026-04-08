package recommendation

import (
	"testing"
	"time"

	"ozMadeBack/internal/models"
)

func TestGlobalScorePrefersRecentPopularProducts(t *testing.T) {
	now := time.Date(2026, time.April, 8, 12, 0, 0, 0, time.UTC)

	recentPopular := globalScore(models.Product{
		ViewCount:     200,
		AverageRating: 4.5,
		CreatedAt:     now.Add(-6 * time.Hour),
	}, 10, 3, now)

	oldUnpopular := globalScore(models.Product{
		ViewCount:     10,
		AverageRating: 1.0,
		CreatedAt:     now.Add(-20 * 24 * time.Hour),
	}, 1, 0, now)

	if recentPopular <= oldUnpopular {
		t.Fatalf("expected recent popular product to score higher, got recent=%f old=%f", recentPopular, oldUnpopular)
	}
}

func TestNormalizePreferenceKey(t *testing.T) {
	if got := normalizePreferenceKey("  Home "); got != "home" {
		t.Fatalf("expected normalized key to be home, got %q", got)
	}
}
