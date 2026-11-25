package domain

import "time"

type TestType string

const (
    Download TestType = "download"
    Upload   TestType = "upload"
    Stress   TestType = "stress"
)

type Config struct {
    Duration      time.Duration
    Parallel      int
    Overhead      float64
    Keywords      []string
    Servers       []string
    AutoRun       bool
    AutoAfter     time.Duration
    StressDuration time.Duration
    BufferSize    int
    ListenAddr    string
    Mode          string
}

type Result struct {
    ID             string
    Type           TestType
    Server         string
    Bytes          int64
    Duration       time.Duration
    Mbps           float64
    Parallel       int
    OverheadApplied bool
    Timestamp      time.Time
}

type ServerInfo struct {
    URL        string
    Keywords   []string
    Latency    time.Duration
}

type Report struct {
    DownloadMbps float64
    UploadMbps   float64
    TotalMbps    float64
}
