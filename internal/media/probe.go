package media

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
)

type ProbeResult struct {
	ResourcePath string `json:"resourcePath"`
	FormatName   string `json:"formatName"`
	DurationMS   int64  `json:"durationMs"`
	AudioCodec   string `json:"audioCodec,omitempty"`
	VideoCodec   string `json:"videoCodec,omitempty"`
	Width        int    `json:"width,omitempty"`
	Height       int    `json:"height,omitempty"`
	SampleRate   int    `json:"sampleRate,omitempty"`
	Channels     int    `json:"channels,omitempty"`
	PosterPath   string `json:"posterPath,omitempty"`
}

type ffprobeOutput struct {
	Format struct {
		Duration   string `json:"duration"`
		FormatName string `json:"format_name"`
	} `json:"format"`
	Streams []struct {
		CodecType  string `json:"codec_type"`
		CodecName  string `json:"codec_name"`
		Width      int    `json:"width"`
		Height     int    `json:"height"`
		SampleRate string `json:"sample_rate"`
		Channels   int    `json:"channels"`
	} `json:"streams"`
}

type Prober struct {
	root          string
	generatedRoot string
	ffprobe       string
	ffmpeg        string
}

func NewProber(root, generatedRoot, ffprobe, ffmpeg string) (*Prober, error) {
	absRoot, err := filepath.Abs(root)
	if err != nil {
		return nil, err
	}
	absGenerated, err := filepath.Abs(generatedRoot)
	if err != nil {
		return nil, err
	}
	if err := os.MkdirAll(absGenerated, 0o755); err != nil {
		return nil, err
	}
	return &Prober{root: absRoot, generatedRoot: absGenerated, ffprobe: ffprobe, ffmpeg: ffmpeg}, nil
}

func (p *Prober) Root() string { return p.root }

func (p *Prober) GeneratedRoot() string { return p.generatedRoot }

func (p *Prober) ResolveResource(relative string) (string, error) {
	return resolveWithin(p.root, relative)
}

func (p *Prober) ResolveGenerated(relative string) (string, error) {
	return resolveWithin(p.generatedRoot, relative)
}

func (p *Prober) Probe(ctx context.Context, relative string) (ProbeResult, error) {
	absPath, err := p.ResolveResource(relative)
	if err != nil {
		return ProbeResult{}, err
	}
	info, err := os.Stat(absPath)
	if err != nil {
		return ProbeResult{}, err
	}
	if !info.Mode().IsRegular() {
		return ProbeResult{}, fmt.Errorf("resource is not a regular file")
	}
	cmd := exec.CommandContext(ctx, p.ffprobe,
		"-v", "error",
		"-show_entries", "format=duration,format_name:stream=codec_type,codec_name,width,height,sample_rate,channels",
		"-of", "json", absPath,
	)
	out, err := cmd.Output()
	if err != nil {
		return ProbeResult{}, commandError("ffprobe", err)
	}
	var raw ffprobeOutput
	if err := json.Unmarshal(out, &raw); err != nil {
		return ProbeResult{}, fmt.Errorf("decode ffprobe output: %w", err)
	}
	result := ProbeResult{ResourcePath: filepath.ToSlash(relative), FormatName: raw.Format.FormatName}
	if seconds, err := strconv.ParseFloat(raw.Format.Duration, 64); err == nil {
		result.DurationMS = int64(seconds*1000 + 0.5)
	}
	for _, stream := range raw.Streams {
		switch stream.CodecType {
		case "audio":
			if result.AudioCodec == "" {
				result.AudioCodec = stream.CodecName
				result.Channels = stream.Channels
				result.SampleRate, _ = strconv.Atoi(stream.SampleRate)
			}
		case "video":
			if result.VideoCodec == "" {
				result.VideoCodec = stream.CodecName
				result.Width = stream.Width
				result.Height = stream.Height
			}
		}
	}
	return result, nil
}

func (p *Prober) ExtractFirstFrame(ctx context.Context, relative, posterKey string) (string, error) {
	input, err := p.ResolveResource(relative)
	if err != nil {
		return "", err
	}
	output, err := p.ResolveGenerated(posterKey)
	if err != nil {
		return "", err
	}
	if err := os.MkdirAll(filepath.Dir(output), 0o755); err != nil {
		return "", err
	}
	cmd := exec.CommandContext(ctx, p.ffmpeg,
		"-hide_banner", "-loglevel", "error", "-y",
		"-i", input, "-map", "0:v:0", "-frames:v", "1", "-q:v", "2", output,
	)
	if out, err := cmd.CombinedOutput(); err != nil {
		message := strings.TrimSpace(string(out))
		if message == "" {
			return "", commandError("ffmpeg", err)
		}
		return "", fmt.Errorf("ffmpeg: %s", message)
	}
	return filepath.ToSlash(posterKey), nil
}

func resolveWithin(root, relative string) (string, error) {
	if strings.TrimSpace(relative) == "" {
		relative = "."
	}
	if filepath.IsAbs(relative) {
		return "", fmt.Errorf("resourcePath must be relative to configured root")
	}
	clean := filepath.Clean(relative)
	if clean == ".." || strings.HasPrefix(clean, ".."+string(filepath.Separator)) {
		return "", fmt.Errorf("resourcePath escapes configured root")
	}
	joined := filepath.Join(root, clean)
	resolved, err := filepath.EvalSymlinks(joined)
	if err == nil {
		joined = resolved
	}
	absJoined, err := filepath.Abs(joined)
	if err != nil {
		return "", err
	}
	rel, err := filepath.Rel(root, absJoined)
	if err != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return "", fmt.Errorf("resourcePath escapes configured root")
	}
	return absJoined, nil
}

func commandError(name string, err error) error {
	var exitErr *exec.ExitError
	if strings.Contains(err.Error(), "executable file not found") {
		return fmt.Errorf("%s executable not found: %w", name, err)
	}
	if errors.As(err, &exitErr) {
		message := strings.TrimSpace(string(exitErr.Stderr))
		if message != "" {
			return fmt.Errorf("run %s: %s", name, message)
		}
		return fmt.Errorf("run %s: %w", name, exitErr)
	}
	if err != nil {
		return fmt.Errorf("run %s: %w", name, err)
	}
	return nil
}
