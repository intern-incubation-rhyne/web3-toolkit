package query

import (
	"context"
	"encoding/json"
	"fmt"
	"math/big"
	"os"
	"path/filepath"
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
				var logs []types.Log
				var err error
				for {
					logs, err = client.FilterLogs(ctx, ethereum.FilterQuery{
						FromBlock: fromBlock,
						ToBlock:   toBlock,
						Addresses: config.Addresses,
						Topics:    config.Topics,
					})
					if err != nil {
						time.Sleep(1 * time.Second)
						continue
					}
					break
				}
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

// BundleSearchConfig represents configuration for bundle search
type BundleSearchConfig struct {
	EventSignatures []string
	Addresses       []common.Address // Addresses to match for each transaction in sequence
	StartBlock      *big.Int
	EndBlock        *big.Int
	MaxWorkers      int
	ChunkSize       int64
	OutputFile      string
}

// SearchBundle searches for transaction bundles matching a sequence of event signatures
func SearchBundle(ctx context.Context, client *ethclient.Client, config BundleSearchConfig) ([][]*types.Transaction, error) {
	if len(config.EventSignatures) == 0 {
		return nil, fmt.Errorf("at least one event signature is required")
	}
	if config.StartBlock == nil || config.EndBlock == nil {
		return nil, fmt.Errorf("StartBlock and EndBlock must be specified")
	}
	if config.StartBlock.Cmp(config.EndBlock) > 0 {
		return nil, fmt.Errorf("StartBlock must be less than or equal to EndBlock")
	}
	if len(config.Addresses) > 0 && len(config.Addresses) != len(config.EventSignatures) {
		return nil, fmt.Errorf("number of addresses must match number of event signatures")
	}

	// Set defaults
	if config.MaxWorkers <= 0 {
		config.MaxWorkers = 3
	}
	if config.ChunkSize <= 0 {
		config.ChunkSize = 1000
	}
	if config.OutputFile == "" {
		config.OutputFile = fmt.Sprintf("bundles_%s_%s.json",
			config.StartBlock.String(), config.EndBlock.String())
	}

	fmt.Printf("Searching for bundles with %d event signatures from block %s to %s\n",
		len(config.EventSignatures), config.StartBlock.String(), config.EndBlock.String())

	// First, get all logs matching the first event signature
	firstSig := config.EventSignatures[0]
	firstSigHash := common.HexToHash(firstSig)

	queryConfig := QueryConfig{
		FromBlock:  config.StartBlock,
		ToBlock:    config.EndBlock,
		Addresses:  []common.Address{config.Addresses[0]},
		Topics:     [][]common.Hash{{firstSigHash}},
		MaxWorkers: config.MaxWorkers,
		ChunkSize:  config.ChunkSize,
	}

	allLogs, err := PaginatedQuery(ctx, client, queryConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to query first signature logs: %w", err)
	}

	fmt.Printf("Found %d logs matching first signature %s\n", len(allLogs), firstSig)

	// Group logs by block number and transaction hash
	blockTxMap := make(map[string]map[string][]types.Log)
	for _, log := range allLogs {
		blockKey := big.NewInt(int64(log.BlockNumber)).String()
		txKey := log.TxHash.Hex()

		if blockTxMap[blockKey] == nil {
			blockTxMap[blockKey] = make(map[string][]types.Log)
		}
		blockTxMap[blockKey][txKey] = append(blockTxMap[blockKey][txKey], log)
	}

	var bundles [][]*types.Transaction
	var mu sync.Mutex

	// Process each block
	for blockNumStr, txMap := range blockTxMap {
		blockNum := new(big.Int)
		blockNum.SetString(blockNumStr, 10)

		// Get block details to get transaction order
		block, err := client.BlockByNumber(ctx, blockNum)
		if err != nil {
			fmt.Printf("Failed to get block %s: %v\n", blockNumStr, err)
			continue
		}

		// Collect all potential starting transaction indices
		var startIndices []int
		for txHashStr := range txMap {
			txHash := common.HexToHash(txHashStr)

			// Find transaction index in block
			txIndex := -1
			for i, tx := range block.Transactions() {
				if tx.Hash() == txHash {
					txIndex = i
					break
				}
			}

			if txIndex != -1 {
				startIndices = append(startIndices, txIndex)
			}
		}

		// Process potential bundles concurrently
		if len(startIndices) > 0 {
			var wg sync.WaitGroup
			bundleChan := make(chan []*types.Transaction, len(startIndices))

			for _, startIndex := range startIndices {
				wg.Add(1)
				go func(idx int) {
					defer wg.Done()
					bundleTxs := findBundleInBlock(ctx, client, block, idx, config.EventSignatures, config.Addresses)
					if len(bundleTxs) > 0 {
						bundleChan <- bundleTxs
					}
				}(startIndex)
			}

			// Close channel when all goroutines are done
			go func() {
				wg.Wait()
				close(bundleChan)
			}()

			// Collect results
			for bundleTxs := range bundleChan {
				mu.Lock()
				bundles = append(bundles, bundleTxs)
				mu.Unlock()
			}
		}
	}
	return bundles, nil
}

// findBundleInBlock attempts to find a bundle starting from a specific transaction index
func findBundleInBlock(ctx context.Context, client *ethclient.Client, block *types.Block, startIndex int, signatures []string, addresses []common.Address) []*types.Transaction {
	transactions := block.Transactions()

	// We need at least as many transactions as signatures
	if len(transactions)-startIndex < len(signatures) {
		return nil
	}

	var bundleTxs []*types.Transaction

	// Check each signature in sequence
	for i, sig := range signatures {
		txIndex := startIndex + i
		if txIndex >= len(transactions) {
			return nil
		}

		tx := transactions[txIndex]
		txHash := tx.Hash()

		// Get transaction receipt to check logs
		receipt, err := client.TransactionReceipt(ctx, txHash)
		if err != nil {
			return nil
		}

		// Check if any log matches the current signature and address
		sigHash := common.HexToHash(sig)
		found := false
		for _, log := range receipt.Logs {
			if len(log.Topics) > 0 && log.Topics[0] == sigHash {
				// Check address matching if addresses are provided
				if len(addresses) > i {
					expectedAddr := addresses[i]
					// If address is zero address, it matches any address
					if expectedAddr == (common.Address{}) || log.Address == expectedAddr {
						found = true
						break
					}
				} else {
					// No address filtering, just check signature
					found = true
					break
				}
			}
		}

		if !found {
			return nil
		}

		// Add transaction to bundle
		bundleTxs = append(bundleTxs, tx)
	}

	return bundleTxs
}

// saveBundlesToFile saves bundles (as lists of transactions) to a JSON file
func SaveBundlesToFile(bundles [][]*types.Transaction, filename string) error {
	// Create directory if it doesn't exist
	dir := filepath.Dir(filename)
	if dir != "." {
		err := os.MkdirAll(dir, 0755)
		if err != nil {
			return fmt.Errorf("failed to create directory: %w", err)
		}
	}

	data, err := json.MarshalIndent(bundles, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal bundles: %w", err)
	}

	err = os.WriteFile(filename, data, 0644)
	if err != nil {
		return fmt.Errorf("failed to write file: %w", err)
	}

	return nil
}

func LatestBlock(ctx context.Context, client *ethclient.Client) (*big.Int, error) {
	header, err := client.HeaderByNumber(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to get latest block: %w", err)
	}
	return header.Number, nil
}
