package filterlogs_test

import (
	"context"
	"log"
	"math/big"
	"os"
	"sort"
	"testing"
	"time"

	"toolkit/filterlogs"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/joho/godotenv"
)

var (
	ctx       context.Context
	arbClient *ethclient.Client
	uniClient *ethclient.Client
)

func init() {
	err := godotenv.Load("../.env")
	if err != nil {
		log.Fatal("Error loading .env file")
	}

	ctx = context.Background()

	arbClient, err = ethclient.Dial(os.Getenv("ARB_RPC_URL_ARCHIVE"))
	if err != nil {
		log.Fatal(err)
	}

	uniClient, err = ethclient.Dial(os.Getenv("UNICHAIN_RPC_URL"))
	if err != nil {
		log.Fatal(err)
	}
}

func TestFilterLogs(t *testing.T) {
	logs, err := arbClient.FilterLogs(ctx, ethereum.FilterQuery{
		FromBlock: big.NewInt(365917900),
		ToBlock:   big.NewInt(365927900),
		Addresses: []common.Address{common.HexToAddress("0x5fcb496a31b7AE91e7c9078Ec662bd7A55cd3079")},
		Topics: [][]common.Hash{
			{common.HexToHash(filterlogs.SetExpressLaneControllerSig)},
		},
	})
	if err != nil {
		t.Fatal(err)
	}

	for _, log := range logs {
		event, err := filterlogs.ParseSetExpressLaneControllerEvent(log)
		if err != nil {
			t.Logf("Failed to parse log: %v", err)
			t.Logf("Raw log: %+v", log)
			continue
		}

		t.Log(event.PrettyPrint())
	}
}

func TestFilterAuctionResolvedLogs(t *testing.T) {
	logs, err := arbClient.FilterLogs(ctx, ethereum.FilterQuery{
		FromBlock: big.NewInt(349800062),
		ToBlock:   big.NewInt(349801062),
		Addresses: []common.Address{common.HexToAddress("0x5fcb496a31b7AE91e7c9078Ec662bd7A55cd3079")},
		Topics: [][]common.Hash{
			{common.HexToHash(filterlogs.AuctionResolvedSig)},
		},
	})
	if err != nil {
		t.Fatal(err)
	}

	t.Logf("Found %d AuctionResolved events", len(logs))

	for _, log := range logs {
		event, err := filterlogs.ParseAuctionResolvedEvent(log)
		if err != nil {
			t.Logf("Failed to parse log: %v", err)
			t.Logf("Raw log: %+v", log)
			continue
		}

		t.Log(event.PrettyPrint())
	}
}

func TestAuctionResolved(t *testing.T) {
	// Test with a large block range that would exceed the 10,000 block limit
	fromBlock := big.NewInt(366696036)
	toBlock := big.NewInt(369446153)

	startTime := time.Now()

	// Use paginated filtering with 3 concurrent workers
	events, err := filterlogs.FilterAuctionResolvedLogs(ctx, arbClient, fromBlock, toBlock, 3)
	if err != nil {
		t.Fatal(err)
	}

	duration := time.Since(startTime)

	t.Logf("Paginated AuctionResolved query completed in %v", duration)
	t.Logf("Found %d AuctionResolved events across %d blocks (%d to %d)", len(events), new(big.Int).Sub(toBlock, fromBlock), fromBlock, toBlock)
	t.Logf("--------------------------------")

	maxRound := events[0].Round
	minRound := events[0].Round
	totalPrice := big.NewInt(0)

	controllerDistributions := make(map[common.Address]int)
	for _, event := range events {
		controllerDistributions[event.FirstPriceBidder]++
		if event.Round > maxRound {
			maxRound = event.Round
		}
		if event.Round < minRound {
			minRound = event.Round
		}
		totalPrice.Add(totalPrice, event.Price)
	}

	// Create a slice to sort by count
	type controllerCount struct {
		controller common.Address
		count      int
	}

	var sortedControllers []controllerCount
	for controller, count := range controllerDistributions {
		sortedControllers = append(sortedControllers, controllerCount{controller, count})
	}

	// Sort by count in descending order
	sort.Slice(sortedControllers, func(i, j int) bool {
		return sortedControllers[i].count > sortedControllers[j].count
	})

	t.Logf("Controller distribution (sorted by count):")
	for _, item := range sortedControllers {
		t.Logf("Controller %s: %d rounds", item.controller.Hex(), item.count)
	}

	t.Logf("--------------------------------")
	t.Logf("Average deal price: %s", totalPrice.Div(totalPrice, big.NewInt(int64(len(events)))))

	t.Logf("--------------------------------")
	t.Logf("Round range: %d - %d", minRound, maxRound)
	t.Logf("Total rounds: %d", maxRound-minRound+1)
	t.Logf("Bought rounds: %d", len(events))
	t.Logf("Round occupancy rate: %.2f%%", float64(len(events))/float64(maxRound-minRound+1)*100)
}

func TestPaginatedSetExpressLaneControllerLogs(t *testing.T) {
	// Test with a large block range
	fromBlock := big.NewInt(366696036)
	toBlock := big.NewInt(369446153)

	startTime := time.Now()

	// Use paginated filtering with 5 concurrent workers
	events, err := filterlogs.FilterSetExpressLaneControllerLogs(ctx, arbClient, fromBlock, toBlock, 5)
	if err != nil {
		t.Fatal(err)
	}

	duration := time.Since(startTime)

	t.Logf("Paginated SetExpressLaneController query completed in %v", duration)
	t.Logf("Found %d SetExpressLaneController events across %d blocks", len(events), new(big.Int).Sub(toBlock, fromBlock))
	t.Logf("--------------------------------")

	maxRound := events[0].Round
	minRound := events[0].Round

	controllerDistributions := make(map[common.Address]int)
	for _, event := range events {
		controllerDistributions[event.NewExpressLaneController]++
		if event.Round > maxRound {
			maxRound = event.Round
		}
		if event.Round < minRound {
			minRound = event.Round
		}
	}

	// Create a slice to sort by count
	type controllerCount struct {
		controller common.Address
		count      int
	}

	var sortedControllers []controllerCount
	for controller, count := range controllerDistributions {
		sortedControllers = append(sortedControllers, controllerCount{controller, count})
	}

	// Sort by count in descending order
	sort.Slice(sortedControllers, func(i, j int) bool {
		return sortedControllers[i].count > sortedControllers[j].count
	})

	t.Logf("Controller distribution (sorted by count):")
	for _, item := range sortedControllers {
		t.Logf("Controller %s: %d rounds", item.controller.Hex(), item.count)
	}

	t.Logf("--------------------------------")
	t.Logf("Round range: %d - %d", minRound, maxRound)
	t.Logf("Total rounds: %d", maxRound-minRound+1)
	t.Logf("Bought rounds: %d", len(events))
	t.Logf("Round occupancy rate: %.2f%%", float64(len(events))/float64(maxRound-minRound+1)*100)
}
