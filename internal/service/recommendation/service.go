package recommendation

import (
	"context"
	"fmt"
	"math"
	"sort"
	"strings"
	"time"

	"ozMadeBack/internal/database"
	"ozMadeBack/internal/models"
	redisrepo "ozMadeBack/internal/repository/redis"

	"gorm.io/gorm"
)

const (
	defaultRecommendationLimit = 20
	defaultRecommendationTTL   = 30 * time.Minute
	globalRecommendationKey    = "global"
)

type cacheRepository interface {
	GetRecommendationIDs(ctx context.Context, cacheKey string) ([]uint, error)
	SetRecommendationIDs(ctx context.Context, cacheKey string, ids []uint, ttl time.Duration) error
}

type Service struct {
	db    *gorm.DB
	cache cacheRepository
	now   func() time.Time
	ttl   time.Duration
}

type productMetrics struct {
	Product       models.Product
	FavoriteCount int64
	OrderCount    int64
	BaseScore     float64
}

type countRow struct {
	ProductID uint
	Count     int64
}

type userPreferences struct {
	ExcludedProductIDs map[uint]struct{}
	CategoryWeights    map[string]float64
	TypeWeights        map[string]float64
}

type scoredProduct struct {
	ProductID uint
	Score     float64
}

func NewService(db *gorm.DB, cache cacheRepository, ttl time.Duration) *Service {
	if ttl <= 0 {
		ttl = defaultRecommendationTTL
	}

	return &Service{
		db:    db,
		cache: cache,
		now:   time.Now,
		ttl:   ttl,
	}
}

func NewDefaultService() *Service {
	return NewService(
		database.DB,
		redisrepo.NewCacheRepository(database.RDB, 15*time.Minute),
		defaultRecommendationTTL,
	)
}

func (s *Service) GetRecommendationsForUser(ctx context.Context, userID uint, limit int) ([]models.Product, error) {
	if limit <= 0 {
		limit = defaultRecommendationLimit
	}

	ids, err := s.getCachedIDs(ctx, userRecommendationKey(userID))
	if err != nil {
		return nil, err
	}
	if len(ids) == 0 {
		if err := s.RefreshRecommendationsForUser(ctx, userID, limit); err != nil {
			return nil, err
		}
		ids, err = s.getCachedIDs(ctx, userRecommendationKey(userID))
		if err != nil {
			return nil, err
		}
	}

	if len(ids) == 0 {
		if err := s.RefreshGlobalRecommendations(ctx, limit); err != nil {
			return nil, err
		}
		ids, err = s.getCachedIDs(ctx, globalRecommendationKey)
		if err != nil {
			return nil, err
		}
	}

	if len(ids) == 0 {
		return []models.Product{}, nil
	}

	if len(ids) > limit {
		ids = ids[:limit]
	}

	return s.loadProductsByOrderedIDs(ctx, ids)
}

func (s *Service) RefreshGlobalRecommendations(ctx context.Context, limit int) error {
	if limit <= 0 {
		limit = defaultRecommendationLimit
	}

	metrics, err := s.loadProductMetrics(ctx)
	if err != nil {
		return err
	}

	sort.Slice(metrics, func(i, j int) bool {
		if metrics[i].BaseScore == metrics[j].BaseScore {
			return metrics[i].Product.ID < metrics[j].Product.ID
		}
		return metrics[i].BaseScore > metrics[j].BaseScore
	})

	ids := make([]uint, 0, min(limit, len(metrics)))
	for _, metric := range metrics {
		ids = append(ids, metric.Product.ID)
		if len(ids) >= limit {
			break
		}
	}

	return s.cache.SetRecommendationIDs(ctx, globalRecommendationKey, ids, s.ttl)
}

func (s *Service) RefreshAllUserRecommendations(ctx context.Context, limit int) error {
	if limit <= 0 {
		limit = defaultRecommendationLimit
	}

	var userIDs []uint
	if err := s.db.WithContext(ctx).Model(&models.User{}).Pluck("id", &userIDs).Error; err != nil {
		return err
	}

	for _, userID := range userIDs {
		if err := s.RefreshRecommendationsForUser(ctx, userID, limit); err != nil {
			return err
		}
	}

	return nil
}

func (s *Service) RefreshRecommendationsForUser(ctx context.Context, userID uint, limit int) error {
	if limit <= 0 {
		limit = defaultRecommendationLimit
	}

	metrics, err := s.loadProductMetrics(ctx)
	if err != nil {
		return err
	}

	preferences, err := s.loadUserPreferences(ctx, userID)
	if err != nil {
		return err
	}

	if len(preferences.CategoryWeights) == 0 && len(preferences.TypeWeights) == 0 {
		return s.RefreshGlobalRecommendations(ctx, limit)
	}

	scored := make([]scoredProduct, 0, len(metrics))
	for _, metric := range metrics {
		if _, excluded := preferences.ExcludedProductIDs[metric.Product.ID]; excluded {
			continue
		}

		score := metric.BaseScore

		for _, category := range metric.Product.Categories {
			score += preferences.CategoryWeights[normalizePreferenceKey(category)] * 3
		}

		score += preferences.TypeWeights[normalizePreferenceKey(metric.Product.Type)] * 2
		if metric.Product.AverageRating > 0 {
			score += metric.Product.AverageRating * 0.5
		}

		scored = append(scored, scoredProduct{
			ProductID: metric.Product.ID,
			Score:     score,
		})
	}

	sort.Slice(scored, func(i, j int) bool {
		if scored[i].Score == scored[j].Score {
			return scored[i].ProductID < scored[j].ProductID
		}
		return scored[i].Score > scored[j].Score
	})

	ids := make([]uint, 0, min(limit, len(scored)))
	for _, item := range scored {
		ids = append(ids, item.ProductID)
		if len(ids) >= limit {
			break
		}
	}

	if len(ids) == 0 {
		return s.cache.SetRecommendationIDs(ctx, userRecommendationKey(userID), []uint{}, s.ttl)
	}

	return s.cache.SetRecommendationIDs(ctx, userRecommendationKey(userID), ids, s.ttl)
}

func (s *Service) getCachedIDs(ctx context.Context, cacheKey string) ([]uint, error) {
	if s.cache == nil {
		return []uint{}, nil
	}

	return s.cache.GetRecommendationIDs(ctx, cacheKey)
}

func (s *Service) loadProductMetrics(ctx context.Context) ([]productMetrics, error) {
	var products []models.Product
	if err := s.db.WithContext(ctx).Find(&products).Error; err != nil {
		return nil, err
	}

	favoriteCounts, err := s.loadCounts(ctx, &models.Favorite{}, "product_id")
	if err != nil {
		return nil, err
	}

	orderCounts, err := s.loadOrderCounts(ctx)
	if err != nil {
		return nil, err
	}

	metrics := make([]productMetrics, 0, len(products))
	now := s.now()
	for _, product := range products {
		favoriteCount := favoriteCounts[product.ID]
		orderCount := orderCounts[product.ID]

		metrics = append(metrics, productMetrics{
			Product:       product,
			FavoriteCount: favoriteCount,
			OrderCount:    orderCount,
			BaseScore:     globalScore(product, favoriteCount, orderCount, now),
		})
	}

	return metrics, nil
}

func (s *Service) loadCounts(ctx context.Context, model any, groupBy string) (map[uint]int64, error) {
	var rows []countRow
	if err := s.db.WithContext(ctx).
		Model(model).
		Select(fmt.Sprintf("%s as product_id, COUNT(*) as count", groupBy)).
		Group(groupBy).
		Scan(&rows).Error; err != nil {
		return nil, err
	}

	counts := make(map[uint]int64, len(rows))
	for _, row := range rows {
		counts[row.ProductID] = row.Count
	}

	return counts, nil
}

func (s *Service) loadOrderCounts(ctx context.Context) (map[uint]int64, error) {
	var rows []countRow
	if err := s.db.WithContext(ctx).
		Model(&models.Order{}).
		Where("status <> ? AND status <> ? AND status <> ?", models.StatusCancelledByBuyer, models.StatusCancelledBySeller, models.StatusExpired).
		Select("product_id as product_id, COUNT(*) as count").
		Group("product_id").
		Scan(&rows).Error; err != nil {
		return nil, err
	}

	counts := make(map[uint]int64, len(rows))
	for _, row := range rows {
		counts[row.ProductID] = row.Count
	}

	return counts, nil
}

func (s *Service) loadUserPreferences(ctx context.Context, userID uint) (userPreferences, error) {
	preferences := userPreferences{
		ExcludedProductIDs: make(map[uint]struct{}),
		CategoryWeights:    make(map[string]float64),
		TypeWeights:        make(map[string]float64),
	}

	favoriteProducts, err := s.loadFavoriteProducts(ctx, userID)
	if err != nil {
		return preferences, err
	}
	for _, product := range favoriteProducts {
		preferences.ExcludedProductIDs[product.ID] = struct{}{}
		addProductPreferences(&preferences, product, 3, 2)
	}

	orderedProducts, err := s.loadOrderedProducts(ctx, userID)
	if err != nil {
		return preferences, err
	}
	for _, product := range orderedProducts {
		preferences.ExcludedProductIDs[product.ID] = struct{}{}
		addProductPreferences(&preferences, product, 2, 1.5)
	}

	ownProductIDs, err := s.loadOwnProductIDs(ctx, userID)
	if err != nil {
		return preferences, err
	}
	for _, productID := range ownProductIDs {
		preferences.ExcludedProductIDs[productID] = struct{}{}
	}

	return preferences, nil
}

func (s *Service) loadFavoriteProducts(ctx context.Context, userID uint) ([]models.Product, error) {
	var favorites []models.Favorite
	if err := s.db.WithContext(ctx).Where("user_id = ?", userID).Find(&favorites).Error; err != nil {
		return nil, err
	}

	productIDs := make([]uint, 0, len(favorites))
	for _, favorite := range favorites {
		productIDs = append(productIDs, favorite.ProductID)
	}

	return s.loadProductsByIDs(ctx, productIDs)
}

func (s *Service) loadOrderedProducts(ctx context.Context, userID uint) ([]models.Product, error) {
	var orders []models.Order
	if err := s.db.WithContext(ctx).
		Where("user_id = ? AND status <> ? AND status <> ? AND status <> ?", userID, models.StatusCancelledByBuyer, models.StatusCancelledBySeller, models.StatusExpired).
		Find(&orders).Error; err != nil {
		return nil, err
	}

	productIDs := make([]uint, 0, len(orders))
	for _, order := range orders {
		productIDs = append(productIDs, order.ProductID)
	}

	return s.loadProductsByIDs(ctx, productIDs)
}

func (s *Service) loadOwnProductIDs(ctx context.Context, userID uint) ([]uint, error) {
	var seller models.Seller
	if err := s.db.WithContext(ctx).Where("user_id = ?", userID).First(&seller).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return []uint{}, nil
		}
		return nil, err
	}

	var productIDs []uint
	if err := s.db.WithContext(ctx).Model(&models.Product{}).Where("seller_id = ?", seller.ID).Pluck("id", &productIDs).Error; err != nil {
		return nil, err
	}

	return productIDs, nil
}

func (s *Service) loadProductsByIDs(ctx context.Context, ids []uint) ([]models.Product, error) {
	if len(ids) == 0 {
		return []models.Product{}, nil
	}

	uniqueIDs := uniqueUint(ids)
	var products []models.Product
	if err := s.db.WithContext(ctx).Where("id IN ?", uniqueIDs).Find(&products).Error; err != nil {
		return nil, err
	}

	return products, nil
}

func (s *Service) loadProductsByOrderedIDs(ctx context.Context, ids []uint) ([]models.Product, error) {
	products, err := s.loadProductsByIDs(ctx, ids)
	if err != nil {
		return nil, err
	}

	productsByID := make(map[uint]models.Product, len(products))
	for _, product := range products {
		productsByID[product.ID] = product
	}

	ordered := make([]models.Product, 0, len(ids))
	for _, id := range ids {
		product, ok := productsByID[id]
		if !ok {
			continue
		}
		ordered = append(ordered, product)
	}

	return ordered, nil
}

func addProductPreferences(preferences *userPreferences, product models.Product, categoryWeight float64, typeWeight float64) {
	for _, category := range product.Categories {
		preferences.CategoryWeights[normalizePreferenceKey(category)] += categoryWeight
	}

	typeKey := normalizePreferenceKey(product.Type)
	if typeKey != "" {
		preferences.TypeWeights[typeKey] += typeWeight
	}
}

func globalScore(product models.Product, favoriteCount int64, orderCount int64, now time.Time) float64 {
	hoursOld := now.Sub(product.CreatedAt).Hours()
	if hoursOld < 0 {
		hoursOld = 0
	}

	recencyBoost := 5 / (1 + hoursOld/24)
	viewScore := math.Log1p(float64(product.ViewCount)) * 1.5
	favoriteScore := math.Log1p(float64(favoriteCount)) * 2
	orderScore := math.Log1p(float64(orderCount)) * 2.5
	ratingScore := product.AverageRating

	return viewScore + favoriteScore + orderScore + ratingScore + recencyBoost
}

func userRecommendationKey(userID uint) string {
	return fmt.Sprintf("user:%d", userID)
}

func normalizePreferenceKey(value string) string {
	return strings.ToLower(strings.TrimSpace(value))
}

func uniqueUint(ids []uint) []uint {
	seen := make(map[uint]struct{}, len(ids))
	result := make([]uint, 0, len(ids))
	for _, id := range ids {
		if _, exists := seen[id]; exists {
			continue
		}
		seen[id] = struct{}{}
		result = append(result, id)
	}
	return result
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
