package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"math"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/fernandovmedina/netflix-clone/microservices/shared/renditions"
)

type probeResult struct {
	Width, Height int
	FPS, Duration float64
	HasAudio      bool
}
type probeJSON struct {
	Streams []struct {
		CodecType    string `json:"codec_type"`
		Width        int    `json:"width"`
		Height       int    `json:"height"`
		AvgFrameRate string `json:"avg_frame_rate"`
		RFrameRate   string `json:"r_frame_rate"`
	} `json:"streams"`
	Format struct {
		Duration string `json:"duration"`
	} `json:"format"`
}

func probe(ctx context.Context, path string) (probeResult, error) {
	cmd := exec.CommandContext(ctx, "ffprobe", "-v", "error", "-print_format", "json", "-show_streams", "-show_format", path)
	raw, err := cmd.Output()
	if err != nil {
		return probeResult{}, fmt.Errorf("ffprobe: %w", err)
	}
	var data probeJSON
	if err = json.Unmarshal(raw, &data); err != nil {
		return probeResult{}, err
	}
	var out probeResult
	for _, s := range data.Streams {
		if s.CodecType == "video" && out.Width == 0 {
			out.Width, out.Height = s.Width, s.Height
			out.FPS = parseRate(s.AvgFrameRate)
			if out.FPS == 0 {
				out.FPS = parseRate(s.RFrameRate)
			}
		}
		if s.CodecType == "audio" {
			out.HasAudio = true
		}
	}
	out.Duration, _ = strconv.ParseFloat(data.Format.Duration, 64)
	if out.Width < 1 || out.Height < 1 || out.FPS <= 0 {
		return out, fmt.Errorf("source has no usable video stream")
	}
	return out, nil
}
func parseRate(value string) float64 {
	parts := strings.Split(value, "/")
	if len(parts) == 2 {
		a, _ := strconv.ParseFloat(parts[0], 64)
		b, _ := strconv.ParseFloat(parts[1], 64)
		if b != 0 {
			return a / b
		}
	}
	v, _ := strconv.ParseFloat(value, 64)
	return v
}

type profile struct {
	Height                         int
	Video, MaxRate, BufSize, Audio int
}

var profiles = map[int]profile{144: {144, 200, 214, 400, 64}, 240: {240, 400, 428, 800, 64}, 360: {360, 800, 856, 1600, 96}, 480: {480, 1400, 1498, 2800, 128}, 720: {720, 2800, 2996, 5600, 128}, 1080: {1080, 5000, 5350, 10000, 192}, 1440: {1440, 8000, 8560, 16000, 192}}

func profileFor(height int) profile {
	if p, ok := profiles[height]; ok {
		return p
	}
	return profile{Height: height, Video: 200, MaxRate: 214, BufSize: 400, Audio: 64}
}
func evenWidth(sourceWidth, sourceHeight, targetHeight int) int {
	if sourceWidth < 1 || sourceHeight < 1 || targetHeight < 1 {
		return 0
	}
	width := int(math.Round((float64(sourceWidth)*float64(targetHeight)/float64(sourceHeight))/2) * 2)
	if width < 2 {
		return 2
	}
	return width
}

func (w *worker) process(ctx context.Context, j job) error {
	source := w.store.Path(j.SourcePath)
	if source == "" {
		return fmt.Errorf("invalid source path")
	}
	meta, err := probe(ctx, source)
	if err != nil {
		return err
	}
	heights := renditions.Heights(meta.Height)
	if len(heights) == 0 {
		return fmt.Errorf("no renditions selected")
	}
	tmp := filepath.Join(w.root, "hls", ".tmp-"+j.ID.String())
	_ = os.RemoveAll(tmp)
	if err = os.MkdirAll(tmp, 0o755); err != nil {
		return err
	}
	success := false
	defer func() {
		if !success {
			_ = os.RemoveAll(tmp)
		}
	}()
	done := make(chan struct{})
	go w.heartbeat(ctx, j, done)
	defer close(done)
	variants := make([]variant, 0, len(heights))
	qualities := make([]string, 0, len(heights))
	for _, height := range heights {
		p := profileFor(height)
		quality := fmt.Sprintf("%dp", height)
		dir := filepath.Join(tmp, quality)
		if err = os.MkdirAll(dir, 0o755); err != nil {
			return err
		}
		width := evenWidth(meta.Width, meta.Height, height)
		args := ffmpegArgs(source, dir, p, meta.FPS, meta.HasAudio)
		cmd := exec.CommandContext(ctx, "ffmpeg", args...)
		var stderr bytes.Buffer
		cmd.Stderr = &stderr
		if err = cmd.Run(); err != nil {
			return fmt.Errorf("ffmpeg %s: %w: %s", quality, err, truncate(stderr.String(), 4000))
		}
		variants = append(variants, variant{Quality: quality, Width: width, Height: height, Profile: p, Audio: meta.HasAudio})
		qualities = append(qualities, quality)
	}
	master := masterPlaylist(variants)
	if err = os.WriteFile(filepath.Join(tmp, "master.m3u8"), []byte(master), 0o644); err != nil {
		return err
	}
	destination := filepath.Join(w.root, "hls", j.AssetID.String())
	if _, statErr := os.Stat(destination); statErr == nil {
		archive := destination + ".superseded-" + time.Now().UTC().Format("20060102T150405.000000000")
		if err = os.Rename(destination, archive); err != nil {
			return err
		}
	}
	if err = os.Rename(tmp, destination); err != nil {
		return err
	}
	success = true
	size, err := directorySize(destination)
	if err != nil {
		return err
	}
	return w.complete(ctx, j, meta, qualities, filepath.ToSlash(filepath.Join("hls", j.AssetID.String(), "master.m3u8")), size)
}
func ffmpegArgs(source, dir string, p profile, fps float64, hasAudio bool) []string {
	gop := strconv.Itoa(maxInt(1, int(math.Round(2*fps))))
	args := []string{"-hide_banner", "-y", "-i", source, "-map", "0:v:0", "-vf", fmt.Sprintf("scale=-2:%d", p.Height), "-c:v", "libx264", "-pix_fmt", "yuv420p", "-profile:v", "main", "-preset", "veryfast", "-b:v", fmt.Sprintf("%dk", p.Video), "-maxrate", fmt.Sprintf("%dk", p.MaxRate), "-bufsize", fmt.Sprintf("%dk", p.BufSize), "-g", gop, "-keyint_min", gop, "-sc_threshold", "0", "-force_key_frames", "expr:gte(t,n_forced*6)"}
	if hasAudio {
		args = append(args, "-map", "0:a:0", "-c:a", "aac", "-b:a", fmt.Sprintf("%dk", p.Audio))
	} else {
		args = append(args, "-an")
	}
	args = append(args, "-hls_time", "6", "-hls_playlist_type", "vod", "-hls_segment_type", "mpegts", "-hls_segment_filename", filepath.Join(dir, "seg_%05d.ts"), filepath.Join(dir, "playlist.m3u8"))
	return args
}

type variant struct {
	Quality       string
	Width, Height int
	Profile       profile
	Audio         bool
}

func masterPlaylist(vs []variant) string {
	var b strings.Builder
	b.WriteString("#EXTM3U\n#EXT-X-VERSION:3\n")
	for _, v := range vs {
		average := v.Profile.Video * 1000
		peak := v.Profile.MaxRate * 1000
		codecs := "avc1.4d401f"
		if v.Audio {
			average += v.Profile.Audio * 1000
			peak += v.Profile.Audio * 1000
			codecs += ",mp4a.40.2"
		}
		fmt.Fprintf(&b, "#EXT-X-STREAM-INF:BANDWIDTH=%d,AVERAGE-BANDWIDTH=%d,RESOLUTION=%dx%d,CODECS=\"%s\"\n%s/playlist.m3u8\n", peak, average, v.Width, v.Height, codecs, v.Quality)
	}
	return b.String()
}
func directorySize(root string) (int64, error) {
	var total int64
	err := filepath.Walk(root, func(_ string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() {
			total += info.Size()
		}
		return nil
	})
	return total, err
}
func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}
