package services

import (
	"encoding/json"
	"testing"
)

func TestBuildSearchBodyIncludesQueryAndFilters(t *testing.T) {
	minCost := 10.0
	maxCost := 50.0
	service := &ProductSearchService{}

	body, err := service.buildSearchBody(ProductSearchParams{
		Query:    "handmade lamp",
		Type:     "home",
		Category: "decor",
		MinCost:  &minCost,
		MaxCost:  &maxCost,
		Limit:    12,
		Offset:   24,
	})
	if err != nil {
		t.Fatalf("buildSearchBody returned error: %v", err)
	}

	var payload map[string]any
	if err := json.Unmarshal(body, &payload); err != nil {
		t.Fatalf("failed to decode search body: %v", err)
	}

	if payload["size"].(float64) != 12 {
		t.Fatalf("expected size 12, got %v", payload["size"])
	}
	if payload["from"].(float64) != 24 {
		t.Fatalf("expected offset 24, got %v", payload["from"])
	}

	query := payload["query"].(map[string]any)["bool"].(map[string]any)
	if len(query["should"].([]any)) == 0 {
		t.Fatal("expected full-text query in should clause")
	}
	if query["minimum_should_match"].(float64) != 1 {
		t.Fatalf("expected minimum_should_match 1, got %v", query["minimum_should_match"])
	}
	if len(query["filter"].([]any)) != 3 {
		t.Fatalf("expected 3 filters, got %d", len(query["filter"].([]any)))
	}
}

func TestBuildSearchBodyUsesDefaults(t *testing.T) {
	service := &ProductSearchService{}

	body, err := service.buildSearchBody(ProductSearchParams{})
	if err != nil {
		t.Fatalf("buildSearchBody returned error: %v", err)
	}

	var payload map[string]any
	if err := json.Unmarshal(body, &payload); err != nil {
		t.Fatalf("failed to decode search body: %v", err)
	}

	if payload["size"].(float64) != 20 {
		t.Fatalf("expected default size 20, got %v", payload["size"])
	}

	query := payload["query"].(map[string]any)["bool"].(map[string]any)
	if _, ok := query["should"]; ok {
		t.Fatal("did not expect should clause for empty search")
	}
}
