package httpapi

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"mime"
	"net/http"
	"path"
	"strconv"
	"strings"
	"time"

	"github.com/zeromicro/go-zero/rest"
	"github.com/zeromicro/go-zero/rest/pathvar"
	"github.com/zxxf18/kids-planet-be/internal/media"
	"github.com/zxxf18/kids-planet-be/internal/model"
	"github.com/zxxf18/kids-planet-be/internal/storage"
	"github.com/zxxf18/kids-planet-be/internal/store"
)

type API struct {
	store          *store.MySQL
	prober         *media.Prober
	scanner        *media.Scanner
	assets         *storage.Assets
	publicBasePath string
}

func New(store *store.MySQL, prober *media.Prober, scanner *media.Scanner, assets *storage.Assets, publicBasePath string) *API {
	publicBasePath = strings.Trim(publicBasePath, "/")
	if publicBasePath != "" {
		publicBasePath = "/" + publicBasePath
	}
	return &API{
		store: store, prober: prober, scanner: scanner, assets: assets,
		publicBasePath: publicBasePath,
	}
}

func (a *API) Register(server *rest.Server) {
	server.AddRoutes([]rest.Route{
		{Method: http.MethodGet, Path: "/api/v1/healthz", Handler: a.health},
		{Method: http.MethodGet, Path: "/api/v1/media", Handler: a.listMedia},
		{Method: http.MethodGet, Path: "/api/v1/tags", Handler: a.listTags},
		{Method: http.MethodGet, Path: "/api/v1/media/:id", Handler: a.getMedia},
		{Method: http.MethodGet, Path: "/api/v1/media/:id/content", Handler: a.mediaContent},
		{Method: http.MethodHead, Path: "/api/v1/media/:id/content", Handler: a.mediaContent},
		{Method: http.MethodPost, Path: "/api/v1/admin/probe", Handler: a.probeResource},
	})
	server.AddRoute(
		rest.Route{Method: http.MethodPost, Path: "/api/v1/admin/scan", Handler: a.scanResources},
		rest.WithTimeout(30*time.Minute),
	)
}

func (a *API) health(w http.ResponseWriter, r *http.Request) {
	if err := a.store.Ping(r.Context()); err != nil {
		writeError(w, http.StatusServiceUnavailable, "database unavailable")
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok", "time": time.Now().Format(time.RFC3339)})
}

func (a *API) listMedia(w http.ResponseWriter, r *http.Request) {
	page := intQuery(r, "page", 1)
	pageSize := intQuery(r, "pageSize", 30)
	if pageSize > 50 {
		pageSize = 50
	}
	ids, err := parseIDs(r.URL.Query().Get("ids"))
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	items, total, err := a.store.List(r.Context(), store.ListFilter{
		Query: r.URL.Query().Get("q"), Tag: r.URL.Query().Get("tag"), IDs: ids,
	}, page, pageSize)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	for i := range items {
		a.attachURLs(&items[i])
	}
	writeJSON(w, http.StatusOK, map[string]any{"items": items, "total": total, "page": page, "pageSize": pageSize})
}

func (a *API) listTags(w http.ResponseWriter, r *http.Request) {
	tags, err := a.store.ListTags(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"items": tags})
}

func (a *API) getMedia(w http.ResponseWriter, r *http.Request) {
	item, ok := a.loadMedia(w, r)
	if !ok {
		return
	}
	a.attachURLs(&item)
	writeJSON(w, http.StatusOK, item)
}

func (a *API) mediaContent(w http.ResponseWriter, r *http.Request) {
	item, ok := a.loadMedia(w, r)
	if !ok {
		return
	}
	kind := r.URL.Query().Get("kind")
	storageKind, key, err := objectKey(item, kind, r.URL.Query().Get("quality"), r.URL.Query().Get("language"))
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	if key == "" {
		writeError(w, http.StatusNotFound, "requested media type is unavailable")
		return
	}
	if strings.HasPrefix(storageKind, "lyrics") {
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		w.Header().Set("Cache-Control", "public, max-age=3600")
	}
	if a.assets.IsLocal() {
		localPath, err := a.assets.LocalPath(storageKind, key)
		if err != nil {
			writeError(w, http.StatusBadRequest, err.Error())
			return
		}
		http.ServeFile(w, r, localPath)
		return
	}
	object, info, err := a.assets.Open(r.Context(), storageKind, key)
	if err != nil {
		writeError(w, http.StatusBadGateway, err.Error())
		return
	}
	defer object.Close()
	contentType := info.ContentType
	if contentType == "" || contentType == "application/octet-stream" {
		contentType = mime.TypeByExtension(path.Ext(key))
	}
	if contentType != "" && !strings.HasPrefix(storageKind, "lyrics") {
		w.Header().Set("Content-Type", contentType)
	}
	if info.ETag != "" {
		w.Header().Set("ETag", `"`+strings.Trim(info.ETag, `"`)+`"`)
	}
	if kind == "poster" {
		w.Header().Set("Cache-Control", "public, max-age=86400")
	} else if strings.HasPrefix(storageKind, "lyrics") {
		w.Header().Set("Cache-Control", "public, max-age=86400")
	} else {
		w.Header().Set("Cache-Control", "public, max-age=3600")
	}
	w.Header().Set("Accept-Ranges", "bytes")
	http.ServeContent(w, r, path.Base(key), info.LastModified, object)
}

func (a *API) probeResource(w http.ResponseWriter, r *http.Request) {
	var req struct {
		ResourcePath      string `json:"resourcePath"`
		ExtractFirstFrame bool   `json:"extractFirstFrame"`
	}
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	if strings.TrimSpace(req.ResourcePath) == "" {
		writeError(w, http.StatusBadRequest, "resourcePath is required")
		return
	}
	result, err := a.prober.Probe(r.Context(), req.ResourcePath)
	if err != nil {
		writeError(w, http.StatusUnprocessableEntity, err.Error())
		return
	}
	if req.ExtractFirstFrame {
		if result.VideoCodec == "" {
			writeError(w, http.StatusUnprocessableEntity, "resource has no video stream")
			return
		}
		stem := strings.TrimSuffix(filepathBase(req.ResourcePath), filepathExt(req.ResourcePath))
		posterKey := "probes/" + sanitizeFilename(stem) + ".jpg"
		result.PosterPath, err = a.prober.ExtractFirstFrame(r.Context(), req.ResourcePath, posterKey)
		if err != nil {
			writeError(w, http.StatusUnprocessableEntity, err.Error())
			return
		}
	}
	writeJSON(w, http.StatusOK, result)
}

func (a *API) scanResources(w http.ResponseWriter, r *http.Request) {
	var req struct {
		ResourcePath string `json:"resourcePath"`
	}
	if r.ContentLength != 0 {
		if err := decodeJSON(r, &req); err != nil {
			writeError(w, http.StatusBadRequest, err.Error())
			return
		}
	}
	result, err := a.scanner.Scan(r.Context(), req.ResourcePath)
	if err != nil {
		writeError(w, http.StatusUnprocessableEntity, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, result)
}

func (a *API) loadMedia(w http.ResponseWriter, r *http.Request) (model.MediaItem, bool) {
	id, err := strconv.ParseInt(pathvar.Vars(r)["id"], 10, 64)
	if err != nil || id < 1 {
		writeError(w, http.StatusBadRequest, "invalid media id")
		return model.MediaItem{}, false
	}
	item, err := a.store.Get(r.Context(), id)
	if errors.Is(err, store.ErrNotFound) {
		writeError(w, http.StatusNotFound, "media not found")
		return model.MediaItem{}, false
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return model.MediaItem{}, false
	}
	return item, true
}

func (a *API) attachURLs(item *model.MediaItem) {
	base := fmt.Sprintf("%s/api/v1/media/%d/content?kind=", a.publicBasePath, item.ID)
	if item.HasAudio {
		item.AudioURL = base + "audio"
	}
	if item.HasVideo {
		item.VideoSources = make(map[string]string, 2)
		if item.Video480ObjectKey != nil {
			item.VideoSources["480"] = base + "video&quality=480"
		}
		if item.Video720ObjectKey != nil {
			item.VideoSources["720"] = base + "video&quality=720"
		}
	}
	if item.HasLyrics {
		item.LyricsSources = make(map[string]string, 3)
		if item.LyricsEnObjectKey != nil {
			item.LyricsSources["en"] = base + "lyrics&language=en"
		}
		if item.LyricsZhObjectKey != nil {
			item.LyricsSources["zh"] = base + "lyrics&language=zh"
		}
		if item.LyricsBiObjectKey != nil {
			item.LyricsSources["bilingual"] = base + "lyrics&language=bilingual"
		}
	}
	if item.HasPoster {
		item.PosterURL = base + "poster"
	}
}

func objectKey(item model.MediaItem, kind, quality, language string) (string, string, error) {
	var value *string
	storageKind := kind
	switch kind {
	case "audio":
		value = item.AudioObjectKey
	case "video":
		switch quality {
		case "480":
			storageKind, value = "video480", item.Video480ObjectKey
		case "720":
			storageKind, value = "video720", item.Video720ObjectKey
		default:
			return "", "", fmt.Errorf("quality must be 480 or 720")
		}
	case "lyrics":
		switch language {
		case "en":
			storageKind, value = "lyricsEn", item.LyricsEnObjectKey
		case "zh":
			storageKind, value = "lyricsZh", item.LyricsZhObjectKey
		case "bilingual":
			storageKind, value = "lyricsBilingual", item.LyricsBiObjectKey
		default:
			return "", "", fmt.Errorf("language must be en, zh or bilingual")
		}
	case "poster":
		value = item.PosterObjectKey
	default:
		return "", "", fmt.Errorf("unsupported media kind")
	}
	if value == nil {
		return storageKind, "", nil
	}
	return storageKind, *value, nil
}

func parseIDs(value string) ([]int64, error) {
	if strings.TrimSpace(value) == "" {
		return nil, nil
	}
	parts := strings.Split(value, ",")
	if len(parts) > 500 {
		return nil, fmt.Errorf("ids supports at most 500 values")
	}
	result := make([]int64, 0, len(parts))
	seen := make(map[int64]struct{}, len(parts))
	for _, part := range parts {
		id, err := strconv.ParseInt(strings.TrimSpace(part), 10, 64)
		if err != nil || id < 1 {
			return nil, fmt.Errorf("invalid media id in ids")
		}
		if _, exists := seen[id]; exists {
			continue
		}
		seen[id] = struct{}{}
		result = append(result, id)
	}
	return result, nil
}

func intQuery(r *http.Request, name string, fallback int) int {
	value, err := strconv.Atoi(r.URL.Query().Get(name))
	if err != nil || value < 1 {
		return fallback
	}
	return value
}

func decodeJSON(r *http.Request, target any) error {
	decoder := json.NewDecoder(io.LimitReader(r.Body, 1<<20))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(target); err != nil {
		return fmt.Errorf("invalid JSON: %w", err)
	}
	return nil
}

func writeJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}

func writeError(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, map[string]string{"error": message})
}

func filepathBase(value string) string {
	value = strings.ReplaceAll(value, "\\", "/")
	parts := strings.Split(value, "/")
	return parts[len(parts)-1]
}

func filepathExt(value string) string {
	base := filepathBase(value)
	if index := strings.LastIndex(base, "."); index >= 0 {
		return base[index:]
	}
	return ""
}

func sanitizeFilename(value string) string {
	var builder strings.Builder
	for _, r := range value {
		if r >= 'a' && r <= 'z' || r >= 'A' && r <= 'Z' || r >= '0' && r <= '9' || r == '-' || r == '_' {
			builder.WriteRune(r)
		} else {
			builder.WriteByte('_')
		}
	}
	if builder.Len() == 0 {
		return "poster"
	}
	return builder.String()
}
