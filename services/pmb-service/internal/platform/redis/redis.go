package redis

import (
	"context"
	"time"

	"github.com/pkm-asia/pmb-backend/services/pmb-service/internal/platform/config"
	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"
)

func ConnectRedis(cfg *config.Config, log *zap.Logger) (*redis.Client, error) {
	if cfg.RedisAddr == "" {
		return nil, nil
	}

	rdb := redis.NewClient(&redis.Options{
		Addr:     cfg.RedisAddr,
		Password: cfg.RedisPassword,
	})

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	if err := rdb.Ping(ctx).Err(); err != nil {
		log.Warn("Koneksi Redis pmb-service belum tersedia: " + err.Error())
		return rdb, err
	}

	return rdb, nil
}
