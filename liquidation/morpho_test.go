package liquidation_test

import (
	"encoding/json"
	"fmt"
	"math/big"
	"os"
	"sort"
	"testing"
	"toolkit/liquidation"
	"toolkit/query"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
)

func TestMorphoLiquidations(t *testing.T) {
	logs, err := liquidation.MorphoLiquidations(ctx, client, big.NewInt(20701950), big.NewInt(26171693))
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
	blockNumber := big.NewInt(23031979)
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
	data, err := os.ReadFile("data/mainnet_morpho_logs.json")
	if err != nil {
		t.Fatal(err)
	}
	var logs []types.Log
	err = json.Unmarshal(data, &logs)
	if err != nil {
		t.Fatal(err)
	}

	// filteredLogs := make([]types.Log, 0, len(logs))
	// for _, log := range logs {
	// 	if log.BlockTimestamp >= 1748736000 {
	// 		filteredLogs = append(filteredLogs, log)
	// 	}
	// }

	marketRevenue := make(map[common.Hash]*big.Int)
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

		if _, ok := marketRevenue[bonus.MarketId]; !ok {
			marketRevenue[bonus.MarketId] = ethValue
		} else {
			marketRevenue[bonus.MarketId].Add(marketRevenue[bonus.MarketId], ethValue)
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
		t.Logf("Revenue in ETH: %.6f ETH", new(big.Float).Quo(new(big.Float).SetInt(marketRevenue[marketId]), big.NewFloat(1e18)))
		// t.Logf("Revenue in load token: %v %s", bonus.Bonus(), name)
		t.Logf("Liquidation Bonus: %v", bonus.BonusRate())
	}
	t.Log("================================================")

	// Additional: sort markets by total ETH revenue (descending) and print in order
	type marketEntry struct {
		id      common.Hash
		revenue *big.Int
	}
	orderedMarkets := make([]marketEntry, 0, len(marketRevenue))
	for id, rev := range marketRevenue {
		orderedMarkets = append(orderedMarkets, marketEntry{id: id, revenue: rev})
	}
	sort.Slice(orderedMarkets, func(i, j int) bool {
		return orderedMarkets[i].revenue.Cmp(orderedMarkets[j].revenue) > 0
	})
	for _, m := range orderedMarkets {
		bonus := marketBonus[m.id]
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
		t.Logf("[Sorted] Market ID: %s", m.id)
		t.Logf("[Sorted] Revenue in load token: %.6f %s", amount, name)
		t.Logf("[Sorted] Revenue in ETH: %.6f ETH", new(big.Float).Quo(new(big.Float).SetInt(m.revenue), big.NewFloat(1e18)))
		t.Logf("[Sorted] Liquidation Bonus: %v", bonus.BonusRate())
	}
}

func TestMorphoTxProfit(t *testing.T) {
	txHash := common.HexToHash("0x68320c3da0f070f097350facd56588f40cf0b0dc170d34537ab1ee89209ee88a")
	profit, totalRevenue, err := liquidation.ParseMorphoTxProfit(ctx, client, txHash)
	if err != nil {
		t.Fatal(err)
	}
	margin := new(big.Float).Quo(new(big.Float).SetInt(totalRevenue), new(big.Float).SetInt(profit))
	fmt.Printf("Profit: %v\n", profit)
	fmt.Printf("Margin: %v\n", margin)
}

func TestMorphoContractProfit(t *testing.T) {
	data, err := os.ReadFile("data/mainnet_morpho_logs_with_revenue.json")
	if err != nil {
		t.Fatal(err)
	}
	type LogWithRevenue struct {
		Address          common.Address `json:"address"`
		Topics           []string       `json:"topics"`
		Data             []byte         `json:"data"`
		BlockNumber      uint64         `json:"blockNumber"`
		TransactionHash  common.Hash    `json:"transactionHash"`
		TransactionIndex uint           `json:"transactionIndex"`
		To               common.Address `json:"to"`
		BlockHash        common.Hash    `json:"blockHash"`
		BlockTimestamp   uint64         `json:"blockTimestamp"`
		Index            uint           `json:"logIndex"`
		Removed          bool           `json:"removed"`
		Revenue          *big.Int       `json:"revenue"`
		MarketId         common.Hash    `json:"marketId"`
	}
	var logs []LogWithRevenue
	err = json.Unmarshal(data, &logs)
	if err != nil {
		t.Fatal(err)
	}
	// filteredLogs := make([]LogWithRevenue, 0, len(logs))
	// for _, log := range logs {
	// 	if log.BlockTimestamp >= 1748736000 {
	// 		filteredLogs = append(filteredLogs, log)
	// 	}
	// }

	txHashes := make(map[common.Hash]bool)
	for _, log := range logs {
		if log.To == common.HexToAddress("0x00000000009e50a7ddb7a7b0e2ee6604fd120e49") {
			txHashes[log.TransactionHash] = true
		}
	}

	accumulatedProfit := big.NewInt(0)
	for txHash := range txHashes {
		fmt.Println("--------------------------------")
		fmt.Printf("txHash: %v\n", txHash)
		profit, totalRevenue, err := liquidation.ParseMorphoTxProfit(ctx, client, txHash)
		if err != nil {
			t.Fatal(err)
		}
		fmt.Printf("Profit: %v\n", profit)
		margin := new(big.Float).Quo(new(big.Float).SetInt(profit), new(big.Float).SetInt(totalRevenue))
		fmt.Printf("Margin: %v\n", margin)
		accumulatedProfit = new(big.Int).Add(accumulatedProfit, profit)
		fmt.Printf("Total Profit: %v\n", accumulatedProfit)
	}
}

func TestMorphoProfits(t *testing.T) {
	data, err := os.ReadFile("data/mainnet_morpho_logs.json")
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
		if log.BlockTimestamp >= 1748736000 {
			txHashes[log.TxHash] = true
		}
	}

	profitByContract := make(map[common.Address]*big.Int)
	revenueByContract := make(map[common.Address]*big.Int)
	totalProfit := big.NewInt(0)
	for txHash := range txHashes {
		fmt.Println("--------------------------------")
		fmt.Printf("txHash: %v\n", txHash)

		profit, revenue, err := liquidation.ParseMorphoTxProfit(ctx, client, txHash)
		if err != nil {
			t.Logf("Failed to parse profit: %v", err)
			continue
		}

		fmt.Printf("Profit: %v\n", profit)
		margin := new(big.Float).Quo(new(big.Float).SetInt(profit), new(big.Float).SetInt(revenue))
		fmt.Printf("Margin: %v\n", margin)
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

		if _, ok := revenueByContract[toAddress]; !ok {
			revenueByContract[toAddress] = revenue
		} else {
			revenueByContract[toAddress] = new(big.Int).Add(revenueByContract[toAddress], revenue)
		}
	}

	fmt.Println("--------------------------------")
	// Sort by profit descending before printing
	type contractProfit struct {
		addr   common.Address
		profit *big.Int
	}
	ordered := make([]contractProfit, 0, len(profitByContract))
	for addr, profit := range profitByContract {
		ordered = append(ordered, contractProfit{addr: addr, profit: profit})
	}
	sort.Slice(ordered, func(i, j int) bool { return ordered[i].profit.Cmp(ordered[j].profit) > 0 })
	for _, cp := range ordered {
		margin := new(big.Float).Quo(new(big.Float).SetInt(cp.profit), new(big.Float).SetInt(revenueByContract[cp.addr]))
		fmt.Printf("Contract: %v, Profit: %.4f ETH, Margin: %v\n", cp.addr, new(big.Float).Quo(new(big.Float).SetInt(cp.profit), big.NewFloat(1e18)), margin)
	}

	fmt.Printf("Total Profit: %.4f ETH\n", new(big.Float).Quo(new(big.Float).SetInt(totalProfit), big.NewFloat(1e18)))
}

func TestAugmentMorphoLogs(t *testing.T) {
	data, err := os.ReadFile("data/unichainOEV_morpho_logs.json")
	if err != nil {
		t.Fatal(err)
	}

	var logs []types.Log
	err = json.Unmarshal(data, &logs)
	if err != nil {
		t.Fatal(err)
	}

	t.Logf("Processing %d logs", len(logs))

	type AugmentedLog struct {
		Address          common.Address `json:"address"`
		Topics           []string       `json:"topics"`
		Data             []byte         `json:"data"`
		BlockNumber      uint64         `json:"blockNumber"`
		TransactionHash  common.Hash    `json:"transactionHash"`
		TransactionIndex uint           `json:"transactionIndex"`
		To               common.Address `json:"to"`
		BlockHash        common.Hash    `json:"blockHash"`
		BlockTimestamp   uint64         `json:"blockTimestamp"`
		Index            uint           `json:"logIndex"`
		Removed          bool           `json:"removed"`
		Revenue          *big.Int       `json:"revenue"`
		MarketId         common.Hash    `json:"marketId"`
	}

	logsWithRevenue := make([]AugmentedLog, 0, len(logs))

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
		tx, _, err := client.TransactionByHash(ctx, log.TxHash)
		if err != nil {
			t.Log(err)
			continue
		}
		toAddress := *tx.To()

		logsWithRevenue = append(logsWithRevenue, AugmentedLog{
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
			To:               toAddress,
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

	outputFilename := "data/unichainOEV_augmented_morpho_logs.json"
	outputData, err := json.MarshalIndent(logsWithRevenue, "", "  ")
	if err != nil {
		t.Fatal(err)
	}

	err = os.WriteFile(outputFilename, outputData, 0644)
	if err != nil {
		t.Fatal(err)
	}

	t.Logf("augmented logs saved to %s", outputFilename)
}
