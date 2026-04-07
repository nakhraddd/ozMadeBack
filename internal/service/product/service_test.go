package product

import (
	"testing"
	"time"
)

func TestTrendingScorePrefersRecentProducts(t *testing.T) {
	now := time.Date(2026, time.April, 5, 12, 0, 0, 0, time.UTC)
	recent := TrendingScore(100, now.Add(-2*time.Hour), now)
	old := TrendingScore(100, now.Add(-72*time.Hour), now)

	if recent <= old {
		t.Fatalf("expected recent product to have higher score, got recent=%f old=%f", recent, old)
	}
}

func TestTrendingScoreClampsFutureCreationTime(t *testing.T) {
	now := time.Date(2026, time.April, 5, 12, 0, 0, 0, time.UTC)
	score := TrendingScore(50, now.Add(3*time.Hour), now)
	expected := TrendingScore(50, now, now)

	if score != expected {
		t.Fatalf("expected future timestamps to be clamped, got score=%f expected=%f", score, expected)
	}
}
