package product

import (
	"context"
	"math"
	"time"

	"ozMadeBack/internal/database"
	"ozMadeBack/internal/models"
	redisrepo "ozMadeBack/internal/repository/redis"

	"gorm.io/gorm"
)

const (
	defaultTrendingLimit = 20
	defaultProductTTL    = 15 * time.Minute
)

type cacheRepository interface {
	GetProduct(ctx context.Context, productID uint) (*models.Product, error)
	SetProduct(ctx context.Context, product models.Product) error
	DeleteProduct(ctx context.Context, productID uint) error
	IncrementTrendingScore(ctx context.Context, productID uint, delta float64) error
	GetTrendingProductIDs(ctx context.Context, start, stop int64) ([]uint, error)
	ReplaceTrendingScores(ctx context.Context, scores map[uint]float64) error
}

type Service struct {
	db    *gorm.DB
	cache cacheRepository
	now   func() time.Time
}

func NewService(db *gorm.DB, cache cacheRepository) *Service {
	return &Service{
		db:    db,
		cache: cache,
		now:   time.Now,
	}
}

func NewDefaultService() *Service {
	return NewService(
		database.DB,
		redisrepo.NewCacheRepository(database.RDB, defaultProductTTL),
	)
}

func (s *Service) GetProduct(ctx context.Context, productID uint) (models.Product, error) {
	if s.cache != nil {
		cachedProduct, err := s.cache.GetProduct(ctx, productID)
		if err == nil && cachedProduct != nil {
			return *cachedProduct, nil
		}
	}

	var product models.Product
	if err := s.db.Preload("Comments").First(&product, productID).Error; err != nil {
		return models.Product{}, err
	}

	if s.cache != nil {
		_ = s.cache.SetProduct(ctx, product)
	}

	return product, nil
}

func (s *Service) IncrementView(ctx context.Context, productID uint) error {
	tx := s.db.Model(&models.Product{}).
		Where("id = ?", productID).
		UpdateColumn("view_count", gorm.Expr("view_count + ?", 1))
	if tx.Error != nil {
		return tx.Error
	}
	if tx.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}

	if s.cache != nil {
		_ = s.cache.DeleteProduct(ctx, productID)
		_ = s.cache.IncrementTrendingScore(ctx, productID, 1)
	}

	return nil
}

func (s *Service) IncrementOrderCount(ctx context.Context, productID uint) error {
	tx := s.db.Model(&models.Product{}).
		Where("id = ?", productID).
		UpdateColumn("orders_count", gorm.Expr("orders_count + ?", 1))
	if tx.Error != nil {
		return tx.Error
	}
	if tx.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}

	if s.cache != nil {
		_ = s.cache.DeleteProduct(ctx, productID)
	}

	return nil
}

func (s *Service) GetTrendingProducts(ctx context.Context, limit int) ([]models.Product, error) {
	if limit <= 0 {
		limit = defaultTrendingLimit
	}

	if s.cache == nil {
		return []models.Product{}, nil
	}

	ids, err := s.cache.GetTrendingProductIDs(ctx, 0, int64(limit-1))
	if err != nil {
		return nil, err
	}
	if len(ids) == 0 {
		return []models.Product{}, nil
	}

	var products []models.Product
	if err := s.db.Where("id IN ?", ids).Find(&products).Error; err != nil {
		return nil, err
	}

	productsByID := make(map[uint]models.Product, len(products))
	for _, product := range products {
		productsByID[product.ID] = product
		if s.cache != nil {
			_ = s.cache.SetProduct(ctx, product)
		}
	}

	orderedProducts := make([]models.Product, 0, len(ids))
	for _, id := range ids {
		product, ok := productsByID[id]
		if !ok {
			continue
		}
		orderedProducts = append(orderedProducts, product)
	}

	return orderedProducts, nil
}

func (s *Service) RefreshTrendingScores(ctx context.Context) error {
	if s.cache == nil {
		return nil
	}

	var products []models.Product
	if err := s.db.Find(&products).Error; err != nil {
		return err
	}

	scores := make(map[uint]float64, len(products))
	now := s.now()
	for _, product := range products {
		scores[product.ID] = TrendingScore(product.ViewCount, product.CreatedAt, now)
	}

	return s.cache.ReplaceTrendingScores(ctx, scores)
}

func TrendingScore(viewCount int64, createdAt time.Time, now time.Time) float64 {
	hoursOld := now.Sub(createdAt).Hours()
	if hoursOld < 0 {
		hoursOld = 0
	}

	return float64(viewCount) / math.Pow(hoursOld+2, 1.8)
}
