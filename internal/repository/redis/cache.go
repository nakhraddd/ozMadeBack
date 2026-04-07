package redis

import (
	"context"
	"encoding/json"
	"strconv"
	"time"

	"ozMadeBack/internal/models"

	goredis "github.com/go-redis/redis/v8"
)

const (
	trendingProductsKey = "trending_products"
	productCachePrefix  = "product:"
	defaultProductTTL   = 15 * time.Minute
)

type CacheRepository struct {
	client     *goredis.Client
	productTTL time.Duration
}

func NewCacheRepository(client *goredis.Client, productTTL time.Duration) *CacheRepository {
	if productTTL <= 0 {
		productTTL = defaultProductTTL
	}

	return &CacheRepository{
		client:     client,
		productTTL: productTTL,
	}
}

func (r *CacheRepository) GetProduct(ctx context.Context, productID uint) (*models.Product, error) {
	if r == nil || r.client == nil {
		return nil, nil
	}

	payload, err := r.client.Get(ctx, productCacheKey(productID)).Bytes()
	if err == goredis.Nil {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	var product models.Product
	if err := json.Unmarshal(payload, &product); err != nil {
		return nil, err
	}

	return &product, nil
}

func (r *CacheRepository) SetProduct(ctx context.Context, product models.Product) error {
	if r == nil || r.client == nil {
		return nil
	}

	payload, err := json.Marshal(product)
	if err != nil {
		return err
	}

	return r.client.Set(ctx, productCacheKey(product.ID), payload, r.productTTL).Err()
}

func (r *CacheRepository) DeleteProduct(ctx context.Context, productID uint) error {
	if r == nil || r.client == nil {
		return nil
	}

	return r.client.Del(ctx, productCacheKey(productID)).Err()
}

func (r *CacheRepository) IncrementTrendingScore(ctx context.Context, productID uint, delta float64) error {
	if r == nil || r.client == nil {
		return nil
	}

	return r.client.ZIncrBy(ctx, trendingProductsKey, delta, redisMember(productID)).Err()
}

func (r *CacheRepository) GetTrendingProductIDs(ctx context.Context, start, stop int64) ([]uint, error) {
	if r == nil || r.client == nil {
		return []uint{}, nil
	}

	members, err := r.client.ZRevRange(ctx, trendingProductsKey, start, stop).Result()
	if err != nil {
		return nil, err
	}

	ids := make([]uint, 0, len(members))
	for _, member := range members {
		id, err := strconv.ParseUint(member, 10, 64)
		if err != nil {
			continue
		}
		ids = append(ids, uint(id))
	}

	return ids, nil
}

func (r *CacheRepository) ReplaceTrendingScores(ctx context.Context, scores map[uint]float64) error {
	if r == nil || r.client == nil {
		return nil
	}

	pipe := r.client.TxPipeline()
	pipe.Del(ctx, trendingProductsKey)

	if len(scores) > 0 {
		members := make([]*goredis.Z, 0, len(scores))
		for productID, score := range scores {
			members = append(members, &goredis.Z{
				Score:  score,
				Member: redisMember(productID),
			})
		}
		pipe.ZAdd(ctx, trendingProductsKey, members...)
	}

	_, err := pipe.Exec(ctx)
	return err
}

func productCacheKey(productID uint) string {
	return productCachePrefix + redisMember(productID)
}

func redisMember(productID uint) string {
	return strconv.FormatUint(uint64(productID), 10)
}
