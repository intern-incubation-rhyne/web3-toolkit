package liquidation_test

import (
	"context"
	"log"
	"math/big"
	"os"
	"testing"
	"toolkit/liquidation"

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

	client, err = ethclient.Dial(os.Getenv("UNICHAIN_RPC_URL"))
	if err != nil {
		log.Fatalf("Error connecting RPC: %v", err)
	}
}

func TestEVKLiquidations(t *testing.T) {
	logs, err := liquidation.EVKLiquidations(ctx, client, big.NewInt(24957375), big.NewInt(24957375))
	if err != nil {
		t.Fatal(err)
	}

	t.Logf("Found %d logs", len(logs))
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
	logs, err := liquidation.EVKLiquidations(ctx, client, big.NewInt(19850630), big.NewInt(19850630))
	if err != nil {
		t.Fatal(err)
	}

	revenue, err := liquidation.ParseEVKLiquidationRevenue(ctx, client, logs[0])
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
	logs, err := liquidation.EVKLiquidations(ctx, client, big.NewInt(19850630), big.NewInt(19850630))
	if err != nil {
		t.Fatal(err)
	}

	profit, err := liquidation.ParseEVKLiquidationProfit(ctx, client, logs[0])
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("Profit: %v", profit)
}

func TestStatistic(t *testing.T) {
	logs, err := liquidation.EVKLiquidations(ctx, client, big.NewInt(15408102), big.NewInt(24957375))
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("analyzing %d txns", len(logs))

	profitByContract := make(map[common.Address]*big.Int)
	totalProfit := big.NewInt(0)
	for _, log := range logs {
		profit, err := liquidation.ParseEVKLiquidationProfit(ctx, client, log)
		t.Logf("txHash: %v, Profit: %v", log.TxHash, profit)
		if err != nil {
			t.Fatal(err)
		}
		totalProfit = new(big.Int).Add(totalProfit, profit)
		if _, ok := profitByContract[log.Address]; !ok {
			profitByContract[log.Address] = profit
		} else {
			profitByContract[log.Address] = new(big.Int).Add(profitByContract[log.Address], profit)
		}
	}

	for contract, profit := range profitByContract {
		t.Logf("Contract: %v, Profit: %v", contract, profit)
	}

	t.Logf("Total Profit: %v", totalProfit)
}
