package liquidation_test

import (
	"encoding/json"
	"math/big"
	"os"
	"testing"
	"toolkit/liquidation"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
)

func TestVenusLiquidations(t *testing.T) {
	logs, err := liquidation.VenusLiquidations(ctx, client, big.NewInt(20701950), big.NewInt(26171693))
	if err != nil {
		t.Fatal(err)
	}

	t.Logf("Found %d logs", len(logs))

	// Save logs to JSON file
	filename := "venus_logs.json"
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

func TestAugmentVenusLogs(t *testing.T) {
	data, err := os.ReadFile("data/venus_logs.json")
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
	}

	augmentedLogs := make([]AugmentedLog, 0, len(logs))
	for _, log := range logs {
		tx, _, err := client.TransactionByHash(ctx, log.TxHash)
		if err != nil {
			t.Log(err)
			continue
		}
		toAddress := *tx.To()

		augmentedLogs = append(augmentedLogs, AugmentedLog{
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
		})
	}

	outputFilename := "data/augmented_venus_logs.json"
	outputData, err := json.MarshalIndent(augmentedLogs, "", "  ")
	if err != nil {
		t.Fatal(err)
	}

	err = os.WriteFile(outputFilename, outputData, 0644)
	if err != nil {
		t.Fatal(err)
	}

	t.Logf("augmented logs saved to %s", outputFilename)
}
