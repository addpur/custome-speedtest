package cli

import (
	"bytes"
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"math"
	"net/http"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

type CLIConfig struct {
	Server   string  `json:"server"`
	Duration int     `json:"duration"`
	Parallel int     `json:"parallel"`
	Overhead float64 `json:"overhead"`
	JSONOut  bool    `json:"-"`
}

type CLIResult struct {
	Server        string    `json:"server"`
	PingMs        float64   `json:"ping_ms"`
	JitterMs      float64   `json:"jitter_ms"`
	DownloadMbps  float64   `json:"download_mbps"`
	DownloadBytes int64     `json:"download_bytes"`
	UploadMbps    float64   `json:"upload_mbps"`
	UploadBytes   int64     `json:"upload_bytes"`
	Timestamp     time.Time `json:"timestamp"`
}

// Run executes the CLI speedtest client against a remote or local speedtest server
func Run(args []string) error {
	return RunWithTransport(args, nil)
}

// RunWithTransport allows running CLI with a custom HTTP RoundTripper (useful for testing)
func RunWithTransport(args []string, rt http.RoundTripper) error {
	fs := flag.NewFlagSet("speedtest-cli", flag.ContinueOnError)

	var (
		server   string
		duration int
		parallel int
		overhead float64
		jsonOut  bool
	)

	fs.StringVar(&server, "s", "http://localhost:3000", "Server URL (e.g. http://192.168.1.10:3000)")
	fs.StringVar(&server, "server", "http://localhost:3000", "Server URL")
	fs.IntVar(&duration, "d", 10, "Test duration per phase in seconds")
	fs.IntVar(&duration, "duration", 10, "Test duration per phase in seconds")
	fs.IntVar(&parallel, "p", 4, "Number of parallel streams (1-16)")
	fs.IntVar(&parallel, "parallel", 4, "Number of parallel streams")
	fs.Float64Var(&overhead, "overhead", 1.0, "Overhead compensation factor (e.g. 1.05)")
	fs.BoolVar(&jsonOut, "json", false, "Output results in JSON format")

	if err := fs.Parse(args); err != nil {
		return err
	}

	server = strings.TrimRight(server, "/")
	if !strings.HasPrefix(server, "http://") && !strings.HasPrefix(server, "https://") {
		server = "http://" + server
	}

	cfg := CLIConfig{
		Server:   server,
		Duration: duration,
		Parallel: parallel,
		Overhead: overhead,
		JSONOut:  jsonOut,
	}

	if !cfg.JSONOut {
		fmt.Println("======================================================")
		fmt.Println("  ⚡ Custome Speedtest CLI (custome-speedtest)")
		fmt.Println("======================================================")
		fmt.Printf("  Target Server : %s\n", cfg.Server)
		fmt.Printf("  Streams       : %d parallel connections\n", cfg.Parallel)
		fmt.Printf("  Duration      : %ds per phase\n", cfg.Duration)
		fmt.Println("------------------------------------------------------")
	}

	client := &http.Client{
		Timeout: time.Duration(cfg.Duration+10) * time.Second,
	}
	if rt != nil {
		client.Transport = rt
	}

	// 1. Latency & Jitter
	if !cfg.JSONOut {
		fmt.Print("[*] Testing Latency & Jitter (10 samples)... ")
	}
	ping, jitter, err := runPing(client, cfg.Server, 10)
	if err != nil {
		if !cfg.JSONOut {
			fmt.Printf("FAILED (%v)\n", err)
		}
		return fmt.Errorf("ping test failed: %w", err)
	}
	if !cfg.JSONOut {
		fmt.Printf("Done.\n    Latency: %.2f ms | Jitter: %.2f ms\n", ping, jitter)
	}

	// 2. Download Test
	if !cfg.JSONOut {
		fmt.Printf("[*] Testing Download Speed (%d streams, %ds)...\n", cfg.Parallel, cfg.Duration)
	}
	dlMbps, dlBytes, err := runDownload(cfg, rt)
	if err != nil {
		if !cfg.JSONOut {
			fmt.Printf("FAILED (%v)\n", err)
		}
		return fmt.Errorf("download test failed: %w", err)
	}
	if !cfg.JSONOut {
		fmt.Printf("    Download: %.2f Mbps (%s transferred)\n", dlMbps, FormatBytes(dlBytes))
	}

	// 3. Upload Test
	if !cfg.JSONOut {
		fmt.Printf("[*] Testing Upload Speed (%d streams, %ds)...\n", cfg.Parallel, cfg.Duration)
	}
	ulMbps, ulBytes, err := runUpload(cfg, rt)
	if err != nil {
		if !cfg.JSONOut {
			fmt.Printf("FAILED (%v)\n", err)
		}
		return fmt.Errorf("upload test failed: %w", err)
	}
	if !cfg.JSONOut {
		fmt.Printf("    Upload  : %.2f Mbps (%s transferred)\n", ulMbps, FormatBytes(ulBytes))
	}

	res := CLIResult{
		Server:        cfg.Server,
		PingMs:        math.Round(ping*100) / 100,
		JitterMs:      math.Round(jitter*100) / 100,
		DownloadMbps:  math.Round(dlMbps*100) / 100,
		DownloadBytes: dlBytes,
		UploadMbps:    math.Round(ulMbps*100) / 100,
		UploadBytes:   ulBytes,
		Timestamp:     time.Now().UTC(),
	}

	if cfg.JSONOut {
		b, _ := json.MarshalIndent(res, "", "  ")
		fmt.Println(string(b))
	} else {
		fmt.Println("------------------------------------------------------")
		fmt.Println("  🏆 TEST SUMMARY:")
		fmt.Printf("    Latency  : %.2f ms\n", res.PingMs)
		fmt.Printf("    Jitter   : %.2f ms\n", res.JitterMs)
		fmt.Printf("    Download : %.2f Mbps (%s)\n", res.DownloadMbps, FormatBytes(res.DownloadBytes))
		fmt.Printf("    Upload   : %.2f Mbps (%s)\n", res.UploadMbps, FormatBytes(res.UploadBytes))
		fmt.Println("======================================================")
	}

	return nil
}

func runPing(client *http.Client, server string, count int) (float64, float64, error) {
	url := fmt.Sprintf("%s/api/v1/ping", server)
	var latencies []float64

	for i := 0; i < count; i++ {
		start := time.Now()
		req, err := http.NewRequest("GET", fmt.Sprintf("%s?t=%d_%d", url, time.Now().UnixNano(), i), nil)
		if err != nil {
			return 0, 0, err
		}
		req.Header.Set("Cache-Control", "no-cache")

		resp, err := client.Do(req)
		if err != nil {
			return 0, 0, err
		}
		if resp.StatusCode != http.StatusOK {
			resp.Body.Close()
			return 0, 0, fmt.Errorf("server returned HTTP %d", resp.StatusCode)
		}
		io.Copy(io.Discard, resp.Body)
		resp.Body.Close()

		elapsedMs := float64(time.Since(start).Microseconds()) / 1000.0
		latencies = append(latencies, elapsedMs)
		time.Sleep(50 * time.Millisecond)
	}

	if len(latencies) == 0 {
		return 0, 0, fmt.Errorf("no ping samples collected")
	}

	// Calculate minimum latency
	minLatency := latencies[0]
	for _, l := range latencies {
		if l < minLatency {
			minLatency = l
		}
	}

	// Calculate Jitter (RFC 1889)
	var jitterSum float64
	for i := 1; i < len(latencies); i++ {
		jitterSum += math.Abs(latencies[i] - latencies[i-1])
	}
	jitter := jitterSum / float64(len(latencies)-1)

	return minLatency, jitter, nil
}

// FormatBytes formats byte counts dynamically into B, KB, MB, GB, TB, or PB
func FormatBytes(b int64) string {
	const (
		KB = 1024
		MB = 1024 * KB
		GB = 1024 * MB
		TB = 1024 * GB
		PB = 1024 * TB
	)

	bf := float64(b)
	switch {
	case b >= PB:
		return fmt.Sprintf("%.2f PB", bf/PB)
	case b >= TB:
		return fmt.Sprintf("%.2f TB", bf/TB)
	case b >= GB:
		return fmt.Sprintf("%.2f GB", bf/GB)
	case b >= MB:
		return fmt.Sprintf("%.2f MB", bf/MB)
	case b >= KB:
		return fmt.Sprintf("%.2f KB", bf/KB)
	default:
		return fmt.Sprintf("%d B", b)
	}
}

func runDownload(cfg CLIConfig, rt http.RoundTripper) (float64, int64, error) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(cfg.Duration)*time.Second)
	defer cancel()

	var totalBytes int64
	var wg sync.WaitGroup
	start := time.Now()

	client := &http.Client{
		Transport: &http.Transport{
			MaxIdleConns:        cfg.Parallel * 8,
			MaxIdleConnsPerHost: cfg.Parallel * 8,
			DisableCompression:  true,
			ReadBufferSize:      512 * 1024,
			WriteBufferSize:     128 * 1024,
		},
	}
	if rt != nil {
		client.Transport = rt
	}

	for i := 0; i < cfg.Parallel; i++ {
		wg.Add(1)
		go func(streamID int) {
			defer wg.Done()
			var streamBytes int64
			copyBuf := make([]byte, 512*1024)

			for {
				if ctx.Err() != nil {
					break
				}
				url := fmt.Sprintf("%s/api/v1/download?size=33554432&stream=%d", cfg.Server, streamID)
				req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
				if err != nil {
					break
				}
				req.Header.Set("Cache-Control", "no-cache")

				resp, err := client.Do(req)
				if err != nil {
					break
				}
				n, _ := io.CopyBuffer(io.Discard, resp.Body, copyBuf)
				resp.Body.Close()
				streamBytes += n
			}
			atomic.AddInt64(&totalBytes, streamBytes)
		}(i)
	}

	wg.Wait()
	elapsed := time.Since(start)

	bytesTransferred := atomic.LoadInt64(&totalBytes)
	mbps := (float64(bytesTransferred) * 8) / (elapsed.Seconds() * 1_000_000)
	if cfg.Overhead > 0 {
		mbps *= cfg.Overhead
	}

	return mbps, bytesTransferred, nil
}

func runUpload(cfg CLIConfig, rt http.RoundTripper) (float64, int64, error) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(cfg.Duration)*time.Second)
	defer cancel()

	var totalBytes int64
	var wg sync.WaitGroup
	start := time.Now()

	// Incompressible chunk of 4MB in RAM (reduces request frequency & CPU usage)
	chunkSize := 4 * 1024 * 1024
	chunk := make([]byte, chunkSize)
	for i := range chunk {
		chunk[i] = byte(i * 31)
	}

	client := &http.Client{
		Transport: &http.Transport{
			MaxIdleConns:        cfg.Parallel * 4,
			MaxIdleConnsPerHost: cfg.Parallel * 4,
			DisableCompression:  true,
			WriteBufferSize:     256 * 1024,
			ReadBufferSize:      64 * 1024,
		},
	}
	if rt != nil {
		client.Transport = rt
	}

	for i := 0; i < cfg.Parallel; i++ {
		wg.Add(1)
		go func(streamID int) {
			defer wg.Done()
			var streamBytes int64

			for {
				if ctx.Err() != nil {
					break
				}
				url := fmt.Sprintf("%s/api/v1/upload?stream=%d", cfg.Server, streamID)
				req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(chunk))
				if err != nil {
					break
				}
				req.Header.Set("Content-Type", "application/octet-stream")
				req.Header.Set("Cache-Control", "no-cache")

				resp, err := client.Do(req)
				if err != nil {
					break
				}
				io.Copy(io.Discard, resp.Body)
				resp.Body.Close()

				streamBytes += int64(chunkSize)
			}
			atomic.AddInt64(&totalBytes, streamBytes)
		}(i)
	}

	wg.Wait()
	elapsed := time.Since(start)

	bytesTransferred := atomic.LoadInt64(&totalBytes)
	mbps := (float64(bytesTransferred) * 8) / (elapsed.Seconds() * 1_000_000)
	if cfg.Overhead > 0 {
		mbps *= cfg.Overhead
	}

	return mbps, bytesTransferred, nil
}
