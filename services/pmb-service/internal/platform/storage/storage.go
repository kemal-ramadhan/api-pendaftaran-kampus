package storage

import (
	"context"
	"fmt"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
	"github.com/pkm-asia/pmb-backend/services/pmb-service/internal/platform/config"
	"go.uber.org/zap"
)

func ConnectMinio(cfg *config.Config, log *zap.Logger) (*minio.Client, error) {
	if cfg.MinioEndpoint == "" {
		return nil, nil
	}

	client, err := minio.New(cfg.MinioEndpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(cfg.MinioAccessKey, cfg.MinioSecretKey, ""),
		Secure: cfg.MinioUseSSL,
	})
	if err != nil {
		return nil, fmt.Errorf("gagal membuat client minio: %w", err)
	}

	// Pastikan bucket default ada
	defaultBuckets := []string{
		"pmb-admission-documents",
		"pmb-identity-photos",
		"pmb-identity-ktp",
		"pmb-finance-proofs",
		"pmb-finance-receipts",
		"pmb-exam-cards",
	}

	ctx := context.Background()
	for _, bucket := range defaultBuckets {
		exists, err := client.BucketExists(ctx, bucket)
		if err == nil && !exists {
			_ = client.MakeBucket(ctx, bucket, minio.MakeBucketOptions{})
		}
	}

	return client, nil
}
