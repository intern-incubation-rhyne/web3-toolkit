package query

import (
	"context"
	"fmt"
	"math/big"
	"sync"
	"time"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethclient"
)

// QueryConfig represents the configuration for a paginated query
type QueryConfig struct {
	FromBlock  *big.Int
	ToBlock    *big.Int
	Addresses  []common.Address
	Topics     [][]common.Hash
	MaxWorkers int
	ChunkSize  int64
}

// QueryResult represents the result of a single chunk query
type QueryResult struct {
	Logs      []types.Log
	FromBlock *big.Int
	ToBlock   *big.Int
	Err       error
	WorkerID  int
	Duration  time.Duration
}

// PaginatedQuery performs concurrent paginated queries to get logs across a large block range
func PaginatedQuery(ctx context.Context, client *ethclient.Client, config QueryConfig) ([]types.Log, error) {
	// Set defaults
	if config.MaxWorkers <= 0 {
		config.MaxWorkers = 5
	}
	if config.ChunkSize <= 0 {
		config.ChunkSize = 10000
	}

	// Validate input
	if config.FromBlock == nil || config.ToBlock == nil {
		return nil, fmt.Errorf("FromBlock and ToBlock must be specified")
	}
	if config.FromBlock.Cmp(config.ToBlock) > 0 {
		return nil, fmt.Errorf("FromBlock must be less than or equal to ToBlock")
	}

	// Calculate total block range
	blockRange := new(big.Int).Sub(config.ToBlock, config.FromBlock)
	if blockRange.Cmp(big.NewInt(config.ChunkSize)) <= 0 {
		// If range is within chunk size, do a single query
		logs, err := client.FilterLogs(ctx, ethereum.FilterQuery{
			FromBlock: config.FromBlock,
			ToBlock:   config.ToBlock,
			Addresses: config.Addresses,
			Topics:    config.Topics,
		})
		if err != nil {
			return nil, fmt.Errorf("single query failed: %w", err)
		}
		return logs, nil
	}

	// Create channels for results and work
	resultChan := make(chan QueryResult, config.MaxWorkers*2)
	workChan := make(chan []*big.Int, config.MaxWorkers*2)

	// Start worker goroutines
	var wg sync.WaitGroup
	for i := 0; i < config.MaxWorkers; i++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()
			for work := range workChan {
				if len(work) != 2 {
					continue
				}
				fromBlock, toBlock := work[0], work[1]

				start := time.Now()
				logs, err := client.FilterLogs(ctx, ethereum.FilterQuery{
					FromBlock: fromBlock,
					ToBlock:   toBlock,
					Addresses: config.Addresses,
					Topics:    config.Topics,
				})
				duration := time.Since(start)

				resultChan <- QueryResult{
					Logs:      logs,
					FromBlock: fromBlock,
					ToBlock:   toBlock,
					Err:       err,
					WorkerID:  workerID,
					Duration:  duration,
				}
			}
		}(i)
	}

	// Generate work chunks
	go func() {
		defer close(workChan)
		currentBlock := new(big.Int).Set(config.FromBlock)

		for currentBlock.Cmp(config.ToBlock) < 0 {
			chunkEnd := new(big.Int).Add(currentBlock, big.NewInt(config.ChunkSize-1))
			if chunkEnd.Cmp(config.ToBlock) > 0 {
				chunkEnd.Set(config.ToBlock)
			}

			select {
			case workChan <- []*big.Int{new(big.Int).Set(currentBlock), chunkEnd}:
			case <-ctx.Done():
				return
			}

			currentBlock.Add(chunkEnd, big.NewInt(1))
		}
	}()

	// Collect results
	go func() {
		wg.Wait()
		close(resultChan)
	}()

	// Gather all results
	var allLogs []types.Log
	var errors []error
	var totalChunks int
	var successfulChunks int

	for result := range resultChan {
		totalChunks++
		if result.Err != nil {
			errors = append(errors, fmt.Errorf("chunk %d-%d failed (worker %d): %w",
				result.FromBlock, result.ToBlock, result.WorkerID, result.Err))
			continue
		}
		successfulChunks++
		allLogs = append(allLogs, result.Logs...)
	}

	// Check for context cancellation
	if ctx.Err() != nil {
		return nil, fmt.Errorf("query cancelled: %w", ctx.Err())
	}

	// If we have any errors, return them
	if len(errors) > 0 {
		return nil, fmt.Errorf("query completed with %d/%d chunks successful, errors: %v",
			successfulChunks, totalChunks, errors)
	}

	return allLogs, nil
}
