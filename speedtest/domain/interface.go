package domain

import "context"

type SpeedtestEngine interface {
    DownloadStream(w interface{}, duration int) (int64, error)
    UploadDrain(r interface{}, duration int) (int64, error)
    CalculateMbps(bytes int64, duration int, overhead float64) float64
    CreateResult(testType TestType, server string, bytes int64, duration int, parallel int, overhead float64) Result
}

type SpeedtestRegistry interface {
    GetServers(keywords []string) []ServerInfo
    AddServer(url string, keywords []string)
    All() []ServerInfo
}

type SpeedtestStore interface {
    Add(r Result)
    Latest(n int) []Result
    All() []Result
}

type Scheduler interface {
    Start(ctx context.Context)
}
