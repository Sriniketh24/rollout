package load

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

const baseURL = "http://localhost:8080"

type evalRequest struct {
	ProjectID     string         `json:"project_id"`
	EnvironmentID string         `json:"environment_id"`
	Context       evalContext    `json:"context"`
}

type evalContext struct {
	Key        string         `json:"key"`
	Attributes map[string]any `json:"attributes"`
}

func TestLoadEvaluateFlags(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping load test in short mode")
	}

	client := &http.Client{
		Timeout: 5 * time.Second,
		Transport: &http.Transport{
			MaxIdleConns:        200,
			MaxIdleConnsPerHost: 200,
			IdleConnTimeout:     90 * time.Second,
		},
	}

	concurrency := 50
	duration := 30 * time.Second
	var totalRequests atomic.Int64
	var totalErrors atomic.Int64
	var totalLatencyNs atomic.Int64
	var maxLatencyNs atomic.Int64

	var wg sync.WaitGroup
	done := make(chan struct{})

	for i := 0; i < concurrency; i++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()
			for {
				select {
				case <-done:
					return
				default:
				}

				req := evalRequest{
					ProjectID:     "test-project",
					EnvironmentID: "production",
					Context: evalContext{
						Key: fmt.Sprintf("user-%d-%d", workerID, time.Now().UnixNano()),
						Attributes: map[string]any{
							"country": "US",
							"plan":    "premium",
							"version": "2.1.0",
						},
					},
				}

				body, _ := json.Marshal(req)
				start := time.Now()
				resp, err := client.Post(
					baseURL+"/api/v1/projects/test-project/evaluate",
					"application/json",
					bytes.NewReader(body),
				)
				elapsed := time.Since(start)
				totalRequests.Add(1)
				totalLatencyNs.Add(elapsed.Nanoseconds())

				for {
					old := maxLatencyNs.Load()
					if elapsed.Nanoseconds() <= old || maxLatencyNs.CompareAndSwap(old, elapsed.Nanoseconds()) {
						break
					}
				}

				if err != nil {
					totalErrors.Add(1)
					continue
				}
				io.Copy(io.Discard, resp.Body)
				resp.Body.Close()

				if resp.StatusCode != http.StatusOK {
					totalErrors.Add(1)
				}
			}
		}(i)
	}

	time.Sleep(duration)
	close(done)
	wg.Wait()

	total := totalRequests.Load()
	errors := totalErrors.Load()
	avgLatency := time.Duration(0)
	if total > 0 {
		avgLatency = time.Duration(totalLatencyNs.Load() / total)
	}
	maxLatency := time.Duration(maxLatencyNs.Load())
	rps := float64(total) / duration.Seconds()

	t.Logf("=== Load Test Results ===")
	t.Logf("Duration:     %s", duration)
	t.Logf("Concurrency:  %d", concurrency)
	t.Logf("Total Reqs:   %d", total)
	t.Logf("Errors:       %d (%.2f%%)", errors, float64(errors)/float64(total)*100)
	t.Logf("RPS:          %.2f", rps)
	t.Logf("Avg Latency:  %s", avgLatency)
	t.Logf("Max Latency:  %s", maxLatency)
}

func TestLoadSSEConnections(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping load test in short mode")
	}

	numClients := 100
	duration := 15 * time.Second
	var connected atomic.Int64
	var eventsReceived atomic.Int64

	var wg sync.WaitGroup
	done := make(chan struct{})

	for i := 0; i < numClients; i++ {
		wg.Add(1)
		go func(clientID int) {
			defer wg.Done()
			url := fmt.Sprintf("%s/api/v1/projects/test-project/stream/production", baseURL)
			resp, err := http.Get(url)
			if err != nil {
				return
			}
			defer resp.Body.Close()
			connected.Add(1)

			buf := make([]byte, 4096)
			for {
				select {
				case <-done:
					return
				default:
				}
				n, err := resp.Body.Read(buf)
				if err != nil {
					return
				}
				if n > 0 {
					eventsReceived.Add(1)
				}
			}
		}(i)
	}

	time.Sleep(2 * time.Second)
	t.Logf("Connected SSE clients: %d", connected.Load())

	time.Sleep(duration)
	close(done)
	wg.Wait()

	t.Logf("=== SSE Load Test Results ===")
	t.Logf("Target Clients:    %d", numClients)
	t.Logf("Connected:         %d", connected.Load())
	t.Logf("Events Received:   %d", eventsReceived.Load())
}

func TestLoadIngestPipeline(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping load test in short mode")
	}

	client := &http.Client{
		Timeout: 10 * time.Second,
		Transport: &http.Transport{
			MaxIdleConns:        100,
			MaxIdleConnsPerHost: 100,
		},
	}

	batchSize := 100
	numBatches := 100
	concurrency := 10
	var totalEvents atomic.Int64
	var totalErrors atomic.Int64

	start := time.Now()
	var wg sync.WaitGroup
	sem := make(chan struct{}, concurrency)

	for batch := 0; batch < numBatches; batch++ {
		wg.Add(1)
		sem <- struct{}{}
		go func(batchNum int) {
			defer wg.Done()
			defer func() { <-sem }()

			events := make([]map[string]any, batchSize)
			for i := 0; i < batchSize; i++ {
				events[i] = map[string]any{
					"flag_key":       "test-flag",
					"environment_id": "production",
					"user_key":       fmt.Sprintf("user-%d-%d", batchNum, i),
					"variation":      `"control"`,
					"reason":         "FALLTHROUGH",
					"timestamp":      time.Now().Format(time.RFC3339Nano),
				}
			}

			body, _ := json.Marshal(map[string]any{"events": events})
			resp, err := client.Post(
				baseURL+"/api/v1/events/exposures",
				"application/json",
				bytes.NewReader(body),
			)
			if err != nil {
				totalErrors.Add(1)
				return
			}
			io.Copy(io.Discard, resp.Body)
			resp.Body.Close()

			if resp.StatusCode == http.StatusAccepted {
				totalEvents.Add(int64(batchSize))
			} else {
				totalErrors.Add(1)
			}
		}(batch)
	}

	wg.Wait()
	elapsed := time.Since(start)

	total := totalEvents.Load()
	eps := float64(total) / elapsed.Seconds()

	t.Logf("=== Ingest Load Test Results ===")
	t.Logf("Duration:      %s", elapsed)
	t.Logf("Batch Size:    %d", batchSize)
	t.Logf("Num Batches:   %d", numBatches)
	t.Logf("Total Events:  %d", total)
	t.Logf("Errors:        %d", totalErrors.Load())
	t.Logf("Events/sec:    %.2f", eps)
}
