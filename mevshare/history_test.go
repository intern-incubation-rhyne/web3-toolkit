package mevshare_test

import (
	"context"
	"fmt"
	"log"
	"math/big"
	"os"
	"testing"
	"toolkit/mevshare"

	"github.com/joho/godotenv"
)

var (
	ctx    context.Context
	client *mevshare.Client
)

func init() {
	err := godotenv.Load("../.env")
	if err != nil {
		log.Fatal("Error loading .env file")
	}

	ctx = context.Background()
	client = mevshare.NewClient(mevshare.Mainnet)
}

func TestHistoryByBlock(t *testing.T) {
	blockNumber := big.NewInt(23031978)

	resp, err := client.HistoryByBlock(ctx, blockNumber)
	if err != nil {
		t.Fatalf("HistoryByBlock failed: %v", err)
	}

	err = mevshare.SaveToFile(resp, fmt.Sprintf("history_%s.json", blockNumber.String()))
	if err != nil {
		t.Fatalf("SaveToFile failed: %v", err)
	}
}

func TestHistoryByBlockRange(t *testing.T) {
	blockStart := big.NewInt(23142045)
	blockEnd := big.NewInt(23142052)

	resp, err := client.HistoryByBlockRange(ctx, blockStart, blockEnd)
	if err != nil {
		t.Fatalf("HistoryByBlockRange failed: %v", err)
	}

	err = mevshare.SaveToFile(resp, fmt.Sprintf("data/history_%s_%s.json", blockStart.String(), blockEnd.String()))
	if err != nil {
		t.Fatalf("SaveToFile failed: %v", err)
	}
}

func TestLatestHistory(t *testing.T) {
	latestBlock, resp, err := client.FetchLatestHistory(ctx, os.Getenv("MAINNET_RPC_URL"))
	if err != nil {
		t.Fatalf("LatestHistory failed: %v", err)
	}

	err = mevshare.SaveToFile(resp, fmt.Sprintf("data/latest_history_%s.json", latestBlock.String()))
	if err != nil {
		t.Fatalf("SaveToFile %s failed: %v", latestBlock.String(), err)
	}
}
