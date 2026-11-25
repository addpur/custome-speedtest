package service

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"sync"
	"sync/atomic"
	"time"

	restdomain "github.com/addpur/custome-speedtest/rest/domain"
)

func StartStress(ctx context.Context, req *restdomain.RunRequest) error {
	var wg sync.WaitGroup
	var totalBytes int64
	start := time.Now()

	servers := []string{}
	if len(req.Servers) > 0 {
		servers = req.Servers
	} else {
		servers = []string{"http://localhost:3000"}
	}

	for _, server := range servers {
		for i := 0; i < req.Parallel; i++ {
			wg.Add(1)
			go func(s string) {
				defer wg.Done()
				runParallelDownload(ctx, s, req.Duration, &totalBytes)
			}(server)

			wg.Add(1)
			go func(s string) {
				defer wg.Done()
				runParallelUpload(ctx, s, req.Duration, &totalBytes)
			}(server)
		}
	}

	wg.Wait()
	elapsed := time.Since(start)
	mbps := float64(atomic.LoadInt64(&totalBytes)) * 8 / (elapsed.Seconds() * 1_000_000)
	if req.Overhead > 0 {
		mbps *= req.Overhead
	}

	fmt.Printf("[STRESS] Total: %.2f Mbps | Duration: %v | Parallel: %d\n", mbps, elapsed, req.Parallel)
	return nil
}

func runParallelDownload(ctx context.Context, server string, duration int, totalBytes *int64) {
	client := &http.Client{Timeout: 30 * time.Second}
	url := server + "/api/v1/download?duration=" + fmt.Sprintf("%d", duration)

	req, _ := http.NewRequestWithContext(ctx, "GET", url, nil)
	resp, err := client.Do(req)
	if err != nil {
		return
	}
	defer resp.Body.Close()

	buf := make([]byte, 32*1024)
	for {
		n, err := resp.Body.Read(buf)
		if n > 0 {
			atomic.AddInt64(totalBytes, int64(n))
		}
		if err == io.EOF {
			break
		}
		if err != nil {
			return
		}
	}
}

func runParallelUpload(ctx context.Context, server string, duration int, totalBytes *int64) {
	client := &http.Client{Timeout: 30 * time.Second}
	url := server + "/api/v1/upload"

	pr, pw := io.Pipe()
	go func() {
		buf := make([]byte, 4*1024*1024)
		for i := range buf {
			buf[i] = byte(i)
		}
		pw.Write(buf)
		pw.Close()
	}()

	req, _ := http.NewRequestWithContext(ctx, "POST", url, pr)

	resp, err := client.Do(req)
	if err != nil {
		pw.CloseWithError(err)
		return
	}
	defer resp.Body.Close()

	buf := make([]byte, 32*1024)
	for {
		n, err := resp.Body.Read(buf)
		if n > 0 {
			atomic.AddInt64(totalBytes, int64(n))
		}
		if err == io.EOF {
			break
		}
		if err != nil {
			return
		}
	}
}
