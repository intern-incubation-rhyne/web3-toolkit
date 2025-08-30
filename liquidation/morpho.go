package liquidation

import (
	"context"
	"fmt"
	"math/big"
	"strings"
	"toolkit/query"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethclient"
)

type MorphoMarketParams struct {
	LoanToken       common.Address `json:"loanToken"`
	CollateralToken common.Address `json:"collateralToken"`
	Oracle          common.Address `json:"oracle"`
	Irm             common.Address `json:"irm"`
	Lltv            *big.Int       `json:"lltv"`
}

type LiquidationBonus struct {
	MarketId    common.Hash
	LoanToken   common.Address
	LoanAmount  *big.Int
	SeizedValue *big.Int
}

const (
	morphoLiquidationSignature = "0xa4946ede45d0c6f06a0f5ce92c9ad3b4751452d2fe0e25010783bcab57a67e41"

	morphoAddress = "0x8f5ae9CddB9f68de460C77730b018Ae7E04a140A"

	morphoAbi                  = `[{"inputs": [{"internalType": "Id","name": "","type": "bytes32"}],"name": "idToMarketParams","outputs": [{"internalType": "address","name": "loanToken","type": "address"},{"internalType": "address","name": "collateralToken","type": "address"},{"internalType": "address","name": "oracle","type": "address"},{"internalType": "address","name": "irm","type": "address"},{"internalType": "uint256","name": "lltv","type": "uint256"}],"stateMutability": "view","type": "function"}]`
	morphoChainlinkOracleV2Abi = `[{"inputs":[],"name":"price","outputs":[{"internalType":"uint256","name":"","type":"uint256"}],"stateMutability":"view","type":"function"}]`
	erc20Abi                   = `[
		{"inputs":[],"name":"decimals","outputs":[{"internalType":"uint8","name":"","type":"uint8"}],"stateMutability":"view","type":"function"},
		{"inputs":[],"name":"name","outputs":[{"internalType":"string","name":"name_","type":"string"}],"stateMutability":"view","type":"function"}
	]`
)

func MorphoLiquidations(ctx context.Context, client *ethclient.Client, startBlock *big.Int, endBlock *big.Int) ([]types.Log, error) {
	q := query.QueryConfig{
		FromBlock: startBlock,
		ToBlock:   endBlock,
		Addresses: []common.Address{common.HexToAddress(morphoAddress)},
		Topics:    [][]common.Hash{{common.HexToHash(morphoLiquidationSignature)}},
	}
	logs, err := query.PaginatedQuery(ctx, client, q)
	if err != nil {
		return nil, fmt.Errorf("failed to parse ABI: %v", err)
	}
	return logs, nil
}

func IdToMarketParams(ctx context.Context, client *ethclient.Client, id common.Hash, blockNumber uint64) (MorphoMarketParams, error) {
	parsedABI, err := abi.JSON(strings.NewReader(morphoAbi))
	if err != nil {
		return MorphoMarketParams{}, fmt.Errorf("failed to pack ABI: %v", err)
	}
	data, err := parsedABI.Pack("idToMarketParams", id)
	if err != nil {
		return MorphoMarketParams{}, fmt.Errorf("failed to pack ABI: %v", err)
	}
	morpho := common.HexToAddress(morphoAddress)
	msg := ethereum.CallMsg{
		To:   &morpho,
		Data: data,
	}
	output, err := client.CallContract(ctx, msg, big.NewInt(int64(blockNumber)))
	if err != nil {
		return MorphoMarketParams{}, fmt.Errorf("failed to call contract: %v", err)
	}

	var params MorphoMarketParams
	if err := parsedABI.UnpackIntoInterface(&params, "idToMarketParams", output); err != nil {
		return MorphoMarketParams{}, fmt.Errorf("failed to unpack ABI: %v", err)
	}
	return params, nil
}

func ParseMorphoLiquidationBonus(ctx context.Context, client *ethclient.Client, logItem types.Log) (LiquidationBonus, error) {
	marketParams, err := IdToMarketParams(ctx, client, logItem.Topics[1], logItem.BlockNumber)
	if err != nil {
		return LiquidationBonus{}, fmt.Errorf("failed to get market params: %v", err)
	}

	loanAmount := new(big.Int).SetBytes(logItem.Data[0:32])
	collateralToken := marketParams.CollateralToken
	collateralAmount := new(big.Int).SetBytes(logItem.Data[64:96])
	collateralDecimals, err := Decimals(ctx, client, collateralToken, logItem.BlockNumber)
	if err != nil {
		return LiquidationBonus{}, fmt.Errorf("failed to get collateral decimals: %v", err)
	}

	collateralPrice, err := CollateralToLoanPrice(ctx, client, marketParams.Oracle, logItem.BlockNumber)
	if err != nil {
		return LiquidationBonus{}, fmt.Errorf("failed to get collateral price: %v", err)
	}

	collateralValue := new(big.Int).Mul(collateralAmount, collateralPrice)
	collateralValue = collateralValue.Div(collateralValue, new(big.Int).Exp(big.NewInt(10), big.NewInt(int64(collateralDecimals+18)), nil))

	return LiquidationBonus{
		MarketId:    logItem.Topics[1],
		LoanToken:   marketParams.LoanToken,
		LoanAmount:  loanAmount,
		SeizedValue: collateralValue,
	}, nil
}

// 1 whole collateral token quoted by load token unit in 18 decimals
func CollateralToLoanPrice(ctx context.Context, client *ethclient.Client, oracle common.Address, blockNumber uint64) (*big.Int, error) {
	parsedABI, err := abi.JSON(strings.NewReader(morphoChainlinkOracleV2Abi))
	if err != nil {
		return nil, fmt.Errorf("failed to pack ABI: %v", err)
	}
	data, err := parsedABI.Pack("price")
	if err != nil {
		return nil, fmt.Errorf("failed to pack ABI: %v", err)
	}
	msg := ethereum.CallMsg{
		To:   &oracle,
		Data: data,
	}
	output, err := client.CallContract(ctx, msg, big.NewInt(int64(blockNumber)))
	if err != nil {
		return nil, fmt.Errorf("failed to call contract: %v", err)
	}
	var price *big.Int
	if err := parsedABI.UnpackIntoInterface(&price, "price", output); err != nil {
		return nil, fmt.Errorf("failed to unpack ABI: %v", err)
	}
	return price, nil
}

func Decimals(ctx context.Context, client *ethclient.Client, token common.Address, blockNumber uint64) (uint8, error) {
	parsedABI, err := abi.JSON(strings.NewReader(erc20Abi))
	if err != nil {
		return 0, fmt.Errorf("failed to pack ABI: %v", err)
	}
	data, err := parsedABI.Pack("decimals")
	if err != nil {
		return 0, fmt.Errorf("failed to pack ABI: %v", err)
	}
	msg := ethereum.CallMsg{
		To:   &token,
		Data: data,
	}
	var blockNumberBig *big.Int
	if blockNumber == 0 {
		blockNumberBig = nil
	} else {
		blockNumberBig = big.NewInt(int64(blockNumber))
	}
	output, err := client.CallContract(ctx, msg, blockNumberBig)
	if err != nil {
		return 0, fmt.Errorf("failed to call contract: %v", err)
	}
	var decimals uint8
	if err := parsedABI.UnpackIntoInterface(&decimals, "decimals", output); err != nil {
		return 0, fmt.Errorf("failed to unpack ABI: %v", err)
	}
	return decimals, nil
}

func (lb *LiquidationBonus) Bonus() *big.Int {
	bonus := new(big.Int).Sub(lb.SeizedValue, lb.LoanAmount)
	return bonus
}

func (lb *LiquidationBonus) BonusRate() *big.Float {
	bonusRate := new(big.Float).Quo(new(big.Float).SetInt(lb.Bonus()), new(big.Float).SetInt(lb.LoanAmount))
	return bonusRate
}
