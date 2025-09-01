package query_test

import (
	"context"
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
