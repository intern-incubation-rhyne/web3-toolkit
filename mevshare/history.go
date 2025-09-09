package mevshare

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"math/big"
	"net/http"
	"os"
	"path/filepath"
	"time"
	"toolkit/query"

	"github.com/ethereum/go-ethereum/ethclient"
)

const (
	Mainnet = "https://mev-share.flashbots.net"
	Sepolia = "https://mev-share-sepolia.flashbots.net"
)

// HistoryResponse represents the structure of the response from the /api/v1/history endpoint
type HistoryResponse struct {
	Block     int64 `json:"block"`
	Timestamp int64 `json:"timestamp"`
	Hint      *Hint `json:"hint"`
}

// HistoryInfo represents the structure of the response from the /api/v1/history/info endpoint
type HistoryInfo struct {
	Count        int64 `json:"count"`
	MinBlock     int64 `json:"minBlock"`
	MaxBlock     int64 `json:"maxBlock"`
	MinTimestamp int64 `json:"minTimestamp"`
	MaxTimestamp int64 `json:"maxTimestamp"`
	MaxLimit     int64 `json:"maxLimit"`
}

// Hint represents the hint data in the history response
type Hint struct {
	Txs         []Transaction `json:"txs"`
	Hash        string        `json:"hash"`
	Logs        interface{}   `json:"logs"`
	GasUsed     string        `json:"gasUsed"`
	MevGasPrice string        `json:"mevGasPrice"`
}

// Transaction represents a transaction in the hint
type Transaction struct {
	To               string `json:"to"`
	CallData         string `json:"callData"`
	FunctionSelector string `json:"functionSelector"`
}

// HistoryRequest represents query parameters for the history endpoint
type HistoryRequest struct {
	BlockStart     *big.Int `json:"blockStart,omitempty"`
	BlockEnd       *big.Int `json:"blockEnd,omitempty"`
	TimestampStart *int64   `json:"timestampStart,omitempty"`
	TimestampEnd   *int64   `json:"timestampEnd,omitempty"`
	Limit          *int     `json:"limit,omitempty"`
	Offset         *int     `json:"offset,omitempty"`
}

// Client represents a MEV-Share API client
type Client struct {
	BaseURL    string
	HTTPClient *http.Client
}

// NewClient creates a new MEV-Share API client
func NewClient(baseURL string) *Client {
	return &Client{
		BaseURL: baseURL,
		HTTPClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// History queries the MEV-Share /api/v1/history endpoint and returns the parsed response
func (c *Client) History(ctx context.Context, req *HistoryRequest) ([]HistoryResponse, error) {
	url := c.BaseURL + "/api/v1/history"

	// Build query parameters
	if req != nil {
		queryParams := make([]string, 0)
		if req.BlockStart != nil {
			queryParams = append(queryParams, fmt.Sprintf("blockStart=%s", req.BlockStart.String()))
		}
		if req.BlockEnd != nil {
			queryParams = append(queryParams, fmt.Sprintf("blockEnd=%s", req.BlockEnd.String()))
		}
		if req.TimestampStart != nil {
			queryParams = append(queryParams, fmt.Sprintf("timestampStart=%d", *req.TimestampStart))
		}
		if req.TimestampEnd != nil {
			queryParams = append(queryParams, fmt.Sprintf("timestampEnd=%d", *req.TimestampEnd))
		}
		if req.Limit != nil {
			queryParams = append(queryParams, fmt.Sprintf("limit=%d", *req.Limit))
		}
		if req.Offset != nil {
			queryParams = append(queryParams, fmt.Sprintf("offset=%d", *req.Offset))
		}

		if len(queryParams) > 0 {
			url += "?" + queryParams[0]
			for i := 1; i < len(queryParams); i++ {
				url += "&" + queryParams[i]
			}
		}
	}

	httpReq, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %v", err)
	}

	fmt.Println("Query: ", httpReq.URL.String())

	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("User-Agent", "web3-toolkit/1.0")

	resp, err := c.HTTPClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch history: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("unexpected status code %d: %s", resp.StatusCode, string(body))
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %v", err)
	}

	var history []HistoryResponse
	if err := json.Unmarshal(body, &history); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %v", err)
	}

	return history, nil
}

// HistoryPaginated queries the MEV-Share /api/v1/history endpoint with automatic pagination
func (c *Client) HistoryPaginated(ctx context.Context, req *HistoryRequest) ([]HistoryResponse, error) {
	pageSize := 500 // Fixed page size

	var allHistory []HistoryResponse
	offset := 0

	for {
		// Create a copy of the request for this page
		pageReq := &HistoryRequest{
			BlockStart:     req.BlockStart,
			BlockEnd:       req.BlockEnd,
			TimestampStart: req.TimestampStart,
			TimestampEnd:   req.TimestampEnd,
			Limit:          &pageSize,
			Offset:         &offset,
		}

		// Fetch current page
		pageHistory, err := c.History(ctx, pageReq)
		if err != nil {
			return nil, fmt.Errorf("failed to fetch page at offset %d: %v", offset, err)
		}

		// If no more data, break
		if len(pageHistory) == 0 {
			break
		}

		// Append to results
		allHistory = append(allHistory, pageHistory...)

		// If we got fewer results than page size, we've reached the end
		if len(pageHistory) < pageSize {
			break
		}

		// Move to next page
		offset += pageSize

		// Check context cancellation
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
		}
	}

	return allHistory, nil
}

// HistoryByBlock queries the history endpoint for a specific block number
func (c *Client) HistoryByBlock(ctx context.Context, blockNumber *big.Int) ([]HistoryResponse, error) {
	req := &HistoryRequest{
		BlockStart: blockNumber,
		BlockEnd:   blockNumber,
	}
	return c.HistoryPaginated(ctx, req)
}

// HistoryWithPagination queries the history endpoint with pagination parameters
func (c *Client) HistoryWithPagination(ctx context.Context, limit, offset int) ([]HistoryResponse, error) {
	req := &HistoryRequest{
		Limit:  &limit,
		Offset: &offset,
	}
	return c.HistoryPaginated(ctx, req)
}

// FetchLatestHistory fetches the most recent history entries
func (c *Client) FetchLatestHistory(ctx context.Context, rpcURL string) (*big.Int, []HistoryResponse, error) {
	ethclient, err := ethclient.Dial(rpcURL)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to connect to RPC: %v", err)
	}
	latestBlock, err := query.LatestBlock(ctx, ethclient)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to get latest block: %v", err)
	}

	req := &HistoryRequest{
		BlockStart: latestBlock,
	}

	history, err := c.HistoryPaginated(ctx, req)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to get history: %v", err)
	}
	return latestBlock, history, nil
}

// HistoryByBlockRange queries the history endpoint for a range of blocks
func (c *Client) HistoryByBlockRange(ctx context.Context, blockStart, blockEnd *big.Int) ([]HistoryResponse, error) {
	req := &HistoryRequest{
		BlockStart: blockStart,
		BlockEnd:   blockEnd,
	}
	return c.HistoryPaginated(ctx, req)
}

// HistoryByTimestampRange queries the history endpoint for a range of timestamps
func (c *Client) HistoryByTimestampRange(ctx context.Context, timestampStart, timestampEnd int64) ([]HistoryResponse, error) {
	req := &HistoryRequest{
		TimestampStart: &timestampStart,
		TimestampEnd:   &timestampEnd,
	}
	return c.HistoryPaginated(ctx, req)
}

// HistoryInfo queries the MEV-Share /api/v1/history/info endpoint and returns information about available historical data
func (c *Client) HistoryInfo(ctx context.Context) (*HistoryInfo, error) {
	url := c.BaseURL + "/api/v1/history/info"

	httpReq, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %v", err)
	}

	fmt.Println("Query: ", httpReq.URL.String())

	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("User-Agent", "web3-toolkit/1.0")

	resp, err := c.HTTPClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch history info: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("unexpected status code %d: %s", resp.StatusCode, string(body))
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %v", err)
	}

	var historyInfo HistoryInfo
	if err := json.Unmarshal(body, &historyInfo); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %v", err)
	}

	return &historyInfo, nil
}

// SaveToFile saves HistoryResponse data to a JSON file
func SaveToFile(history []HistoryResponse, filename string) error {
	// Create directory if it doesn't exist
	dir := filepath.Dir(filename)
	if dir != "." {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return fmt.Errorf("failed to create directory %s: %v", dir, err)
		}
	}

	// Marshal data to JSON with indentation
	data, err := json.MarshalIndent(history, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal history to JSON: %v", err)
	}

	// Write to file
	if err := os.WriteFile(filename, data, 0644); err != nil {
		return fmt.Errorf("failed to write file %s: %v", filename, err)
	}

	return nil
}
