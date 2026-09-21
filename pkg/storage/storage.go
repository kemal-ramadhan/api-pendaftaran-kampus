package storage

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
)

var (
	ErrBucketEmpty         = errors.New("nama bucket tidak boleh kosong")
	ErrObjectKeyEmpty      = errors.New("object key tidak boleh kosong")
	ErrFileTypeNotAllowed  = errors.New("tipe file tidak diizinkan")
	ErrFileTooLarge        = errors.New("ukuran file melebihi batas yang diizinkan")
)

// Config konfigurasi koneksi MinIO/S3
type Config struct {
	Endpoint        string // mis. "minio:9000" atau "127.0.0.1:9000"
	AccessKeyID     string
	SecretAccessKey string
	UseSSL          bool
	PublicEndpoint  string // opsional untuk presign URL host eksternal (mis. "localhost:9000")
}

// Client interface untuk operasi penyimpanan objek MinIO
type Client interface {
	EnsureBucket(ctx context.Context, bucketName string) error
	Upload(ctx context.Context, bucketName, objectKey string, reader io.Reader, size int64, contentType string) error
	Download(ctx context.Context, bucketName, objectKey string) (io.ReadCloser, error)
	PresignedGetObject(ctx context.Context, bucketName, objectKey string, expiry time.Duration) (string, error)
	PresignedPutObject(ctx context.Context, bucketName, objectKey string, expiry time.Duration) (string, error)
	DeleteObject(ctx context.Context, bucketName, objectKey string) error
	SniffAndValidate(reader io.Reader, allowedTypes []string) (io.Reader, string, error)
}

type minioStorage struct {
	client       *minio.Client
	presignClient *minio.Client
	cfg          Config
}

// NewClient membuat instance baru MinIO Storage Client
func NewClient(cfg Config) (Client, error) {
	if cfg.Endpoint == "" {
		return nil, errors.New("minio endpoint tidak boleh kosong")
	}

	cli, err := minio.New(cfg.Endpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(cfg.AccessKeyID, cfg.SecretAccessKey, ""),
		Secure: cfg.UseSSL,
	})
	if err != nil {
		return nil, fmt.Errorf("gagal inisialisasi minio client: %w", err)
	}

	presignCli := cli
	if cfg.PublicEndpoint != "" && cfg.PublicEndpoint != cfg.Endpoint {
		pCli, err := minio.New(cfg.PublicEndpoint, &minio.Options{
			Creds:  credentials.NewStaticV4(cfg.AccessKeyID, cfg.SecretAccessKey, ""),
			Secure: cfg.UseSSL,
		})
		if err == nil {
			presignCli = pCli
		}
	}

	return &minioStorage{
		client:        cli,
		presignClient: presignCli,
		cfg:           cfg,
	}, nil
}

// EnsureBucket memeriksa dan membuat bucket jika belum ada
func (s *minioStorage) EnsureBucket(ctx context.Context, bucketName string) error {
	if bucketName == "" {
		return ErrBucketEmpty
	}

	exists, err := s.client.BucketExists(ctx, bucketName)
	if err != nil {
		return fmt.Errorf("gagal memeriksa keberadaan bucket %s: %w", bucketName, err)
	}

	if !exists {
		err = s.client.MakeBucket(ctx, bucketName, minio.MakeBucketOptions{})
		if err != nil {
			return fmt.Errorf("gagal membuat bucket %s: %w", bucketName, err)
		}
	}
	return nil
}

// Upload mengunggah objek ke bucket MinIO
func (s *minioStorage) Upload(ctx context.Context, bucketName, objectKey string, reader io.Reader, size int64, contentType string) error {
	if bucketName == "" {
		return ErrBucketEmpty
	}
	if objectKey == "" {
		return ErrObjectKeyEmpty
	}

	if contentType == "" {
		contentType = "application/octet-stream"
	}

	_, err := s.client.PutObject(ctx, bucketName, objectKey, reader, size, minio.PutObjectOptions{
		ContentType: contentType,
	})
	if err != nil {
		return fmt.Errorf("gagal upload object %s ke bucket %s: %w", objectKey, bucketName, err)
	}

	return nil
}

// Download mengambil stream file dari bucket MinIO
func (s *minioStorage) Download(ctx context.Context, bucketName, objectKey string) (io.ReadCloser, error) {
	if bucketName == "" {
		return nil, ErrBucketEmpty
	}
	if objectKey == "" {
		return nil, ErrObjectKeyEmpty
	}

	obj, err := s.client.GetObject(ctx, bucketName, objectKey, minio.GetObjectOptions{})
	if err != nil {
		return nil, fmt.Errorf("gagal download object %s: %w", objectKey, err)
	}

	// Verifikasi apakah objek benar-benar ditemukan
	_, err = obj.Stat()
	if err != nil {
		_ = obj.Close()
		return nil, fmt.Errorf("object %s tidak ditemukan atau tidak dapat diakses: %w", objectKey, err)
	}

	return obj, nil
}

// PresignedGetObject membuat URL berumur sementara untuk mengunduh berkas privat
func (s *minioStorage) PresignedGetObject(ctx context.Context, bucketName, objectKey string, expiry time.Duration) (string, error) {
	if bucketName == "" {
		return "", ErrBucketEmpty
	}
	if objectKey == "" {
		return "", ErrObjectKeyEmpty
	}
	if expiry <= 0 {
		expiry = 5 * time.Minute
	}

	reqParams := make(url.Values)
	u, err := s.presignClient.PresignedGetObject(ctx, bucketName, objectKey, expiry, reqParams)
	if err != nil {
		return "", fmt.Errorf("gagal membuat presigned get object URL: %w", err)
	}

	return u.String(), nil
}

// PresignedPutObject membuat URL berumur sementara untuk direct upload dari klien
func (s *minioStorage) PresignedPutObject(ctx context.Context, bucketName, objectKey string, expiry time.Duration) (string, error) {
	if bucketName == "" {
		return "", ErrBucketEmpty
	}
	if objectKey == "" {
		return "", ErrObjectKeyEmpty
	}
	if expiry <= 0 {
		expiry = 10 * time.Minute
	}

	u, err := s.presignClient.PresignedPutObject(ctx, bucketName, objectKey, expiry)
	if err != nil {
		return "", fmt.Errorf("gagal membuat presigned put object URL: %w", err)
	}

	return u.String(), nil
}

// DeleteObject menghapus objek dari bucket MinIO
func (s *minioStorage) DeleteObject(ctx context.Context, bucketName, objectKey string) error {
	if bucketName == "" {
		return ErrBucketEmpty
	}
	if objectKey == "" {
		return ErrObjectKeyEmpty
	}

	err := s.client.RemoveObject(ctx, bucketName, objectKey, minio.RemoveObjectOptions{})
	if err != nil {
		return fmt.Errorf("gagal menghapus object %s: %w", objectKey, err)
	}

	return nil
}

// SniffAndValidate membaca 512 byte pertama dari stream untuk mendeteksi MIME type sebenarnya
// dan mengembalikan stream baru yang utuh beserta mime type yang tervalidasi.
func (s *minioStorage) SniffAndValidate(reader io.Reader, allowedTypes []string) (io.Reader, string, error) {
	buf := make([]byte, 512)
	n, err := io.ReadFull(reader, buf)
	if err != nil && err != io.EOF && err != io.ErrUnexpectedEOF {
		return nil, "", fmt.Errorf("gagal membaca header file: %w", err)
	}

	sniffedData := buf[:n]
	detectedMime := http.DetectContentType(sniffedData)

	// Pembersihan parameter charset (mis. "text/plain; charset=utf-8" -> "text/plain")
	cleanMime := strings.Split(detectedMime, ";")[0]
	cleanMime = strings.TrimSpace(cleanMime)

	// Validasi against allowedTypes
	if len(allowedTypes) > 0 {
		allowed := false
		for _, at := range allowedTypes {
			if strings.EqualFold(cleanMime, strings.TrimSpace(at)) {
				allowed = true
				break
			}
		}
		if !allowed {
			return nil, cleanMime, fmt.Errorf("%w: terdeteksi %s, diizinkan: %v", ErrFileTypeNotAllowed, cleanMime, allowedTypes)
		}
	}

	// Gabungkan kembali header yang sudah dibaca dengan sisa reader
	fullReader := io.MultiReader(bytes.NewReader(sniffedData), reader)
	return fullReader, cleanMime, nil
}
