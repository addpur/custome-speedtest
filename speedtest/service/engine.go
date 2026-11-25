package service

import (
	"bytes"
	"io"
	"time"

	"github.com/addpur/custome-speedtest/common"
	"github.com/addpur/custome-speedtest/speedtest/domain"
)

type Engine struct {
	buffer []byte
}

func NewEngine(cfg domain.Config) *Engine {
	bufSize := cfg.BufferSize
	if bufSize <= 0 {
		bufSize = common.DefaultBufferSize
	}
	return &Engine{
		buffer: GenerateIncompressibleBuffer(bufSize),
	}
}

func (e *Engine) Buffer() []byte {
	return e.buffer
}

func (e *Engine) DownloadStream(w io.Writer, duration time.Duration) (int64, error) {
	if w == nil {
		w = io.Discard
	}
	var total int64
	deadline := time.Now().Add(duration)
	for time.Now().Before(deadline) {
		n, err := e.bufferWrite(w)
		if err != nil {
			return total, err
		}
		total += int64(n)
	}
	return total, nil
}

func (e *Engine) bufferWrite(w io.Writer) (int64, error) {
	n, err := w.Write(e.buffer)
	return int64(n), err
}

func (e *Engine) UploadDrain(r io.Reader, duration time.Duration) (int64, error) {
	if r == nil {
		r = bytes.NewReader(e.buffer)
	}
	var total int64
	buf := make([]byte, 32*1024)
	deadline := time.Now().Add(duration)
	for time.Now().Before(deadline) {
		n, err := r.Read(buf)
		if n > 0 {
			total += int64(n)
		}
		if err == io.EOF {
			break
		}
		if err != nil {
			return total, err
		}
	}
	return total, nil
}

func (e *Engine) CalculateMbps(bytes int64, duration time.Duration, overhead float64) float64 {
	if duration <= 0 {
		return 0
	}
	mbps := (float64(bytes) * 8) / (duration.Seconds() * 1_000_000)
	if overhead > 0 {
		mbps *= overhead
	}
	return mbps
}

func (e *Engine) RunDownload(w io.Writer, duration time.Duration) (int64, time.Duration, error) {
	if w == nil {
		w = io.Discard
	}
	start := time.Now()
	bytes, err := e.DownloadStream(w, duration)
	elapsed := time.Since(start)
	return bytes, elapsed, err
}

type infiniteBufferReader struct {
	data []byte
	pos  int
}

func (r *infiniteBufferReader) Read(p []byte) (n int, err error) {
	if len(r.data) == 0 {
		return 0, io.EOF
	}
	n = copy(p, r.data[r.pos:])
	r.pos = (r.pos + n) % len(r.data)
	return n, nil
}

func (e *Engine) RunUpload(r io.Reader, duration time.Duration) (int64, time.Duration, error) {
	if r == nil {
		r = &infiniteBufferReader{data: e.buffer}
	}
	start := time.Now()
	bytes, err := e.UploadDrain(r, duration)
	elapsed := time.Since(start)
	return bytes, elapsed, err
}

func (e *Engine) CreateResult(testType domain.TestType, server string, bytes int64, duration time.Duration, parallel int, overhead float64) domain.Result {
	mbps := e.CalculateMbps(bytes, duration, overhead)
	return domain.Result{
		ID:              time.Now().Format("20060102150405"),
		Type:            testType,
		Server:          server,
		Bytes:           bytes,
		Duration:        duration,
		Mbps:            mbps,
		Parallel:        parallel,
		OverheadApplied: overhead > 0,
		Timestamp:       time.Now(),
	}
}
