package service_test

import (
	"bytes"
	"io"
	"testing"
	"time"

	"github.com/addpur/custome-speedtest/speedtest/domain"
	"github.com/addpur/custome-speedtest/speedtest/service"
)

func TestEngineBuffer(t *testing.T) {
	cfg := domain.Config{
		BufferSize: 1024 * 1024,
	}
	engine := service.NewEngine(cfg)
	if len(engine.Buffer()) != 1024*1024 {
		t.Fatalf("Expected buffer size %d, got %d", 1024*1024, len(engine.Buffer()))
	}
}

func TestEngineCalculateMbps(t *testing.T) {
	cfg := domain.Config{BufferSize: 1024}
	engine := service.NewEngine(cfg)

	// 12,500,000 bytes in 1 second = 100,000,000 bits = 100 Mbps
	mbps := engine.CalculateMbps(12_500_000, 1*time.Second, 1.0)
	if mbps != 100.0 {
		t.Errorf("Expected 100.0 Mbps, got %f", mbps)
	}

	// With overhead 1.05
	mbpsOverhead := engine.CalculateMbps(12_500_000, 1*time.Second, 1.05)
	if mbpsOverhead != 105.0 {
		t.Errorf("Expected 105.0 Mbps with overhead, got %f", mbpsOverhead)
	}
}

func TestEngineRunDownloadAndUpload(t *testing.T) {
	cfg := domain.Config{BufferSize: 64 * 1024}
	engine := service.NewEngine(cfg)

	// Run Download into Discard for 50ms
	bytesDl, elapsedDl, err := engine.RunDownload(io.Discard, 50*time.Millisecond)
	if err != nil {
		t.Fatalf("RunDownload failed: %v", err)
	}
	if bytesDl <= 0 || elapsedDl <= 0 {
		t.Errorf("Expected positive bytes and elapsed, got %d, %v", bytesDl, elapsedDl)
	}

	// Run Upload from repeating reader for 50ms
	dummyReader := bytes.NewReader(make([]byte, 1024*1024))
	bytesUl, elapsedUl, err := engine.RunUpload(dummyReader, 50*time.Millisecond)
	if err != nil {
		t.Fatalf("RunUpload failed: %v", err)
	}
	if bytesUl <= 0 || elapsedUl <= 0 {
		t.Errorf("Expected positive bytes and elapsed, got %d, %v", bytesUl, elapsedUl)
	}
}

func TestStore(t *testing.T) {
	store := service.NewStore()
	store.Add(domain.Result{
		ID:    "1",
		Mbps:  120.5,
		Bytes: 50000000,
	})
	results := store.All()
	if len(results) != 1 {
		t.Fatalf("Expected 1 result, got %d", len(results))
	}
	if results[0].Mbps != 120.5 {
		t.Errorf("Expected Mbps 120.5, got %f", results[0].Mbps)
	}
}
