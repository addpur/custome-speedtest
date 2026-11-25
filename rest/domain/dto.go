package domain

import "time"

type PingResponse struct {
	Timestamp int64 `json:"timestamp"`
}

type DownloadRequest struct {
	Duration int `query:"duration"`
	Size     int `query:"size"`
}

type DownloadResponse struct {
	Bytes    int64   `json:"bytes"`
	Mbps     float64 `json:"mbps"`
	Duration float64 `json:"duration"`
}

type UploadRequest struct {
	Duration int `form:"duration"`
}

type UploadResponse struct {
	Bytes    int64   `json:"bytes"`
	Mbps     float64 `json:"mbps"`
	Duration float64 `json:"duration"`
}

type ConfigResponse struct {
	ListenAddr      string  `json:"listen_addr"`
	BufferSize      int     `json:"buffer_size"`
	DefaultDuration int     `json:"default_duration"`
	DefaultParallel int     `json:"default_parallel"`
	DefaultOverhead float64 `json:"default_overhead"`
}

type ServerResponse struct {
	URL      string   `json:"url"`
	Keywords []string `json:"keywords"`
}

type ResultResponse struct {
	ID              string    `json:"id"`
	Type            string    `json:"type"`
	Server          string    `json:"server"`
	Bytes           int64     `json:"bytes"`
	Mbps            float64   `json:"mbps"`
	PingMs          float64   `json:"ping_ms,omitempty"`
	JitterMs        float64   `json:"jitter_ms,omitempty"`
	Parallel        int       `json:"parallel"`
	OverheadApplied bool      `json:"overhead_applied"`
	Timestamp       time.Time `json:"timestamp"`
}

type SaveResultRequest struct {
	Type     string  `json:"type"`
	Server   string  `json:"server"`
	Bytes    int64   `json:"bytes"`
	Mbps     float64 `json:"mbps"`
	PingMs   float64 `json:"ping_ms"`
	JitterMs float64 `json:"jitter_ms"`
	Parallel int     `json:"parallel"`
}

type StressResponse struct {
	DownloadMbps float64 `json:"download_mbps"`
	UploadMbps   float64 `json:"upload_mbps"`
	TotalMbps    float64 `json:"total_mbps"`
	Duration     float64 `json:"duration"`
}

type RunRequest struct {
	Duration  int      `json:"duration"`
	Parallel  int      `json:"parallel"`
	Overhead  float64  `json:"overhead"`
	Servers   []string `json:"servers"`
	Keywords  []string `json:"keywords"`
	AutoAfter int      `json:"auto_after"`
	Stress    bool     `json:"stress"`
}
