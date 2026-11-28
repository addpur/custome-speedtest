package cli_test

import (
	"fmt"
	"net/http"
	"testing"

	"github.com/addpur/custome-speedtest/bundlefx"
	"github.com/addpur/custome-speedtest/cli"
	"github.com/addpur/custome-speedtest/rest/delivery"
	"github.com/addpur/custome-speedtest/rest/service"
	"github.com/addpur/custome-speedtest/speedtest/domain"
	stservice "github.com/addpur/custome-speedtest/speedtest/service"
)

type fiberRoundTripper struct {
	handler *delivery.Handler
}

func (rt *fiberRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	app := bundlefx.NewFiberApp()
	rt.handler.RegisterRoutes(app)
	return app.Test(req, 10000)
}

func TestCLIRun(t *testing.T) {
	cfg := domain.Config{
		ListenAddr: ":0",
		BufferSize: 256 * 1024,
	}
	engine := stservice.NewEngine(cfg)
	registry := stservice.NewRegistry(cfg)
	store := stservice.NewStore()
	apiService := service.NewAPIService(cfg, engine, registry, store)
	handler := delivery.NewHandler(apiService)

	rt := &fiberRoundTripper{handler: handler}

	// Run CLI test with short 1s duration
	args := []string{
		"-server", "http://speedtest.local",
		"-duration", "1",
		"-parallel", "2",
		"-json",
	}

	err := cli.RunWithTransport(args, rt)
	if err != nil {
		t.Fatalf("CLI Run failed: %v", err)
	}
}

func TestCLIHelp(t *testing.T) {
	err := cli.Run([]string{"-h"})
	if err == nil {
		t.Logf("Expected flag.ErrHelp")
	}
}

func TestFormatBytes(t *testing.T) {
	tests := []struct {
		bytes    int64
		expected string
	}{
		{500, "500 B"},
		{1024, "1.00 KB"},
		{1536, "1.50 KB"},
		{10 * 1024 * 1024, "10.00 MB"},
		{2500 * 1024 * 1024, "2.44 GB"},
		{5 * 1024 * 1024 * 1024 * 1024, "5.00 TB"},
		{2 * 1024 * 1024 * 1024 * 1024 * 1024, "2.00 PB"},
	}

	for _, tc := range tests {
		actual := cli.FormatBytes(tc.bytes)
		if actual != tc.expected {
			t.Errorf("FormatBytes(%d) = %s, expected %s", tc.bytes, actual, tc.expected)
		}
	}
}

func TestCLIInvalidServer(t *testing.T) {
	err := cli.RunWithTransport([]string{"-server", "http://invalid-unreachable-host:9999", "-duration", "1"}, &mockFailingRT{})
	if err == nil {
		t.Errorf("Expected error for unreachable server")
	}
}

type mockFailingRT struct{}

func (m *mockFailingRT) RoundTrip(req *http.Request) (*http.Response, error) {
	return nil, fmt.Errorf("connection refused")
}
