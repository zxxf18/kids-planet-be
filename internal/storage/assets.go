package storage

import (
	"context"
	"fmt"
	"net/url"
	"path"
	"path/filepath"
	"strings"
	"time"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
	"github.com/zxxf18/kids-planet-be/internal/config"
	"github.com/zxxf18/kids-planet-be/internal/media"
)

type Assets struct {
	mode          string
	resourceRoot  string
	generatedRoot string
	minio         *minio.Client
	bucket        string
	prefix        string
	expiry        time.Duration
}

func NewAssets(cfg config.Config, prober *media.Prober) (*Assets, error) {
	assets := &Assets{
		mode: cfg.Storage.Mode, resourceRoot: prober.Root(), generatedRoot: prober.GeneratedRoot(),
		bucket: cfg.Storage.MinIO.Bucket, prefix: strings.Trim(cfg.Storage.MinIO.Prefix, "/"),
		expiry: time.Duration(cfg.Storage.MinIO.URLExpiryMinutes) * time.Minute,
	}
	if assets.expiry <= 0 {
		assets.expiry = time.Hour
	}
	if assets.mode == "" || assets.mode == "local" {
		assets.mode = "local"
		return assets, nil
	}
	if assets.mode != "minio" {
		return nil, fmt.Errorf("unsupported storage mode %q", assets.mode)
	}
	if cfg.Storage.MinIO.Endpoint == "" || cfg.Storage.MinIO.AccessKey == "" || cfg.Storage.MinIO.SecretKey == "" || assets.bucket == "" {
		return nil, fmt.Errorf("minio mode requires endpoint, access key, secret key and bucket")
	}
	client, err := minio.New(cfg.Storage.MinIO.Endpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(cfg.Storage.MinIO.AccessKey, cfg.Storage.MinIO.SecretKey, ""),
		Secure: cfg.Storage.MinIO.UseSSL,
	})
	if err != nil {
		return nil, err
	}
	assets.minio = client
	return assets, nil
}

func (a *Assets) PublishPoster(ctx context.Context, posterKey, localPath string) error {
	if a.mode == "local" {
		return nil
	}
	_, err := a.minio.FPutObject(ctx, a.bucket, a.objectKey(posterKey), localPath, minio.PutObjectOptions{ContentType: "image/jpeg"})
	return err
}

func (a *Assets) LocalPath(kind, objectKey string) (string, error) {
	if a.mode != "local" {
		return "", fmt.Errorf("local path unavailable in %s mode", a.mode)
	}
	root := a.resourceRoot
	clean := filepath.Clean(filepath.FromSlash(objectKey))
	if filepath.IsAbs(clean) || clean == ".." || strings.HasPrefix(clean, ".."+string(filepath.Separator)) {
		return "", fmt.Errorf("invalid object key")
	}
	absRoot, _ := filepath.Abs(root)
	absPath, _ := filepath.Abs(filepath.Join(absRoot, clean))
	rel, err := filepath.Rel(absRoot, absPath)
	if err != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return "", fmt.Errorf("object key escapes storage root")
	}
	return absPath, nil
}

func (a *Assets) PresignedURL(ctx context.Context, objectKey string) (*url.URL, error) {
	if a.mode != "minio" {
		return nil, fmt.Errorf("presigned URL unavailable in local mode")
	}
	return a.minio.PresignedGetObject(ctx, a.bucket, a.objectKey(objectKey), a.expiry, nil)
}

func (a *Assets) IsLocal() bool { return a.mode == "local" }

func (a *Assets) objectKey(key string) string {
	if a.prefix == "" {
		return strings.TrimLeft(key, "/")
	}
	return path.Join(a.prefix, key)
}
