package media

import (
	"context"
	"fmt"
	"path"
	"regexp"
	"sort"
	"strconv"
	"strings"

	"github.com/zxxf18/kids-planet-be/internal/model"
)

var leadingNumber = regexp.MustCompile(`^\s*(\d+)`)

type MediaWriter interface {
	Upsert(ctx context.Context, item model.UpsertMedia) error
}

type ResourceEntry struct {
	Kind string
	Key  string
	Name string
}

type ResourceSource interface {
	ListResources(ctx context.Context, subPath string) ([]ResourceEntry, error)
	ProbeResource(ctx context.Context, kind, key string) (ProbeResult, error)
}

type Scanner struct {
	source ResourceSource
	writer MediaWriter
}

type ScanResult struct {
	Discovered int         `json:"discovered"`
	Saved      int         `json:"saved"`
	Ready      int         `json:"ready"`
	Partial    int         `json:"partial"`
	Invalid    int         `json:"invalid"`
	Issues     []ScanIssue `json:"issues"`
}

type ScanIssue struct {
	SourceCode string `json:"sourceCode,omitempty"`
	Path       string `json:"path,omitempty"`
	Message    string `json:"message"`
}

type candidate struct {
	no              int
	code            string
	audio           string
	video480        string
	video720        string
	lyricsEn        string
	lyricsZh        string
	lyricsBilingual string
	poster          string
	titles          map[string]string
}

type namedImageCandidate struct {
	no    int
	code  string
	kind  string
	path  string
	title string
}

func NewScanner(source ResourceSource, writer MediaWriter) *Scanner {
	return &Scanner{source: source, writer: writer}
}

func (s *Scanner) Scan(ctx context.Context, subPath string) (ScanResult, error) {
	entries, err := s.source.ListResources(ctx, subPath)
	if err != nil {
		return ScanResult{}, err
	}
	items := map[int]*candidate{}
	images := make([]namedImageCandidate, 0)
	result := ScanResult{Issues: make([]ScanIssue, 0)}
	for _, entry := range entries {
		name := entry.Name
		if strings.HasPrefix(name, ".") || strings.HasPrefix(name, "_") {
			continue
		}
		kind := entry.Kind
		if !supportedResourceKind(kind) {
			continue
		}
		if isLyricsKind(kind) {
			ext := strings.ToLower(path.Ext(name))
			if ext != ".lrc" && ext != ".txt" {
				continue
			}
		}
		match := leadingNumber.FindStringSubmatch(name)
		if len(match) != 2 {
			result.Issues = append(result.Issues, ScanIssue{Path: entry.Key, Message: "文件名缺少前导数字编号"})
			continue
		}
		no, parseErr := strconv.Atoi(match[1])
		if parseErr != nil {
			result.Issues = append(result.Issues, ScanIssue{Path: entry.Key, Message: "文件编号无效"})
			continue
		}
		relative := entry.Key
		title := cleanTitle(strings.TrimSuffix(name, path.Ext(name)))
		if isCompanionKind(kind) {
			images = append(images, namedImageCandidate{no: no, code: match[1], kind: kind, path: relative, title: title})
			continue
		}
		item := items[no]
		if item == nil {
			item = &candidate{no: no, code: match[1], titles: map[string]string{}}
			items[no] = item
		}
		item.titles[kind] = title
		switch kind {
		case "audio":
			item.audio = relative
		case "video480":
			item.video480 = relative
		case "video720":
			item.video720 = relative
		}
	}
	for _, image := range images {
		item := matchNamedImage(items, image)
		label := companionLabel(image.kind)
		if item == nil {
			result.Issues = append(result.Issues, ScanIssue{
				SourceCode: image.code,
				Path:       image.path,
				Message:    "没有找到同名的音频或视频，" + label + "已忽略",
			})
			continue
		}
		var current *string
		switch image.kind {
		case "lyricsEn":
			current = &item.lyricsEn
		case "lyricsZh":
			current = &item.lyricsZh
		case "lyricsBilingual":
			current = &item.lyricsBilingual
		case "poster":
			current = &item.poster
		default:
			continue
		}
		if *current != "" {
			result.Issues = append(result.Issues, ScanIssue{
				SourceCode: item.code,
				Path:       image.path,
				Message:    "同一媒体匹配到多张" + label + "，保留第一张",
			})
			continue
		}
		*current = image.path
		item.titles[image.kind] = image.title
	}
	for no, item := range items {
		if hasPlayableMedia(item) {
			continue
		}
		result.Issues = append(result.Issues, ScanIssue{
			SourceCode: item.code,
			Path:       item.lyricsEn,
			Message:    "仅有配套资源，缺少可播放的音频或视频，已忽略",
		})
		delete(items, no)
	}
	result.Discovered = len(items)
	numbers := make([]int, 0, len(items))
	for no := range items {
		numbers = append(numbers, no)
	}
	sort.Ints(numbers)
	for _, no := range numbers {
		item := items[no]
		record, issues := s.inspectCandidate(ctx, item)
		for _, issue := range issues {
			result.Issues = append(result.Issues, ScanIssue{SourceCode: item.code, Message: issue})
		}
		if err := s.writer.Upsert(ctx, record); err != nil {
			result.Issues = append(result.Issues, ScanIssue{SourceCode: item.code, Message: "写入数据库失败: " + err.Error()})
			continue
		}
		result.Saved++
		switch record.ValidationStatus {
		case "ready":
			result.Ready++
		case "invalid":
			result.Invalid++
		default:
			result.Partial++
		}
	}
	return result, nil
}

func hasPlayableMedia(item *candidate) bool {
	return item != nil && (item.audio != "" || item.video480 != "" || item.video720 != "")
}

func matchNamedImage(items map[int]*candidate, image namedImageCandidate) *candidate {
	if isCompanionKind(image.kind) {
		return items[image.no]
	}
	title := normalizeTitle(image.title)
	numbers := make([]int, 0, len(items))
	for no := range items {
		numbers = append(numbers, no)
	}
	sort.Ints(numbers)
	for _, no := range numbers {
		item := items[no]
		for _, kind := range []string{"audio", "video720", "video480"} {
			if title != "" && normalizeTitle(item.titles[kind]) == title {
				return item
			}
		}
	}
	item := items[image.no]
	if item == nil {
		return nil
	}
	for _, kind := range []string{"audio", "video720", "video480"} {
		if titleSimilarity(image.title, item.titles[kind]) >= 0.5 {
			return item
		}
	}
	return nil
}

func normalizeTitle(value string) string {
	value = strings.ToLower(value)
	parts := strings.FieldsFunc(value, func(r rune) bool {
		return (r < 'a' || r > 'z') && (r < '0' || r > '9')
	})
	return strings.Join(parts, " ")
}

func titleSimilarity(left, right string) float64 {
	leftTokens := strings.Fields(normalizeTitle(left))
	rightTokens := strings.Fields(normalizeTitle(right))
	if len(leftTokens) == 0 || len(rightTokens) == 0 {
		return 0
	}
	leftSet := make(map[string]struct{}, len(leftTokens))
	union := make(map[string]struct{}, len(leftTokens)+len(rightTokens))
	for _, token := range leftTokens {
		leftSet[token] = struct{}{}
		union[token] = struct{}{}
	}
	rightSet := make(map[string]struct{}, len(rightTokens))
	for _, token := range rightTokens {
		rightSet[token] = struct{}{}
		union[token] = struct{}{}
	}
	intersection := 0
	for token := range rightSet {
		if _, ok := leftSet[token]; ok {
			intersection++
		}
	}
	return float64(intersection) / float64(len(union))
}

func (s *Scanner) inspectCandidate(ctx context.Context, item *candidate) (model.UpsertMedia, []string) {
	issues := make([]string, 0)
	record := model.UpsertMedia{
		SourceNo: item.no, SourceCode: item.code, Title: chooseTitle(item),
		AudioObjectKey:    optionalString(item.audio),
		Video480ObjectKey: optionalString(item.video480), Video720ObjectKey: optionalString(item.video720),
		LyricsEnObjectKey: optionalString(item.lyricsEn), LyricsZhObjectKey: optionalString(item.lyricsZh),
		LyricsBiObjectKey: optionalString(item.lyricsBilingual),
		PosterObjectKey:   optionalString(item.poster), ValidationStatus: "ready",
	}
	if item.audio == "" || item.video480 == "" || item.video720 == "" || item.lyricsEn == "" || item.lyricsZh == "" || item.lyricsBilingual == "" || item.poster == "" {
		record.ValidationStatus = "partial"
	}
	if item.audio != "" {
		probe, err := s.source.ProbeResource(ctx, "audio", item.audio)
		if err != nil {
			issues = append(issues, "音频校验失败: "+err.Error())
			record.ValidationStatus = "invalid"
		} else {
			record.AudioDurationMS = optionalInt64(probe.DurationMS)
			record.AudioCodec = optionalString(probe.AudioCodec)
		}
	}
	videoKind, videoKey := "video720", item.video720
	if videoKey == "" {
		videoKind, videoKey = "video480", item.video480
	}
	if videoKey != "" {
		probe, err := s.source.ProbeResource(ctx, videoKind, videoKey)
		if err != nil {
			issues = append(issues, "视频校验失败: "+err.Error())
			record.ValidationStatus = "invalid"
		} else {
			record.VideoDurationMS = optionalInt64(probe.DurationMS)
			record.VideoWidth = optionalInt(probe.Width)
			record.VideoHeight = optionalInt(probe.Height)
			record.VideoCodec = optionalString(probe.VideoCodec)
		}
	}
	record.ValidationMessage = strings.Join(issues, "; ")
	return record, issues
}

func cleanTitle(stem string) string {
	stem = leadingNumber.ReplaceAllString(stem, "")
	stem = strings.TrimLeft(stem, " .-_—")
	if stem == "" {
		return "未命名儿歌"
	}
	stem = strings.NewReplacer(
		",s", "'s", ",re", "'re", ",m", "'m", ",t", "'t", ",ve", "'ve", ",ll", "'ll", ",d", "'d",
		"，s", "'s", "，re", "'re", "，m", "'m", "，t", "'t", "？", "?",
	).Replace(stem)
	return stem
}

func chooseTitle(item *candidate) string {
	for _, kind := range []string{"audio", "video720", "video480", "lyricsEn"} {
		if title := strings.TrimSpace(item.titles[kind]); title != "" {
			return title
		}
	}
	return fmt.Sprintf("儿歌 %s", item.code)
}

func supportedResourceKind(kind string) bool {
	switch kind {
	case "audio", "video480", "video720", "lyricsEn", "lyricsZh", "lyricsBilingual", "poster":
		return true
	default:
		return false
	}
}

func isLyricsKind(kind string) bool {
	return kind == "lyricsEn" || kind == "lyricsZh" || kind == "lyricsBilingual"
}

func isCompanionKind(kind string) bool { return isLyricsKind(kind) || kind == "poster" }

func companionLabel(kind string) string {
	switch kind {
	case "lyricsEn":
		return "英文歌词"
	case "lyricsZh":
		return "中文歌词"
	case "lyricsBilingual":
		return "中英文歌词"
	default:
		return "封面图片"
	}
}

func optionalString(value string) *string {
	if value == "" {
		return nil
	}
	return &value
}

func optionalInt64(value int64) *int64 {
	if value == 0 {
		return nil
	}
	return &value
}

func optionalInt(value int) *int {
	if value == 0 {
		return nil
	}
	return &value
}
