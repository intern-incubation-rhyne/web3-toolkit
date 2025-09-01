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

func TestMorpho(t *testing.T) {
	logs, err := liquidation.MorphoLiquidations(ctx, client, big.NewInt(9139027), big.NewInt(24957375))
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
	logItem, err := liquidation.MorphoLiquidations(ctx, client, big.NewInt(18150811), big.NewInt(18150811))
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
		bonus, err := liquidation.ParseMorphoLiquidationBonus(ctx, client, log)
		if err != nil {
			t.Fatal(err)
		}
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
