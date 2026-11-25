package delivery_test

import (
	"bytes"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/addpur/custome-speedtest/bundlefx"
	"github.com/addpur/custome-speedtest/fe"
	"github.com/addpur/custome-speedtest/rest/delivery"
	"github.com/addpur/custome-speedtest/rest/service"
	"github.com/addpur/custome-speedtest/speedtest/domain"
	stservice "github.com/addpur/custome-speedtest/speedtest/service"
	"github.com/gofiber/fiber/v2"
)

func setupTestApp() *fiber.App {
	cfg := domain.Config{
		ListenAddr: ":3000",
		BufferSize: 1024 * 1024, // 1MB for tests
		Duration:   5 * time.Second,
		Parallel:   4,
		Overhead:   1.0,
	}

	engine := stservice.NewEngine(cfg)
	registry := stservice.NewRegistry(cfg)
	store := stservice.NewStore()
	apiService := service.NewAPIService(cfg, engine, registry, store)

	app := bundlefx.NewFiberApp()
	handler := delivery.NewHandler(apiService)
	handler.RegisterRoutes(app)

	// Add static frontend routes as in fe/modul.go
	app.Get("/", func(c *fiber.Ctx) error {
		data, err := fe.StaticFS.ReadFile("static/index.html")
		if err != nil {
			return c.Status(404).SendString("index.html not found")
		}
		c.Set("Content-Type", "text/html; charset=utf-8")
		return c.Send(data)
	})
	app.Get("/static/*", func(c *fiber.Ctx) error {
		path := c.Params("*")
		data, err := fe.StaticFS.ReadFile("static/" + path)
		if err != nil {
			return c.Status(404).SendString("Not found")
		}
		return c.Send(data)
	})

	return app
}

func TestHealthCheck(t *testing.T) {
	app := setupTestApp()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/health", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("Failed to execute request: %v", err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status 200, got %d", resp.StatusCode)
	}
}

func TestPing(t *testing.T) {
	app := setupTestApp()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/ping", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("Failed to execute request: %v", err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status 200, got %d", resp.StatusCode)
	}
	body, _ := io.ReadAll(resp.Body)
	if !bytes.Contains(body, []byte("timestamp")) {
		t.Errorf("Expected body to contain 'timestamp', got %s", string(body))
	}
}

func TestDownloadChunk(t *testing.T) {
	app := setupTestApp()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/download?size=65536", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("Failed to execute request: %v", err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status 200, got %d", resp.StatusCode)
	}
	body, _ := io.ReadAll(resp.Body)
	if len(body) != 65536 {
		t.Errorf("Expected 65536 bytes, got %d bytes", len(body))
	}
}

func TestUploadChunk(t *testing.T) {
	app := setupTestApp()
	payload := make([]byte, 32768)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/upload", bytes.NewReader(payload))
	req.Header.Set("Content-Type", "application/octet-stream")
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("Failed to execute request: %v", err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status 200, got %d", resp.StatusCode)
	}
}

func TestSaveAndGetResults(t *testing.T) {
	app := setupTestApp()
	jsonBody := []byte(`{"type":"standard","server":"Local","bytes":1000000,"mbps":150.5,"ping_ms":12.4,"jitter_ms":1.2,"parallel":4}`)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/results", bytes.NewReader(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("Failed to save result: %v", err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status 200, got %d", resp.StatusCode)
	}

	getReq := httptest.NewRequest(http.MethodGet, "/api/v1/results", nil)
	getResp, err := app.Test(getReq)
	if err != nil {
		t.Fatalf("Failed to get results: %v", err)
	}
	if getResp.StatusCode != http.StatusOK {
		t.Errorf("Expected status 200, got %d", getResp.StatusCode)
	}
}

func TestFrontendServing(t *testing.T) {
	app := setupTestApp()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("Failed to get /: %v", err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status 200, got %d", resp.StatusCode)
	}

	reqJS := httptest.NewRequest(http.MethodGet, "/static/app.js", nil)
	respJS, err := app.Test(reqJS)
	if err != nil {
		t.Fatalf("Failed to get app.js: %v", err)
	}
	if respJS.StatusCode != http.StatusOK {
		t.Errorf("Expected status 200 for app.js, got %d", respJS.StatusCode)
	}
}

func TestServeCLIScript(t *testing.T) {
	app := setupTestApp()
	req := httptest.NewRequest(http.MethodGet, "/cli", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("Failed to get /cli: %v", err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status 200 for /cli, got %d", resp.StatusCode)
	}
	body, _ := io.ReadAll(resp.Body)
	if !bytes.Contains(body, []byte("#!/usr/bin/env bash")) {
		t.Errorf("Expected script to start with bash shebang")
	}
}
