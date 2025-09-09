package mevshare

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// StreamEvent represents an event from the MEV-Share event stream
type StreamEvent struct {
	Hash             string        `json:"hash"`
	Logs             []Log         `json:"logs"`
	Txs              []Transaction `json:"txs"`
	To               *string       `json:"to"`
	FunctionSelector *string       `json:"functionSelector"`
	CallData         *string       `json:"callData"`
	GasUsed          *string       `json:"gasUsed"`
	MevGasPrice      *string       `json:"mevGasPrice"`
}

// Log represents a log entry in the stream event
type Log struct {
	Address string   `json:"address"`
	Topics  []string `json:"topics"`
	Data    string   `json:"data"`
}

// StreamReader handles reading from the MEV-Share event stream
type StreamReader struct {
	BaseURL    string
	HTTPClient *http.Client
}

// NewStreamReader creates a new MEV-Share event stream reader
func NewStreamReader(baseURL string) *StreamReader {
	return &StreamReader{
		BaseURL: baseURL,
		HTTPClient: &http.Client{
			Timeout: 0, // No timeout for streaming
		},
	}
}

// ReadStream reads from the MEV-Share event stream and pushes events to the provided channel
func (sr *StreamReader) ReadStream(ctx context.Context, eventChan chan<- StreamEvent) error {
	url := sr.BaseURL // The stream endpoint is the base URL itself

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %v", err)
	}

	req.Header.Set("Accept", "text/event-stream")
	req.Header.Set("Cache-Control", "no-cache")
	req.Header.Set("User-Agent", "web3-toolkit/1.0")

	resp, err := sr.HTTPClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to connect to stream: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("unexpected status code %d: %s", resp.StatusCode, string(body))
	}

	scanner := bufio.NewScanner(resp.Body)

	for scanner.Scan() {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		line := scanner.Text()

		// Skip empty lines and ping messages
		if line == "" || line == ":ping" {
			continue
		}

		// Parse Server-Sent Events format
		if strings.HasPrefix(line, "data: ") {
			data := strings.TrimPrefix(line, "data: ")

			// Skip empty data
			if data == "" {
				continue
			}

			var event StreamEvent
			if err := json.Unmarshal([]byte(data), &event); err != nil {
				fmt.Printf("Failed to unmarshal event: %v, data: %s\n", err, data)
				continue
			}

			// Send event to channel
			select {
			case eventChan <- event:
			case <-ctx.Done():
				return ctx.Err()
			}
		}
	}

	if err := scanner.Err(); err != nil {
		return fmt.Errorf("error reading stream: %v", err)
	}

	return nil
}

// ReadStreamWithRetry reads from the stream with automatic reconnection on errors
func (sr *StreamReader) ReadStreamWithRetry(ctx context.Context, eventChan chan<- StreamEvent, retryDelay time.Duration) {
	for {
		select {
		case <-ctx.Done():
			return
		default:
		}

		err := sr.ReadStream(ctx, eventChan)
		if err != nil {
			fmt.Printf("Stream error: %v, retrying in %v\n", err, retryDelay)

			select {
			case <-time.After(retryDelay):
				continue
			case <-ctx.Done():
				return
			}
		}
	}
}
