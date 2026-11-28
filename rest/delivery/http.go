package delivery

import (
	"bufio"
	"fmt"
	"io"
	"strings"
	"time"

	restdomain "github.com/addpur/custome-speedtest/rest/domain"
	"github.com/addpur/custome-speedtest/rest/service"
	"github.com/gofiber/fiber/v2"
)

type Handler struct {
	api *service.APIService
}

func NewHandler(api *service.APIService) *Handler {
	return &Handler{api: api}
}

func (h *Handler) RegisterRoutes(app *fiber.App) {
	api := app.Group("/api/v1")
	api.Get("/health", h.HealthCheck)
	api.Get("/ping", h.Ping)
	api.Get("/download", h.Download)
	api.Post("/upload", h.Upload)
	api.Get("/config", h.GetConfig)
	api.Get("/servers", h.GetServers)
	api.Get("/results", h.GetResults)
	api.Post("/results", h.SaveResult)
	api.Get("/stress", h.GetStress)
	api.Post("/stress", h.RunStress)

	// Direct aliases for OpenSpeedTest compatibility
	app.Get("/ping", h.Ping)
	app.Get("/download", h.Download)
	app.Post("/upload", h.Upload)

	// Zero-install CLI script for client VMs (curl -sL <server>/cli | bash)
	app.Get("/cli", h.ServeCLIScript)
	app.Get("/speedtest.sh", h.ServeCLIScript)
}

func (h *Handler) HealthCheck(c *fiber.Ctx) error {
	return c.JSON(fiber.Map{
		"status":  "ok",
		"message": "Server is running",
	})
}

func (h *Handler) Ping(c *fiber.Ctx) error {
	c.Set("Cache-Control", "no-store, no-cache, must-revalidate, max-age=0")
	c.Set("Pragma", "no-cache")
	c.Set("Expires", "0")
	return c.JSON(restdomain.PingResponse{
		Timestamp: time.Now().UnixMilli(),
	})
}

func (h *Handler) Download(c *fiber.Ctx) error {
	c.Set("Content-Type", "application/octet-stream")
	c.Set("Cache-Control", "no-store, no-cache, must-revalidate, max-age=0")
	c.Set("Pragma", "no-cache")
	c.Set("Expires", "0")
	c.Set("Content-Disposition", "attachment; filename=speedtest.bin")

	buf := h.api.Engine().Buffer()
	bufLen := len(buf)
	if bufLen == 0 {
		return c.SendStatus(fiber.StatusInternalServerError)
	}

	size := c.QueryInt("size", 0)
	durationSec := c.QueryInt("duration", 0)

	if size > 0 {
		if size <= bufLen {
			return c.Send(buf[:size])
		}
		c.Context().SetBodyStreamWriter(func(w *bufio.Writer) {
			remaining := int64(size)
			for remaining > 0 {
				toWrite := int(remaining)
				if toWrite > bufLen {
					toWrite = bufLen
				}
				n, err := w.Write(buf[:toWrite])
				if err != nil {
					return
				}
				remaining -= int64(n)
			}
			_ = w.Flush()
		})
		return nil
	}

	if durationSec <= 0 {
		return c.Send(buf)
	}

	c.Context().SetBodyStreamWriter(func(w *bufio.Writer) {
		deadline := time.Now().Add(time.Duration(durationSec) * time.Second)
		for time.Now().Before(deadline) {
			if _, err := w.Write(buf); err != nil {
				return
			}
			if err := w.Flush(); err != nil {
				return
			}
		}
	})
	return nil
}

func (h *Handler) Upload(c *fiber.Ctx) error {
	c.Set("Cache-Control", "no-store, no-cache, must-revalidate, max-age=0")
	c.Set("Pragma", "no-cache")

	var bytesRead int64
	bodyStream := c.Context().RequestBodyStream()
	if bodyStream != nil {
		bytesRead, _ = io.Copy(io.Discard, bodyStream)
	} else {
		bytesRead = int64(len(c.Body()))
	}

	return c.JSON(fiber.Map{
		"status": "ok",
		"bytes":  bytesRead,
	})
}

func (h *Handler) GetConfig(c *fiber.Ctx) error {
	return c.JSON(h.api.GetConfig())
}

func (h *Handler) GetServers(c *fiber.Ctx) error {
	keywordsStr := c.Query("keywords")
	var keywords []string
	if keywordsStr != "" {
		keywords = strings.Split(keywordsStr, ",")
		for i := range keywords {
			keywords[i] = strings.TrimSpace(keywords[i])
		}
	}
	servers := h.api.GetServers(keywords)
	return c.JSON(servers)
}

func (h *Handler) GetResults(c *fiber.Ctx) error {
	results := h.api.GetResults()
	return c.JSON(results)
}

func (h *Handler) SaveResult(c *fiber.Ctx) error {
	var req restdomain.SaveResultRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid request"})
	}

	res := restdomain.ResultResponse{
		ID:        time.Now().Format("20060102150405"),
		Type:      req.Type,
		Server:    req.Server,
		Bytes:     req.Bytes,
		Mbps:      req.Mbps,
		PingMs:    req.PingMs,
		JitterMs:  req.JitterMs,
		Parallel:  req.Parallel,
		Timestamp: time.Now(),
	}
	h.api.AddResult(res)
	return c.JSON(res)
}

func (h *Handler) GetStress(c *fiber.Ctx) error {
	resp := h.api.GetStress()
	return c.JSON(resp)
}

func (h *Handler) RunStress(c *fiber.Ctx) error {
	var req restdomain.RunRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid request"})
	}

	go service.StartStress(c.Context(), &req)

	results := h.api.GetResults()
	return c.JSON(results)
}

func (h *Handler) ServeCLIScript(c *fiber.Ctx) error {
	proto := c.Protocol()
	host := c.Hostname()
	serverURL := fmt.Sprintf("%s://%s", proto, host)

	script := fmt.Sprintf(`#!/usr/bin/env bash
# ==============================================================================
# Custome Speedtest (ct-speedtest) CLI
# Zero-Install Speedtest for Client VMs & Servers
# ==============================================================================
set -e

SERVER="%s"
PARALLEL=4
DURATION=10

while [[ $# -gt 0 ]]; do
  case $1 in
    -s|--server)
      SERVER="$2"
      shift 2
      ;;
    -p|--parallel)
      PARALLEL="$2"
      shift 2
      ;;
    -d|--duration)
      DURATION="$2"
      shift 2
      ;;
    -h|--help)
      echo "Custome Speedtest CLI Script"
      echo "Usage: curl -sL <server>/cli | bash -s -- [options]"
      echo ""
      echo "Options:"
      echo "  -s, --server <url>     Target server URL (default: $SERVER)"
      echo "  -p, --parallel <num>   Parallel streams (default: 4)"
      echo "  -d, --duration <sec>   Duration in seconds per phase (default: 10)"
      exit 0
      ;;
    *)
      shift
      ;;
  esac
done

command -v curl >/dev/null 2>&1 || { echo >&2 "Error: curl is required but not installed."; exit 1; }

echo "======================================================"
echo "  ⚡ Custome Speedtest (ct-speedtest) CLI"
echo "======================================================"
echo "  Target Server : $SERVER"
echo "  Streams       : $PARALLEL parallel connections"
echo "  Duration      : ${DURATION}s per phase"
echo "------------------------------------------------------"

# 1. Latency & Jitter Test
printf "[*] Testing Latency (Ping)... "
LATENCY_SUM=0
SAMPLES=5
for i in $(seq 1 $SAMPLES); do
  RTT=$(curl -s -o /dev/null -w "%%{time_total}\n" "$SERVER/api/v1/ping" || echo "0")
  LATENCY_MS=$(awk "BEGIN {print $RTT * 1000}")
  LATENCY_SUM=$(awk "BEGIN {print $LATENCY_SUM + $LATENCY_MS}")
done
AVG_LATENCY=$(awk "BEGIN {printf \"%%.2f\", $LATENCY_SUM / $SAMPLES}")
echo "Done."
echo "    Latency : ${AVG_LATENCY} ms"

format_bytes() {
  local b=$1
  if [ $(awk "BEGIN {print ($b >= 1125899906842624)}") -eq 1 ]; then
    awk "BEGIN {printf \"%%.2f PB\", $b / 1125899906842624}"
  elif [ $(awk "BEGIN {print ($b >= 1099511627776)}") -eq 1 ]; then
    awk "BEGIN {printf \"%%.2f TB\", $b / 1099511627776}"
  elif [ $(awk "BEGIN {print ($b >= 1073741824)}") -eq 1 ]; then
    awk "BEGIN {printf \"%%.2f GB\", $b / 1073741824}"
  elif [ $(awk "BEGIN {print ($b >= 1048576)}") -eq 1 ]; then
    awk "BEGIN {printf \"%%.2f MB\", $b / 1048576}"
  elif [ $(awk "BEGIN {print ($b >= 1024)}") -eq 1 ]; then
    awk "BEGIN {printf \"%%.2f KB\", $b / 1024}"
  else
    echo "${b} B"
  fi
}

# 2. Download Test
echo "[*] Testing Download Speed ($PARALLEL streams, ${DURATION}s)..."
TMP_DIR=$(mktemp -d /tmp/ct_speedtest.XXXXXX)
trap 'rm -rf "$TMP_DIR"' EXIT

START_TIME=$(date +%%s)
for i in $(seq 1 $PARALLEL); do
  curl -s -o /dev/null -w "%%{size_download}\n" \
    "$SERVER/api/v1/download?duration=${DURATION}&stream=${i}" > "$TMP_DIR/dl_${i}.txt" &
done
wait

END_TIME=$(date +%%s)
ELAPSED=$((END_TIME - START_TIME))
if [ "$ELAPSED" -le 0 ]; then ELAPSED=$DURATION; fi

TOTAL_DL_BYTES=0
for f in "$TMP_DIR"/dl_*.txt; do
  BYTES=$(cat "$f" 2>/dev/null || echo 0)
  TOTAL_DL_BYTES=$((TOTAL_DL_BYTES + BYTES))
done

DL_TRANSFERRED=$(format_bytes $TOTAL_DL_BYTES)
DL_MBPS=$(awk "BEGIN {printf \"%%.2f\", ($TOTAL_DL_BYTES * 8) / ($ELAPSED * 1000000)}")
echo "    Download: ${DL_MBPS} Mbps (${DL_TRANSFERRED} transferred)"

# 3. Upload Test
echo "[*] Testing Upload Speed ($PARALLEL streams, ${DURATION}s)..."
UPLOAD_CHUNK="$TMP_DIR/chunk.bin"
dd if=/dev/zero of="$UPLOAD_CHUNK" bs=1048576 count=4 2>/dev/null

START_UL_TIME=$(date +%%s)
for i in $(seq 1 $PARALLEL); do
  (
    END_UL=$(( $(date +%%s) + DURATION ))
    STREAM_BYTES=0
    while [ $(date +%%s) -lt $END_UL ]; do
      BYTES_SENT=$(curl -s -o /dev/null -w "%%{size_upload}\n" -X POST \
        -H "Content-Type: application/octet-stream" \
        --data-binary @"$UPLOAD_CHUNK" "$SERVER/api/v1/upload?stream=${i}" || echo 0)
      STREAM_BYTES=$((STREAM_BYTES + BYTES_SENT))
    done
    echo "$STREAM_BYTES" > "$TMP_DIR/ul_${i}.txt"
  ) &
done
wait
END_UL_TIME=$(date +%%s)
ELAPSED_UL=$((END_UL_TIME - START_UL_TIME))
if [ "$ELAPSED_UL" -le 0 ]; then ELAPSED_UL=$DURATION; fi

TOTAL_UL_BYTES=0
for f in "$TMP_DIR"/ul_*.txt; do
  BYTES=$(cat "$f" 2>/dev/null || echo 0)
  TOTAL_UL_BYTES=$((TOTAL_UL_BYTES + BYTES))
done

UL_TRANSFERRED=$(format_bytes $TOTAL_UL_BYTES)
UL_MBPS=$(awk "BEGIN {printf \"%%.2f\", ($TOTAL_UL_BYTES * 8) / ($ELAPSED_UL * 1000000)}")
echo "    Upload  : ${UL_MBPS} Mbps (${UL_TRANSFERRED} transferred)"

echo "------------------------------------------------------"
echo "  🏆 TEST SUMMARY:"
echo "    Latency  : ${AVG_LATENCY} ms"
echo "    Download : ${DL_MBPS} Mbps"
echo "    Upload   : ${UL_MBPS} Mbps"
echo "======================================================"
`, serverURL)

	c.Set("Content-Type", "text/plain; charset=utf-8")
	c.Set("Cache-Control", "no-cache, no-store, must-revalidate")
	return c.SendString(script)
}
