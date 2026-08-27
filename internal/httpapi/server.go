package httpapi

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
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
	store   *store.MySQL
	prober  *media.Prober
	scanner *media.Scanner
	assets  *storage.Assets
}

func New(store *store.MySQL, prober *media.Prober, scanner *media.Scanner, assets *storage.Assets) *API {
	return &API{store: store, prober: prober, scanner: scanner, assets: assets}
}

func (a *API) Register(server *rest.Server) {
	server.AddRoutes([]rest.Route{
		{Method: http.MethodGet, Path: "/api/v1/healthz", Handler: a.health},
		{Method: http.MethodGet, Path: "/api/v1/media", Handler: a.listMedia},
		{Method: http.MethodGet, Path: "/api/v1/media/:id", Handler: a.getMedia},
		{Method: http.MethodGet, Path: "/api/v1/media/:id/content", Handler: a.mediaContent},
		{Method: http.MethodPost, Path: "/api/v1/admin/probe", Handler: a.probeResource},
	})
	server.AddRoute(
		rest.Route{Method: http.MethodPost, Path: "/api/v1/admin/scan", Handler: a.scanResources},
		rest.WithTimeout(10*time.Minute),
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
	items, total, err := a.store.List(r.Context(), r.URL.Query().Get("q"), page, pageSize)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	for i := range items {
		attachURLs(&items[i])
	}
	writeJSON(w, http.StatusOK, map[string]any{"items": items, "total": total, "page": page, "pageSize": pageSize})
}

func (a *API) getMedia(w http.ResponseWriter, r *http.Request) {
	item, ok := a.loadMedia(w, r)
	if !ok {
		return
	}
	attachURLs(&item)
	writeJSON(w, http.StatusOK, item)
}

func (a *API) mediaContent(w http.ResponseWriter, r *http.Request) {
	item, ok := a.loadMedia(w, r)
	if !ok {
		return
	}
	kind := r.URL.Query().Get("kind")
	key := objectKey(item, kind)
	if key == "" {
		writeError(w, http.StatusNotFound, "requested media type is unavailable")
		return
	}
	if a.assets.IsLocal() {
		localPath, err := a.assets.LocalPath(kind, key)
		if err != nil {
			writeError(w, http.StatusBadRequest, err.Error())
			return
		}
		http.ServeFile(w, r, localPath)
		return
	}
	url, err := a.assets.PresignedURL(r.Context(), key)
	if err != nil {
		writeError(w, http.StatusBadGateway, err.Error())
		return
	}
	http.Redirect(w, r, url.String(), http.StatusTemporaryRedirect)
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

func attachURLs(item *model.MediaItem) {
	base := fmt.Sprintf("/api/v1/media/%d/content?kind=", item.ID)
	if item.HasAudio {
		item.AudioURL = base + "audio"
	}
	if item.HasVideo {
		item.VideoURL = base + "video"
	}
	if item.HasLyric {
		item.LyricURL = base + "lyric"
	}
	if item.HasPoster {
		item.PosterURL = base + "poster"
	}
}

func objectKey(item model.MediaItem, kind string) string {
	var value *string
	switch kind {
	case "audio":
		value = item.AudioObjectKey
	case "video":
		value = item.VideoObjectKey
	case "lyric":
		value = item.LyricObjectKey
	case "poster":
		value = item.PosterObjectKey
	default:
		return ""
	}
	if value == nil {
		return ""
	}
	return *value
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
