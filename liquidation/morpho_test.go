package liquidation_test

import (
	"encoding/json"
	"fmt"
	"math/big"
	"os"
	"testing"
	"toolkit/liquidation"
	"toolkit/query"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
)

func TestMorpho(t *testing.T) {
	logs, err := liquidation.MorphoLiquidations(ctx, client, big.NewInt(21525614), big.NewInt(24957375))
	if err != nil {
		t.Fatal(err)
	}

	t.Logf("Found %d logs", len(logs))

	// Save logs to JSON file
	filename := "morpho_logs.json"
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

func TestIdToMarketParams(t *testing.T) {
	logItem, err := liquidation.MorphoLiquidations(ctx, client, big.NewInt(18150811), big.NewInt(18150811))
	if err != nil {
		t.Fatal(err)
	}
	marketParams, err := liquidation.IdToMarketParams(ctx, client, logItem[0].Topics[1], logItem[0].BlockNumber)
	if err != nil {
		t.Fatal(err)
	}
	t.Log(marketParams.CollateralToken)
	t.Log(marketParams.LoanToken)
	t.Log(marketParams.Oracle)
	t.Log(marketParams.Irm)
	t.Log(marketParams.Lltv)
}

func TestCollateralToLoanPrice(t *testing.T) {
	price, err := liquidation.CollateralToLoanPrice(ctx, client, common.HexToAddress("0xf30BBFdab26B15285A303048b97A7910Fa252db5"), 18150811)
	if err != nil {
		t.Fatal(err)
	}
	t.Log(price)
}

func TestDecimals(t *testing.T) {
	decimals, err := liquidation.Decimals(ctx, client, common.HexToAddress("0x9151434b16b9763660705744891fA906F660EcC5"), 0)
	if err != nil {
		t.Fatal(err)
	}
	t.Log(decimals)
}

func TestParseMorphoLiquidationBonus(t *testing.T) {
	blockNumber := big.NewInt(21569283)
	logItem, err := liquidation.MorphoLiquidations(ctx, client, blockNumber, blockNumber)
	if err != nil {
		t.Fatal(err)
	}
	bonus, err := liquidation.ParseMorphoLiquidationBonus(ctx, client, logItem[0])
	if err != nil {
		t.Fatal(err)
	}
	name, err := liquidation.TokenName(ctx, client, bonus.LoanToken, 0)
	if err != nil {
		t.Fatal(err)
	}
	bonus.Print()
	ethValue, err := query.TokenToEthValue(bonus.LoanToken, bonus.Bonus(), blockNumber, client)
	if err != nil {
		t.Fatal(err)
	}
	fmt.Printf("%s revenue: %d\n", bonus.LoanToken, ethValue)
	t.Log(bonus.Bonus(), name)
	t.Log(bonus.BonusRate())
}

func TestMorphoStatistic(t *testing.T) {
	data, err := os.ReadFile("data/morpho_logs.json")
	if err != nil {
		t.Fatal(err)
	}
	var logs []types.Log
	err = json.Unmarshal(data, &logs)
	if err != nil {
		t.Fatal(err)
	}

	marketBonus := make(map[common.Hash]*liquidation.LiquidationBonus)
	for _, log := range logs {
		fmt.Println("--------------------------------")
		bonus, err := liquidation.ParseMorphoLiquidationBonus(ctx, client, log)
		if err != nil {
			t.Fatal(err)
		}
		ethValue, err := query.TokenToEthValue(bonus.LoanToken, bonus.Bonus(), big.NewInt(int64(log.BlockNumber)), client)
		if err != nil {
			t.Fatal(err)
		}
		fmt.Printf("%s revenue: %d\n", bonus.LoanToken, ethValue)

		if _, ok := marketBonus[bonus.MarketId]; !ok {
			marketBonus[bonus.MarketId] = &bonus
		} else {
			marketBonus[bonus.MarketId].LoanAmount.Add(marketBonus[bonus.MarketId].LoanAmount, bonus.LoanAmount)
			marketBonus[bonus.MarketId].SeizedValue.Add(marketBonus[bonus.MarketId].SeizedValue, bonus.SeizedValue)
		}
	}

	for marketId, bonus := range marketBonus {
		name, err := liquidation.TokenName(ctx, client, bonus.LoanToken, 0)
		if err != nil {
			t.Fatal(err)
		}
		decimals, err := liquidation.Decimals(ctx, client, bonus.LoanToken, 0)
		if err != nil {
			t.Fatal(err)
		}
		divisor := new(big.Int).Exp(big.NewInt(10), big.NewInt(int64(decimals)), nil)
		amount := new(big.Float).Quo(new(big.Float).SetInt(bonus.Bonus()), new(big.Float).SetInt(divisor))
		t.Log("================================================")
		t.Logf("Market ID: %s", marketId)
		t.Logf("Revenue in load token: %.6f %s", amount, name)
		// t.Logf("Revenue in load token: %v %s", bonus.Bonus(), name)
		t.Logf("Liquidation Bonus: %v", bonus.BonusRate())
	}
	t.Log("================================================")
}

func TestMorphoProfits(t *testing.T) {
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

		profit, err := liquidation.ParseMorphoTxProfit(ctx, client, txHash)
		if err != nil {
			t.Logf("Failed to parse profit: %v", err)
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
	}

	fmt.Println("--------------------------------")
	for contract, profit := range profitByContract {
		fmt.Printf("Contract: %v, Profit: %.4f ETH\n", contract, new(big.Float).Quo(new(big.Float).SetInt(profit), big.NewFloat(1e18)))
	}

	fmt.Printf("Total Profit: %.4f ETH\n", new(big.Float).Quo(new(big.Float).SetInt(totalProfit), big.NewFloat(1e18)))
}

func TestMorphoDataUpdate(t *testing.T) {
	// 1. Load data from mainnet_euler_logs.json
	data, err := os.ReadFile("data/morpho_logs.json")
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
		MarketId         common.Hash    `json:"marketId"`
	}

	logsWithRevenue := make([]LogWithRevenue, 0, len(logs))

	for i, log := range logs {
		bonus, err := liquidation.ParseMorphoLiquidationBonus(ctx, client, log)
		if err != nil {
			t.Logf("Failed to parse bonus for log %d: %v", i, err)
			continue
		}
		ethValue, err := query.TokenToEthValue(bonus.LoanToken, bonus.Bonus(), big.NewInt(int64(log.BlockNumber)), client)
		if err != nil {
			t.Logf("Failed to parse revenue for log %d: %v", i, err)
			continue
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
			Revenue:          ethValue,
			MarketId:         bonus.MarketId,
		})

		if i%100 == 0 {
			t.Logf("Processed %d/%d logs", i+1, len(logs))
		}
	}

	// 4. Save the new data in JSON
	outputFilename := "data/morpho_logs_with_revenue.json"
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
