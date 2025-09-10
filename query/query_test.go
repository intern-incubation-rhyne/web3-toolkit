package query_test

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"math/big"
	"os"
	"path/filepath"
	"testing"
	"toolkit/config"
	"toolkit/liquidation"
	"toolkit/query"

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

type AugmentedBundle struct {
	Transactions []*types.Transaction `json:"transactions"`
	Revenue      *big.Int             `json:"revenue"`
	Profit       *big.Int             `json:"profit"`
	Margin       *big.Float           `json:"margin"`
}

func TestSearchBundle(t *testing.T) {
	eventSignatures := []string{
		"0xc797025feeeaf2cd924c99e9205acb8ec04d5cad21c41ce637a38fb6dee6016a",
		"0x1547a878dc89ad3c367b6338b4be6a65a5dd74fb77ae044da1e8747ef1f4f62f",
	}
	addresses := []common.Address{
		common.HexToAddress("0x7d4E742018fb52E48b08BE73d041C18B21de6Fb5"),
		{},
	}

	// startBlock := big.NewInt(21525614)
	// endBlock := big.NewInt(23282386)

	startBlock := big.NewInt(22747180)
	endBlock := big.NewInt(22747195)

	config := query.BundleSearchConfig{
		EventSignatures: eventSignatures,
		Addresses:       addresses,
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

	fmt.Printf("Found %d bundles\n", len(bundleTxs))

	query.SaveBundlesToFile(bundleTxs, "data/test_bundles.json")

	if len(bundleTxs) == 0 {
		t.Log("No bundle found - this might be normal for the given block range and signatures")
	}
}

func TestAugmentMorphoBundle(t *testing.T) {
	// Load existing bundles from test_bundles.json
	bundleData, err := os.ReadFile("data/test_bundles.json")
	if err != nil {
		t.Fatalf("Failed to read test_bundles.json: %v", err)
	}

	var bundleTxs [][]*types.Transaction
	err = json.Unmarshal(bundleData, &bundleTxs)
	if err != nil {
		t.Fatalf("Failed to unmarshal bundles: %v", err)
	}

	fmt.Printf("Loaded %d bundles from test_bundles.json\n", len(bundleTxs))

	// Augment each bundle with revenue, profit, and margin data
	var augmentedBundles []AugmentedBundle
	for _, bundle := range bundleTxs {
		if len(bundle) < 2 {
			continue // Skip incomplete bundles
		}

		// Get profit data from the second transaction (Morpho liquidation)
		profit, totalRevenue, err := liquidation.ParseMorphoTxProfit(ctx, client, bundle[1].Hash())
		if err != nil {
			fmt.Printf("Failed to parse profit for tx %s: %v\n", bundle[1].Hash().Hex(), err)
			continue
		}

		// Calculate margin
		margin := new(big.Float).Quo(new(big.Float).SetInt(totalRevenue), new(big.Float).SetInt(profit))

		augmentedBundles = append(augmentedBundles, AugmentedBundle{
			Transactions: bundle,
			Revenue:      totalRevenue,
			Profit:       profit,
			Margin:       margin,
		})

		fmt.Printf("Bundle with tx %s:\n", bundle[1].Hash().Hex())
		fmt.Printf("  Revenue: %v\n", totalRevenue)
		fmt.Printf("  Profit: %v\n", profit)
		fmt.Printf("  Margin: %v\n", margin)
	}

	// Save augmented bundles to a new file
	outputFile := "data/augmented_morpho_bundles.json"
	data, err := json.MarshalIndent(augmentedBundles, "", "  ")
	if err != nil {
		t.Fatalf("Failed to marshal augmented bundles: %v", err)
	}

	// Create directory if it doesn't exist
	dir := filepath.Dir(outputFile)
	if dir != "." {
		err := os.MkdirAll(dir, 0755)
		if err != nil {
			t.Fatalf("Failed to create directory: %v", err)
		}
	}

	err = os.WriteFile(outputFile, data, 0644)
	if err != nil {
		t.Fatalf("Failed to write augmented bundles: %v", err)
	}

	fmt.Printf("Saved %d augmented bundles to %s\n", len(augmentedBundles), outputFile)
}
