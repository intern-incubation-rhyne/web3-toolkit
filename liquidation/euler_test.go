package liquidation_test

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"math/big"
	"os"
	"testing"
	"time"
	"toolkit/liquidation"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/joho/godotenv"
)

var (
	ctx    context.Context
	client *ethclient.Client
)

func init() {
	err := godotenv.Load("../.env")
	if err != nil {
		log.Fatal("Error loading .env file")
	}

	ctx = context.Background()

	client, err = ethclient.Dial(os.Getenv("MAINNET_RPC_URL"))
	if err != nil {
		log.Fatalf("Error connecting RPC: %v", err)
	}
}

func TestEVKLiquidations(t *testing.T) {
	logs, err := liquidation.EVKLiquidations(ctx, client, big.NewInt(21525614), big.NewInt(23263156))
	if err != nil {
		t.Fatal(err)
	}

	t.Logf("Found %d logs", len(logs))

	// Save logs to JSON file
	filename := "data/mainnet_euler_logs.json"
	data, err := json.MarshalIndent(logs, "", "  ")
	if err != nil {
		t.Fatal(err)
	}

	err = os.WriteFile(filename, data, 0644)
	if err != nil {
		t.Fatal(err)
	}

	t.Logf("Logs saved to %s", filename)
}

func TestGetQuote(t *testing.T) {
	oracle := common.HexToAddress("0xdEb6135daed5470241843838944631Af12cE464B")
	base := common.HexToAddress("0xD49181c522eCDB265f0D9C175Cf26FFACE64eAD3")
	quote := common.HexToAddress("0x0000000000000000000000000000000000000348")
	amount := big.NewInt(129165286)
	blockNumber := uint64(21316108)
	txIndex := uint(2)
	value, err := liquidation.GetQuote(ctx, client, blockNumber, txIndex, &oracle, amount, &base, &quote)
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("USD Value: %v", value)
}

func TestUSDValue(t *testing.T) {
	evk := common.HexToAddress("0xD49181c522eCDB265f0D9C175Cf26FFACE64eAD3")
	amount := big.NewInt(129165286)
	blockNumber := uint64(21316108)
	txIndex := uint(2)
	value, err := liquidation.USDValue(ctx, client, blockNumber, txIndex, &evk, amount)
	if err != nil {
		t.Fatal(err)
	}

	t.Logf("USD Value: %v", value)
}

func TestEthPrice(t *testing.T) {
	blockNumber := uint64(21316108)
	price, err := liquidation.EthPrice(ctx, client, blockNumber)
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("ETH Price: %v", price)
}

func TestParseEVKLiquidationRevenue(t *testing.T) {
	logs, err := liquidation.EVKLiquidations(ctx, client, big.NewInt(22017846), big.NewInt(22017846))
	if err != nil {
		t.Fatal(err)
	}

	revenue, err := liquidation.ParseEVKLiquidationRevenue(ctx, os.Getenv("MAINNET_RPC_URL"), client, logs[0])
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("Revenue: %v", revenue)
}

func TestGetL1Fee(t *testing.T) {
	logs, err := liquidation.EVKLiquidations(ctx, client, big.NewInt(21316108), big.NewInt(21316108))
	if err != nil {
		t.Fatal(err)
	}

	gasPrice, err := liquidation.GetL1Fee(ctx, client, logs[0])
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("L1 Fee: %v", gasPrice)
}

func TestParseEVKLiquidationProfit(t *testing.T) {
	logs, err := liquidation.EVKLiquidations(ctx, client, big.NewInt(22734802), big.NewInt(22734802))
	if err != nil {
		t.Fatal(err)
	}

	profit, err := liquidation.ParseEVKLiquidationProfit(ctx, os.Getenv("MAINNET_RPC_URL"), client, logs[0].TxHash)
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("Profit: %v", profit)
}

func TestEulerStatistic(t *testing.T) {
	// logs, err := liquidation.EVKLiquidations(ctx, client, big.NewInt(15408102), big.NewInt(24957375))
	// if err != nil {
	// 	t.Fatal(err)
	// }
	// t.Logf("analyzing %d txns", len(logs))
	data, err := os.ReadFile("data/mainnet_euler_logs.json")
	if err != nil {
		t.Fatal(err)
	}
	var logs []types.Log
	err = json.Unmarshal(data, &logs)
	if err != nil {
		t.Fatal(err)
	}

	txHashes := make(map[common.Hash]bool)
	for _, log := range logs {
		txHashes[log.TxHash] = true
	}

	profitByContract := make(map[common.Address]*big.Int)
	totalProfit := big.NewInt(0)
	for txHash := range txHashes {
		fmt.Println("--------------------------------")
		fmt.Printf("txHash: %v\n", txHash)
		profit, err := liquidation.ParseEVKLiquidationProfit(ctx, os.Getenv("MAINNET_RPC_URL"), client, txHash)
		if err != nil {
			t.Log(err)
			continue
		}
		fmt.Printf("Profit: %v\n", profit)
		totalProfit = new(big.Int).Add(totalProfit, profit)

		tx, _, err := client.TransactionByHash(ctx, txHash)
		if err != nil {
			t.Log(err)
			continue
		}
		toAddress := *tx.To()

		if _, ok := profitByContract[toAddress]; !ok {
			profitByContract[toAddress] = profit
		} else {
			profitByContract[toAddress] = new(big.Int).Add(profitByContract[toAddress], profit)
		}
		time.Sleep(500 * time.Millisecond)
	}

	for contract, profit := range profitByContract {
		fmt.Printf("Contract: %v, Profit: %.4f ETH\n", contract, new(big.Float).Quo(new(big.Float).SetInt(profit), big.NewFloat(1e18)))
	}

	fmt.Printf("Total Profit: %.4f ETH\n", new(big.Float).Quo(new(big.Float).SetInt(totalProfit), big.NewFloat(1e18)))
}

func TestEulerLens(t *testing.T) {
	rpcUrl := os.Getenv("MAINNET_RPC_URL")
	debt := common.HexToAddress("0x797DD80692c3b2dAdabCe8e30C07fDE5307D48a9")
	debtAmount, _ := new(big.Int).SetString("193366001063", 10)
	collateral := common.HexToAddress("0xF6E2EfDF175e7a91c8847dade42f2d39A9aE57D4")
	collateralAmount, _ := new(big.Int).SetString("46591864627301414905", 10)
	blockNumber := big.NewInt(21573389)
	txIndex := uint(1)
	revenue, debtValue, collateralValue, err := liquidation.GetEulerRevenue(ctx, rpcUrl, client, debt, debtAmount, collateral, collateralAmount, blockNumber, txIndex)
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("Revenue: %v", revenue)
	t.Logf("Debt Value: %v", debtValue)
	t.Logf("Collateral Value: %v", collateralValue)
}

func TestEulerDataUpdate(t *testing.T) {
	// 1. Load data from mainnet_euler_logs.json
	data, err := os.ReadFile("data/mainnet_euler_logs.json")
	if err != nil {
		t.Fatal(err)
	}

	var logs []types.Log
	err = json.Unmarshal(data, &logs)
	if err != nil {
		t.Fatal(err)
	}

	t.Logf("Processing %d logs", len(logs))

	// 2. For each log, calculate its revenue with ParseEVKLiquidationRevenue
	// 3. Add the revenue back into the original data
	type LogWithRevenue struct {
		Address          common.Address `json:"address"`
		Topics           []string       `json:"topics"`
		Data             []byte         `json:"data"`
		BlockNumber      uint64         `json:"blockNumber"`
		TransactionHash  common.Hash    `json:"transactionHash"`
		TransactionIndex uint           `json:"transactionIndex"`
		BlockHash        common.Hash    `json:"blockHash"`
		BlockTimestamp   uint64         `json:"blockTimestamp"`
		Index            uint           `json:"logIndex"`
		Removed          bool           `json:"removed"`
		Revenue          *big.Int       `json:"revenue"`
	}

	logsWithRevenue := make([]LogWithRevenue, 0, len(logs))

	for i, log := range logs {
		revenue, err := liquidation.ParseEVKLiquidationRevenue(ctx, os.Getenv("MAINNET_RPC_URL"), client, log)
		if err != nil {
			t.Logf("Failed to parse revenue for log %d: %v", i, err)
			// Continue with zero revenue if parsing fails
			revenue = big.NewInt(0)
		}

		logsWithRevenue = append(logsWithRevenue, LogWithRevenue{
			Address: log.Address,
			Topics: func() []string {
				topics := make([]string, len(log.Topics))
				for j, topic := range log.Topics {
					topics[j] = topic.Hex()
				}
				return topics
			}(),
			Data:             log.Data,
			BlockNumber:      log.BlockNumber,
			TransactionHash:  log.TxHash,
			TransactionIndex: log.TxIndex,
			BlockHash:        log.BlockHash,
			BlockTimestamp:   log.BlockTimestamp,
			Index:            uint(log.Index),
			Removed:          log.Removed,
			Revenue:          revenue,
		})

		if i%100 == 0 {
			t.Logf("Processed %d/%d logs", i+1, len(logs))
		}
	}

	// 4. Save the new data in JSON
	outputFilename := "data/mainnet_euler_logs_with_revenue.json"
	outputData, err := json.MarshalIndent(logsWithRevenue, "", "  ")
	if err != nil {
		t.Fatal(err)
	}

	err = os.WriteFile(outputFilename, outputData, 0644)
	if err != nil {
		t.Fatal(err)
	}

	t.Logf("Updated data saved to %s", outputFilename)
}
