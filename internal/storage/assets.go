package storage

import (
	"context"
	"fmt"
	"io/fs"
	"net/url"
	"path"
	"path/filepath"
	"sort"
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
	prober        *media.Prober
	minio         *minio.Client
	buckets       map[string]string
	prefix        string
	expiry        time.Duration
}

func NewAssets(cfg config.Config, prober *media.Prober) (*Assets, error) {
	assets := &Assets{
		mode:          cfg.Storage.Mode,
		resourceRoot:  prober.Root(),
		generatedRoot: prober.GeneratedRoot(),
		prober:        prober,
		prefix:        strings.Trim(cfg.Storage.MinIO.Prefix, "/"),
		expiry:        time.Duration(cfg.Storage.MinIO.URLExpiryMinutes) * time.Minute,
		buckets: map[string]string{
			"audio":           cfg.Storage.MinIO.Buckets.Audio,
			"video480":        cfg.Storage.MinIO.Buckets.Video480,
			"video720":        cfg.Storage.MinIO.Buckets.Video720,
			"lyricsEn":        cfg.Storage.MinIO.Buckets.LyricsEn,
			"lyricsZh":        cfg.Storage.MinIO.Buckets.LyricsZh,
			"lyricsBilingual": cfg.Storage.MinIO.Buckets.LyricsBilingual,
			"poster":          cfg.Storage.MinIO.Buckets.Poster,
		},
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
	if cfg.Storage.MinIO.Endpoint == "" || cfg.Storage.MinIO.AccessKey == "" || cfg.Storage.MinIO.SecretKey == "" {
		return nil, fmt.Errorf("minio mode requires endpoint, access key and secret key")
	}
	for _, kind := range assetKinds() {
		bucket := assets.buckets[kind]
		if strings.TrimSpace(bucket) == "" {
			return nil, fmt.Errorf("minio mode requires a bucket for %s", kind)
		}
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
	_, err := a.minio.FPutObject(ctx, a.bucket("poster"), a.objectKey(posterKey), localPath, minio.PutObjectOptions{ContentType: "image/jpeg"})
	return err
}

func (a *Assets) LocalPath(kind, objectKey string) (string, error) {
	if a.mode != "local" {
		return "", fmt.Errorf("local path unavailable in %s mode", a.mode)
	}
	clean := filepath.Clean(filepath.FromSlash(objectKey))
	if filepath.IsAbs(clean) || clean == ".." || strings.HasPrefix(clean, ".."+string(filepath.Separator)) {
		return "", fmt.Errorf("invalid object key")
	}
	absRoot, _ := filepath.Abs(a.resourceRoot)
	absPath, _ := filepath.Abs(filepath.Join(absRoot, clean))
	rel, err := filepath.Rel(absRoot, absPath)
	if err != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return "", fmt.Errorf("object key escapes storage root")
	}
	return absPath, nil
}

// Open returns an S3 object and its metadata. The caller must close the object.
func (a *Assets) Open(ctx context.Context, kind, key string) (*minio.Object, minio.ObjectInfo, error) {
	if a.mode != "minio" {
		return nil, minio.ObjectInfo{}, fmt.Errorf("S3 object unavailable in local mode")
	}
	bucket := a.bucket(kind)
	if bucket == "" {
		return nil, minio.ObjectInfo{}, fmt.Errorf("unsupported media kind %q", kind)
	}
	object, err := a.minio.GetObject(ctx, bucket, a.objectKey(key), minio.GetObjectOptions{})
	if err != nil {
		return nil, minio.ObjectInfo{}, err
	}
	info, err := object.Stat()
	if err != nil {
		_ = object.Close()
		return nil, minio.ObjectInfo{}, err
	}
	return object, info, nil
}

func (a *Assets) ListResources(ctx context.Context, subPath string) ([]media.ResourceEntry, error) {
	cleanSubPath, err := cleanObjectPath(subPath)
	if err != nil {
		return nil, err
	}
	if a.mode == "local" {
		return a.listLocal(cleanSubPath)
	}
	entries := make([]media.ResourceEntry, 0, 1024)
	for _, kind := range assetKinds() {
		bucket := a.bucket(kind)
		listPrefix := a.objectKey(cleanSubPath)
		for object := range a.minio.ListObjects(ctx, bucket, minio.ListObjectsOptions{Prefix: listPrefix, Recursive: true}) {
			if object.Err != nil {
				return nil, fmt.Errorf("list bucket %s: %w", bucket, object.Err)
			}
			key := a.relativeKey(object.Key)
			if key == "" || strings.HasSuffix(key, "/") {
				continue
			}
			entries = append(entries, media.ResourceEntry{Kind: kind, Key: key, Name: path.Base(key)})
		}
	}
	sort.Slice(entries, func(i, j int) bool {
		if entries[i].Kind == entries[j].Kind {
			return entries[i].Key < entries[j].Key
		}
		return entries[i].Kind < entries[j].Kind
	})
	return entries, nil
}

func (a *Assets) ProbeResource(ctx context.Context, kind, key string) (media.ProbeResult, error) {
	if a.mode == "local" {
		return a.prober.Probe(ctx, key)
	}
	bucket := a.bucket(kind)
	if bucket == "" {
		return media.ProbeResult{}, fmt.Errorf("unsupported media kind %q", kind)
	}
	signed, err := a.minio.PresignedGetObject(ctx, bucket, a.objectKey(key), a.expiry, url.Values{})
	if err != nil {
		return media.ProbeResult{}, err
	}
	return a.prober.ProbeURL(ctx, signed.String(), key)
}

func (a *Assets) IsLocal() bool { return a.mode == "local" }

func (a *Assets) listLocal(subPath string) ([]media.ResourceEntry, error) {
	base, err := a.prober.ResolveResource(subPath)
	if err != nil {
		return nil, err
	}
	entries := make([]media.ResourceEntry, 0, 1024)
	err = filepath.WalkDir(base, func(current string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.IsDir() || strings.HasPrefix(entry.Name(), ".") {
			return nil
		}
		relative, err := filepath.Rel(a.resourceRoot, current)
		if err != nil {
			return err
		}
		key := filepath.ToSlash(relative)
		kind := localKind(key)
		if kind != "" {
			entries = append(entries, media.ResourceEntry{Kind: kind, Key: key, Name: entry.Name()})
		}
		return nil
	})
	return entries, err
}

func localKind(key string) string {
	ext := strings.ToLower(path.Ext(key))
	parent := strings.ToLower(path.Base(path.Dir(key)))
	switch ext {
	case ".mp3":
		return "audio"
	case ".mp4":
		if parent == "video_480" || parent == "video-480" {
			return "video480"
		}
		if parent == "video_720" || parent == "video-720" {
			return "video720"
		}
		return ""
	case ".lrc", ".txt":
		switch parent {
		case "lyrs", "lyrics_en", "lyrics-en":
			return "lyricsEn"
		case "lyrs_zh", "lyrs-zh", "lyrics_zh", "lyrics-zh":
			return "lyricsZh"
		case "lyrs_bilingual", "lyrs-bilingual", "lyrics_bilingual", "lyrics-bilingual":
			return "lyricsBilingual"
		default:
			return ""
		}
	case ".jpg", ".jpeg", ".png", ".webp":
		if parent == "poster" || parent == "cover" || parent == "covers" {
			return "poster"
		}
		return ""
	default:
		return ""
	}
}

func assetKinds() []string {
	return []string{"audio", "video480", "video720", "lyricsEn", "lyricsZh", "lyricsBilingual", "poster"}
}

func (a *Assets) bucket(kind string) string { return strings.TrimSpace(a.buckets[kind]) }

func (a *Assets) objectKey(key string) string {
	key = strings.TrimLeft(key, "/")
	if a.prefix == "" {
		return key
	}
	return path.Join(a.prefix, key)
}

func (a *Assets) relativeKey(key string) string {
	key = strings.TrimLeft(key, "/")
	if a.prefix == "" {
		return key
	}
	return strings.TrimPrefix(key, a.prefix+"/")
}

func cleanObjectPath(value string) (string, error) {
	value = strings.TrimSpace(strings.ReplaceAll(value, "\\", "/"))
	if value == "" || value == "." {
		return "", nil
	}
	if value == ".." || strings.HasPrefix(value, "../") || strings.Contains(value, "/../") {
		return "", fmt.Errorf("resourcePath escapes configured prefix")
	}
	clean := path.Clean("/" + value)
	return strings.TrimPrefix(clean, "/"), nil
}
