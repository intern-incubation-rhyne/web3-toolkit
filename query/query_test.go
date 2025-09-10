package query_test

import (
	"context"
	"fmt"
	"log"
	"math/big"
	"os"
	"testing"
	"toolkit/config"
	"toolkit/query"

	"github.com/ethereum/go-ethereum/common"
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

func TestGetRateToEth(t *testing.T) {
	client, err := ethclient.Dial(config.HTTP_RPC)
	if err != nil {
		log.Fatalf("Error connecting RPC: %v", err)
	}

	rate, err := query.GetTokenRateToETH(common.HexToAddress("0xC02aaA39b223FE8D0A0e5C4F27eAD9083C756Cc2"), nil, client)
	if err != nil {
		log.Fatalf("Error getting rate: %v", err)
	}
	t.Logf("rate: %v", rate)
}

func TestGetTokenTotalPriceToETH(t *testing.T) {
	amount, _ := new(big.Int).SetString("3682645216638265720832", 10)
	rate, err := query.TokenToEthValue(common.HexToAddress("0x90D2af7d622ca3141efA4d8f1F24d86E5974Cc8F"), amount, big.NewInt(22910856), client)
	if err != nil {
		log.Fatalf("Error getting rate: %v", err)
	}
	t.Logf("rate: %v", rate)
}

func TestBribe(t *testing.T) {
	bribe, err := query.Bribe(ctx, client, common.HexToHash("0xa673593c10da7394dc73b359ac30e086deeb6c685204158f879868c5baaea948"))
	if err != nil {
		log.Fatalf("Error getting bribe: %v", err)
	}
	t.Logf("bribe: %v", bribe)
}

func TestSearchBundle(t *testing.T) {
	eventSignatures := []string{
		"0xa7fc99ed7617309ee23f63ae90196a1e490d362e6f6a547a59bc809ee2291782",
		"0xa4946ede45d0c6f06a0f5ce92c9ad3b4751452d2fe0e25010783bcab57a67e41",
	}

	// startBlock := big.NewInt(21525614)
	// endBlock := big.NewInt(23282386)

	startBlock := big.NewInt(23031978)
	endBlock := big.NewInt(23031980)

	config := query.BundleSearchConfig{
		EventSignatures: eventSignatures,
		StartBlock:      startBlock,
		EndBlock:        endBlock,
		MaxWorkers:      2,
		OutputFile:      "test_bundles.json",
	}

	fmt.Printf("Searching for bundles from block %s to %s\n",
		startBlock.String(), endBlock.String())

	bundleTxs, err := query.SearchBundle(ctx, client, config)
	if err != nil {
		t.Fatalf("SearchBundle failed: %v", err)
	}

	fmt.Printf("Found %d bundle transactions\n", len(bundleTxs))

	query.SaveBundlesToFile(bundleTxs, "data/test_bundles.json")

	if len(bundleTxs) == 0 {
		t.Log("No bundle transactions found - this might be normal for the given block range and signatures")
	}
}
