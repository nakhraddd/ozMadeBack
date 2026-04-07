package services

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"

	"ozMadeBack/config"
	"ozMadeBack/internal/database"
	"ozMadeBack/internal/models"

	"gorm.io/gorm"
)

var ProductSearch *ProductSearchService

const defaultSearchTimeout = 5 * time.Second

type ProductSearchService struct {
	baseURL    string
	indexName  string
	enabled    bool
	httpClient *http.Client
}

type ProductSearchParams struct {
	Query    string
	Type     string
	Category string
	MinCost  *float64
	MaxCost  *float64
	Limit    int
	Offset   int
}

type productDocument struct {
	ID            uint      `json:"id"`
	SellerID      uint      `json:"seller_id"`
	Title         string    `json:"title"`
	Description   string    `json:"description"`
	Type          string    `json:"type"`
	Address       string    `json:"address"`
	Categories    []string  `json:"categories"`
	Cost          float64   `json:"cost"`
	ViewCount     int64     `json:"view_count"`
	AverageRating float64   `json:"average_rating"`
	Composition   string    `json:"composition"`
	Weight        string    `json:"weight"`
	CreatedAt     time.Time `json:"created_at"`
}

type elasticSearchResponse struct {
	Hits struct {
		Hits []struct {
			ID string `json:"_id"`
		} `json:"hits"`
	} `json:"hits"`
}

func InitProductSearchService() {
	enabled := strings.EqualFold(config.GetEnv("ELASTICSEARCH_ENABLED", "true"), "true")
	baseURL := strings.TrimRight(config.GetEnv("ELASTICSEARCH_URL", "http://localhost:9200"), "/")
	indexName := config.GetEnv("ELASTICSEARCH_INDEX", "products")

	ProductSearch = &ProductSearchService{
		baseURL:   baseURL,
		indexName: indexName,
		enabled:   enabled,
		httpClient: &http.Client{
			Timeout: defaultSearchTimeout,
		},
	}

	if !enabled {
		log.Println("Elasticsearch integration disabled")
		return
	}

	log.Printf("Elasticsearch service initialized with index %q at %s", indexName, baseURL)
}

func BootstrapProductIndex(ctx context.Context) {
	if ProductSearch == nil || !ProductSearch.enabled {
		return
	}

	if err := ProductSearch.EnsureIndex(ctx); err != nil {
		log.Printf("failed to ensure Elasticsearch index: %v", err)
		return
	}

	syncOnStartup := strings.EqualFold(config.GetEnv("ELASTICSEARCH_SYNC_ON_STARTUP", "true"), "true")
	if !syncOnStartup {
		return
	}

	if err := ProductSearch.ReindexAll(ctx); err != nil {
		log.Printf("failed to sync products to Elasticsearch: %v", err)
	}
}

func (s *ProductSearchService) EnsureIndex(ctx context.Context) error {
	if !s.isEnabled() {
		return nil
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPut, s.endpoint(s.indexName), strings.NewReader(indexMapping))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusOK || resp.StatusCode == http.StatusCreated {
		return nil
	}
	if resp.StatusCode == http.StatusBadRequest {
		body, _ := io.ReadAll(resp.Body)
		if bytes.Contains(body, []byte("resource_already_exists_exception")) {
			return nil
		}
	}

	return readElasticError(resp)
}

func (s *ProductSearchService) SearchProducts(ctx context.Context, params ProductSearchParams) ([]uint, error) {
	if !s.isEnabled() {
		return s.searchProductsFallback(ctx, params)
	}

	body, err := s.buildSearchBody(params)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, s.endpoint(s.indexName, "_search"), bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := s.httpClient.Do(req)
	if err != nil {
		log.Printf("Elasticsearch search failed, falling back to database search: %v", err)
		return s.searchProductsFallback(ctx, params)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		log.Printf("Elasticsearch returned %d, falling back to database search", resp.StatusCode)
		return s.searchProductsFallback(ctx, params)
	}

	var payload elasticSearchResponse
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		return nil, err
	}

	ids := make([]uint, 0, len(payload.Hits.Hits))
	for _, hit := range payload.Hits.Hits {
		id, err := strconv.ParseUint(hit.ID, 10, 64)
		if err != nil {
			continue
		}
		ids = append(ids, uint(id))
	}

	return ids, nil
}

func (s *ProductSearchService) IndexProduct(ctx context.Context, product models.Product) error {
	if !s.isEnabled() {
		return nil
	}

	document := buildProductDocument(product)
	payload, err := json.Marshal(document)
	if err != nil {
		return err
	}

	req, err := http.NewRequestWithContext(
		ctx,
		http.MethodPut,
		s.endpoint(s.indexName, "_doc", strconv.FormatUint(uint64(product.ID), 10)),
		bytes.NewReader(payload),
	)
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return readElasticError(resp)
	}

	return nil
}

func (s *ProductSearchService) DeleteProduct(ctx context.Context, productID uint) error {
	if !s.isEnabled() {
		return nil
	}

	req, err := http.NewRequestWithContext(
		ctx,
		http.MethodDelete,
		s.endpoint(s.indexName, "_doc", strconv.FormatUint(uint64(productID), 10)),
		nil,
	)
	if err != nil {
		return err
	}

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return nil
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return readElasticError(resp)
	}

	return nil
}

func (s *ProductSearchService) ReindexAll(ctx context.Context) error {
	if !s.isEnabled() {
		return nil
	}

	var products []models.Product
	if err := database.DB.Find(&products).Error; err != nil {
		return err
	}

	for _, product := range products {
		if err := s.IndexProduct(ctx, product); err != nil {
			return err
		}
	}

	return nil
}

func (s *ProductSearchService) isEnabled() bool {
	return s != nil && s.enabled && s.httpClient != nil && s.baseURL != ""
}

func (s *ProductSearchService) endpoint(parts ...string) string {
	base := s.baseURL
	for _, part := range parts {
		base += "/" + url.PathEscape(part)
	}
	return base
}

func (s *ProductSearchService) buildSearchBody(params ProductSearchParams) ([]byte, error) {
	limit := params.Limit
	if limit <= 0 {
		limit = 20
	}
	if limit > 100 {
		limit = 100
	}

	offset := params.Offset
	if offset < 0 {
		offset = 0
	}

	filter := make([]map[string]any, 0, 4)
	if params.Type != "" {
		filter = append(filter, map[string]any{
			"term": map[string]any{
				"type.keyword": params.Type,
			},
		})
	}
	if params.Category != "" {
		filter = append(filter, map[string]any{
			"term": map[string]any{
				"categories.keyword": params.Category,
			},
		})
	}
	if params.MinCost != nil || params.MaxCost != nil {
		rangeQuery := map[string]any{}
		if params.MinCost != nil {
			rangeQuery["gte"] = *params.MinCost
		}
		if params.MaxCost != nil {
			rangeQuery["lte"] = *params.MaxCost
		}
		filter = append(filter, map[string]any{
			"range": map[string]any{
				"cost": rangeQuery,
			},
		})
	}

	must := make([]map[string]any, 0, 1)
	if params.Query != "" {
		must = append(must, map[string]any{
			"multi_match": map[string]any{
				"query":  params.Query,
				"fields": []string{"title^4", "description^2", "categories^3", "type^2", "composition", "address"},
			},
		})
	}

	query := map[string]any{
		"bool": map[string]any{
			"filter": filter,
		},
	}
	if len(must) > 0 {
		query["bool"].(map[string]any)["must"] = must
	}

	payload := map[string]any{
		"from":  offset,
		"size":  limit,
		"query": query,
		"sort": []map[string]any{
			{
				"_score": map[string]any{"order": "desc"},
			},
			{
				"view_count": map[string]any{"order": "desc"},
			},
			{
				"created_at": map[string]any{"order": "desc"},
			},
		},
	}

	return json.Marshal(payload)
}

func (s *ProductSearchService) searchProductsFallback(ctx context.Context, params ProductSearchParams) ([]uint, error) {
	query := database.DB.WithContext(ctx).Model(&models.Product{})

	if params.Query != "" {
		searchTerm := "%" + strings.ToLower(params.Query) + "%"
		query = query.Where(
			"LOWER(title) LIKE ? OR LOWER(description) LIKE ? OR LOWER(type) LIKE ? OR LOWER(address) LIKE ? OR LOWER(composition) LIKE ? OR CAST(categories AS TEXT) LIKE ?",
			searchTerm, searchTerm, searchTerm, searchTerm, searchTerm, searchTerm,
		)
	}

	if params.Type != "" {
		query = query.Where("type = ?", params.Type)
	}
	if params.Category != "" {
		categoryTerm := "%" + strings.ToLower(params.Category) + "%"
		query = query.Where("LOWER(CAST(categories AS TEXT)) LIKE ?", categoryTerm)
	}
	if params.MinCost != nil {
		query = query.Where("cost >= ?", *params.MinCost)
	}
	if params.MaxCost != nil {
		query = query.Where("cost <= ?", *params.MaxCost)
	}

	limit := params.Limit
	if limit <= 0 {
		limit = 20
	}
	if limit > 100 {
		limit = 100
	}

	offset := params.Offset
	if offset < 0 {
		offset = 0
	}

	var products []models.Product
	if err := query.Order("view_count DESC").Order("created_at DESC").Limit(limit).Offset(offset).Find(&products).Error; err != nil {
		return nil, err
	}

	ids := make([]uint, 0, len(products))
	for _, product := range products {
		ids = append(ids, product.ID)
	}

	return ids, nil
}

func buildProductDocument(product models.Product) productDocument {
	return productDocument{
		ID:            product.ID,
		SellerID:      product.SellerID,
		Title:         product.Title,
		Description:   product.Description,
		Type:          product.Type,
		Address:       product.Address,
		Categories:    product.Categories,
		Cost:          product.Cost,
		ViewCount:     product.ViewCount,
		AverageRating: product.AverageRating,
		Composition:   product.Composition,
		Weight:        product.Weight,
		CreatedAt:     product.CreatedAt,
	}
}

func readElasticError(resp *http.Response) error {
	body, _ := io.ReadAll(resp.Body)
	if len(body) == 0 {
		return fmt.Errorf("elasticsearch request failed with status %d", resp.StatusCode)
	}
	return fmt.Errorf("elasticsearch request failed with status %d: %s", resp.StatusCode, strings.TrimSpace(string(body)))
}

const indexMapping = `{
  "settings": {
    "analysis": {
      "normalizer": {
        "lowercase_normalizer": {
          "type": "custom",
          "char_filter": [],
          "filter": ["lowercase", "asciifolding"]
        }
      }
    }
  },
  "mappings": {
    "properties": {
      "id": { "type": "keyword" },
      "seller_id": { "type": "keyword" },
      "title": {
        "type": "text",
        "fields": {
          "keyword": {
            "type": "keyword",
            "normalizer": "lowercase_normalizer"
          }
        }
      },
      "description": { "type": "text" },
      "type": {
        "type": "text",
        "fields": {
          "keyword": {
            "type": "keyword",
            "normalizer": "lowercase_normalizer"
          }
        }
      },
      "address": { "type": "text" },
      "categories": {
        "type": "text",
        "fields": {
          "keyword": {
            "type": "keyword",
            "normalizer": "lowercase_normalizer"
          }
        }
      },
      "cost": { "type": "double" },
      "view_count": { "type": "long" },
      "average_rating": { "type": "double" },
      "composition": { "type": "text" },
      "weight": { "type": "keyword" },
      "created_at": { "type": "date" }
    }
  }
}`

func IndexProductAsync(product models.Product) {
	if ProductSearch == nil || !ProductSearch.enabled {
		return
	}

	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), defaultSearchTimeout)
		defer cancel()

		if err := ProductSearch.IndexProduct(ctx, product); err != nil {
			log.Printf("failed to index product %d: %v", product.ID, err)
		}
	}()
}

func DeleteProductFromSearchAsync(productID uint) {
	if ProductSearch == nil || !ProductSearch.enabled {
		return
	}

	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), defaultSearchTimeout)
		defer cancel()

		if err := ProductSearch.DeleteProduct(ctx, productID); err != nil && err != gorm.ErrRecordNotFound {
			log.Printf("failed to delete product %d from search index: %v", productID, err)
		}
	}()
}

func ParseSearchFloat(value string) (*float64, error) {
	if strings.TrimSpace(value) == "" {
		return nil, nil
	}

	parsed, err := strconv.ParseFloat(value, 64)
	if err != nil {
		return nil, err
	}

	return &parsed, nil
}

func SearchEnabled() bool {
	return ProductSearch != nil && ProductSearch.enabled
}

func ElasticSearchEnvSummary() string {
	return fmt.Sprintf(
		"ELASTICSEARCH_ENABLED=%s ELASTICSEARCH_URL=%s ELASTICSEARCH_INDEX=%s",
		os.Getenv("ELASTICSEARCH_ENABLED"),
		os.Getenv("ELASTICSEARCH_URL"),
		os.Getenv("ELASTICSEARCH_INDEX"),
	)
}
