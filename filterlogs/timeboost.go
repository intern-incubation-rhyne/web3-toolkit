package filterlogs

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

const (
	SetExpressLaneControllerSig = "0xb59adc820ca642dad493a0a6e0bdf979dcae037dea114b70d5c66b1c0b791c4b"
	AuctionResolvedSig          = "0x7f5bdabbd27a8fc572781b177055488d7c6729a2bade4f57da9d200f31c15d47"
	AuctionContractGenesis      = 314529691
	MaxBlockRange               = 10000 // Ethereum FilterLogs limit
)

// FilterLogsQuery represents a query configuration for filtering logs
type FilterLogsQuery struct {
	FromBlock  *big.Int
	ToBlock    *big.Int
	Addresses  []common.Address
	Topics     [][]common.Hash
	MaxWorkers int // Number of concurrent workers
}

// FilterLogsResult represents the result of a paginated filter logs query
type FilterLogsResult struct {
	Logs []types.Log
	Err  error
}

// PaginatedFilterLogs performs concurrent paginated queries to get logs across a large block range
func PaginatedFilterLogs(ctx context.Context, client *ethclient.Client, query FilterLogsQuery) ([]types.Log, error) {
	if query.MaxWorkers <= 0 {
		query.MaxWorkers = 5 // Default to 5 concurrent workers
	}

	// Calculate the number of chunks needed
	blockRange := new(big.Int).Sub(query.ToBlock, query.FromBlock)
	if blockRange.Cmp(big.NewInt(MaxBlockRange)) <= 0 {
		// If range is within limit, do a single query
		return client.FilterLogs(ctx, ethereum.FilterQuery{
			FromBlock: query.FromBlock,
			ToBlock:   query.ToBlock,
			Addresses: query.Addresses,
			Topics:    query.Topics,
		})
	}

	// Create channels for results and work
	resultChan := make(chan FilterLogsResult, query.MaxWorkers)
	workChan := make(chan []*big.Int, query.MaxWorkers)

	// Start worker goroutines
	var wg sync.WaitGroup
	for i := 0; i < query.MaxWorkers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for work := range workChan {
				if len(work) != 2 {
					continue
				}
				fromBlock, toBlock := work[0], work[1]

				logs, err := client.FilterLogs(ctx, ethereum.FilterQuery{
					FromBlock: fromBlock,
					ToBlock:   toBlock,
					Addresses: query.Addresses,
					Topics:    query.Topics,
				})

				resultChan <- FilterLogsResult{Logs: logs, Err: err}
			}
		}()
	}

	// Generate work chunks
	go func() {
		defer close(workChan)
		currentBlock := new(big.Int).Set(query.FromBlock)

		for currentBlock.Cmp(query.ToBlock) < 0 {
			chunkEnd := new(big.Int).Add(currentBlock, big.NewInt(MaxBlockRange-1))
			if chunkEnd.Cmp(query.ToBlock) > 0 {
				chunkEnd.Set(query.ToBlock)
			}

			workChan <- []*big.Int{new(big.Int).Set(currentBlock), chunkEnd}
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
	for result := range resultChan {
		if result.Err != nil {
			return nil, fmt.Errorf("error in paginated query: %w", result.Err)
		}
		allLogs = append(allLogs, result.Logs...)
	}

	return allLogs, nil
}

// FilterSetExpressLaneControllerLogs filters and parses SetExpressLaneController events
func FilterSetExpressLaneControllerLogs(ctx context.Context, client *ethclient.Client, fromBlock, toBlock *big.Int, maxWorkers int) ([]*SetExpressLaneControllerEvent, error) {
	query := FilterLogsQuery{
		FromBlock:  fromBlock,
		ToBlock:    toBlock,
		Addresses:  []common.Address{common.HexToAddress("0x5fcb496a31b7AE91e7c9078Ec662bd7A55cd3079")},
		Topics:     [][]common.Hash{{common.HexToHash(SetExpressLaneControllerSig)}},
		MaxWorkers: maxWorkers,
	}

	logs, err := PaginatedFilterLogs(ctx, client, query)
	if err != nil {
		return nil, err
	}

	var events []*SetExpressLaneControllerEvent
	for _, log := range logs {
		event, err := ParseSetExpressLaneControllerEvent(log)
		if err != nil {
			continue // Skip logs that can't be parsed
		}
		events = append(events, event)
	}

	return events, nil
}

// FilterAuctionResolvedLogs filters and parses AuctionResolved events
func FilterAuctionResolvedLogs(ctx context.Context, client *ethclient.Client, fromBlock, toBlock *big.Int, maxWorkers int) ([]*AuctionResolvedEvent, error) {
	query := FilterLogsQuery{
		FromBlock:  fromBlock,
		ToBlock:    toBlock,
		Addresses:  []common.Address{common.HexToAddress("0x5fcb496a31b7AE91e7c9078Ec662bd7A55cd3079")},
		Topics:     [][]common.Hash{{common.HexToHash(AuctionResolvedSig)}},
		MaxWorkers: maxWorkers,
	}

	logs, err := PaginatedFilterLogs(ctx, client, query)
	if err != nil {
		return nil, err
	}

	var events []*AuctionResolvedEvent
	for _, log := range logs {
		event, err := ParseAuctionResolvedEvent(log)
		if err != nil {
			continue // Skip logs that can't be parsed
		}
		events = append(events, event)
	}

	return events, nil
}

// SetExpressLaneControllerEvent represents the parsed SetExpressLaneController event
type SetExpressLaneControllerEvent struct {
	Round                         uint64         `json:"round"`
	PreviousExpressLaneController common.Address `json:"previousExpressLaneController"`
	NewExpressLaneController      common.Address `json:"newExpressLaneController"`
	Transferor                    common.Address `json:"transferor"`
	StartTimestamp                uint64         `json:"startTimestamp"`
	EndTimestamp                  uint64         `json:"endTimestamp"`
	BlockNumber                   uint64         `json:"blockNumber"`
	TransactionHash               common.Hash    `json:"transactionHash"`
	LogIndex                      uint           `json:"logIndex"`
}

// AuctionResolvedEvent represents the parsed AuctionResolved event
type AuctionResolvedEvent struct {
	IsMultiBidAuction               bool           `json:"isMultiBidAuction"`
	Round                           uint64         `json:"round"`
	FirstPriceBidder                common.Address `json:"firstPriceBidder"`
	FirstPriceExpressLaneController common.Address `json:"firstPriceExpressLaneController"`
	FirstPriceAmount                *big.Int       `json:"firstPriceAmount"`
	Price                           *big.Int       `json:"price"`
	RoundStartTimestamp             uint64         `json:"roundStartTimestamp"`
	RoundEndTimestamp               uint64         `json:"roundEndTimestamp"`
	BlockNumber                     uint64         `json:"blockNumber"`
	TransactionHash                 common.Hash    `json:"transactionHash"`
	LogIndex                        uint           `json:"logIndex"`
}

// ParseSetExpressLaneControllerEvent parses a log into SetExpressLaneControllerEvent
func ParseSetExpressLaneControllerEvent(log types.Log) (*SetExpressLaneControllerEvent, error) {
	if len(log.Topics) < 4 {
		return nil, fmt.Errorf("insufficient topics for SetExpressLaneController event")
	}

	// Topics[0] is the event signature
	// Topics[1] is previousExpressLaneController (indexed)
	// Topics[2] is newExpressLaneController (indexed)
	// Topics[3] is transferor (indexed)

	// Data contains: round, startTimestamp, endTimestamp
	if len(log.Data) < 96 { // 3 * 32 bytes
		return nil, fmt.Errorf("insufficient data for SetExpressLaneController event")
	}

	event := &SetExpressLaneControllerEvent{
		PreviousExpressLaneController: common.BytesToAddress(log.Topics[1].Bytes()),
		NewExpressLaneController:      common.BytesToAddress(log.Topics[2].Bytes()),
		Transferor:                    common.BytesToAddress(log.Topics[3].Bytes()),
		BlockNumber:                   log.BlockNumber,
		TransactionHash:               log.TxHash,
		LogIndex:                      log.Index,
	}

	// Parse data fields
	event.Round = new(big.Int).SetBytes(log.Data[0:32]).Uint64()
	event.StartTimestamp = new(big.Int).SetBytes(log.Data[32:64]).Uint64()
	event.EndTimestamp = new(big.Int).SetBytes(log.Data[64:96]).Uint64()

	return event, nil
}

// ParseAuctionResolvedEvent parses a log into AuctionResolvedEvent
func ParseAuctionResolvedEvent(log types.Log) (*AuctionResolvedEvent, error) {
	if len(log.Topics) < 4 {
		return nil, fmt.Errorf("insufficient topics for AuctionResolved event")
	}

	// Topics[0] is the event signature
	// Topics[1] is isMultiBidAuction (indexed)
	// Topics[2] is firstPriceBidder (indexed)
	// Topics[3] is firstPriceExpressLaneController (indexed)

	// Data contains: round, firstPriceAmount, price, roundStartTimestamp, roundEndTimestamp
	if len(log.Data) < 160 { // 5 * 32 bytes
		return nil, fmt.Errorf("insufficient data for AuctionResolved event")
	}

	// Parse isMultiBidAuction from Topics[1] (bool is encoded as 32 bytes)
	isMultiBidAuction := new(big.Int).SetBytes(log.Topics[1].Bytes()).Uint64() != 0

	event := &AuctionResolvedEvent{
		IsMultiBidAuction:               isMultiBidAuction,
		FirstPriceBidder:                common.BytesToAddress(log.Topics[2].Bytes()),
		FirstPriceExpressLaneController: common.BytesToAddress(log.Topics[3].Bytes()),
		BlockNumber:                     log.BlockNumber,
		TransactionHash:                 log.TxHash,
		LogIndex:                        log.Index,
	}

	// Parse data fields
	event.Round = new(big.Int).SetBytes(log.Data[0:32]).Uint64()
	event.FirstPriceAmount = new(big.Int).SetBytes(log.Data[32:64])
	event.Price = new(big.Int).SetBytes(log.Data[64:96])
	event.RoundStartTimestamp = new(big.Int).SetBytes(log.Data[96:128]).Uint64()
	event.RoundEndTimestamp = new(big.Int).SetBytes(log.Data[128:160]).Uint64()

	return event, nil
}

// PrettyPrint formats the event for nice output
func (e *SetExpressLaneControllerEvent) PrettyPrint() string {
	startTime := time.Unix(int64(e.StartTimestamp), 0)
	endTime := time.Unix(int64(e.EndTimestamp), 0)

	return fmt.Sprintf(`
SetExpressLaneController Event:
├── Round: %d
├── Previous Express Lane Controller: %s
├── New Express Lane Controller: %s
├── Transferor: %s
├── Start Timestamp: %d (%s)
├── End Timestamp: %d (%s)
├── Block Number: %d
├── Transaction Hash: %s
└── Log Index: %d
`,
		e.Round,
		e.PreviousExpressLaneController.Hex(),
		e.NewExpressLaneController.Hex(),
		e.Transferor.Hex(),
		e.StartTimestamp, startTime.Format(time.RFC3339),
		e.EndTimestamp, endTime.Format(time.RFC3339),
		e.BlockNumber,
		e.TransactionHash.Hex(),
		e.LogIndex,
	)
}

// PrettyPrint formats the event for nice output
func (e *AuctionResolvedEvent) PrettyPrint() string {
	roundStartTime := time.Unix(int64(e.RoundStartTimestamp), 0)
	roundEndTime := time.Unix(int64(e.RoundEndTimestamp), 0)

	// Convert wei to ether for better readability
	firstPriceAmountEth := new(big.Float).Quo(new(big.Float).SetInt(e.FirstPriceAmount), new(big.Float).SetInt(new(big.Int).Exp(big.NewInt(10), big.NewInt(18), nil)))
	priceEth := new(big.Float).Quo(new(big.Float).SetInt(e.Price), new(big.Float).SetInt(new(big.Int).Exp(big.NewInt(10), big.NewInt(18), nil)))

	// Convert big.Float to float64 for formatting
	firstPriceAmountEthFloat, _ := firstPriceAmountEth.Float64()
	priceEthFloat, _ := priceEth.Float64()

	return fmt.Sprintf(`
AuctionResolved Event:
├── Is Multi Bid Auction: %t
├── Round: %d
├── First Price Bidder: %s
├── First Price Express Lane Controller: %s
├── First Price Amount: %s wei (%.6f ETH)
├── Price: %s wei (%.6f ETH)
├── Round Start Timestamp: %d (%s)
├── Round End Timestamp: %d (%s)
├── Block Number: %d
├── Transaction Hash: %s
└── Log Index: %d
`,
		e.IsMultiBidAuction,
		e.Round,
		e.FirstPriceBidder.Hex(),
		e.FirstPriceExpressLaneController.Hex(),
		e.FirstPriceAmount.String(), firstPriceAmountEthFloat,
		e.Price.String(), priceEthFloat,
		e.RoundStartTimestamp, roundStartTime.Format(time.RFC3339),
		e.RoundEndTimestamp, roundEndTime.Format(time.RFC3339),
		e.BlockNumber,
		e.TransactionHash.Hex(),
		e.LogIndex,
	)
}
