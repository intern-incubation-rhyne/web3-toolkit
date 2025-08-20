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
	"github.com/ethereum/go-ethereum/rlp"
)

const (
	gasPriceOracle    = "0x420000000000000000000000000000000000000F"
	eulerRouter       = "0xdEb6135daed5470241843838944631Af12cE464B"
	wethAddress       = "0x4200000000000000000000000000000000000006"
	usdAddress        = "0x0000000000000000000000000000000000000348"
	evkAbi            = `[{"inputs":[],"name":"oracle","outputs":[{"internalType":"address","name":"","type":"address"}],"stateMutability":"view","type":"function"}]`
	priceOracleAbi    = `[{"inputs":[{"internalType":"uint256","name":"inAmount","type":"uint256"},{"internalType":"address","name":"base","type":"address"},{"internalType":"address","name":"quote","type":"address"}],"name":"getQuote","outputs":[{"internalType":"uint256","name":"","type":"uint256"}],"stateMutability":"view","type":"function"}]`
	gasPriceOracleAbi = `[{"inputs":[{"internalType":"bytes","name":"_data","type":"bytes"}],"name":"getL1Fee","outputs":[{"internalType":"uint256","name":"","type":"uint256"}],"stateMutability":"view","type":"function"}]`
)

func EVKLiquidations(ctx context.Context, client *ethclient.Client, startBlock *big.Int, endBlock *big.Int) ([]types.Log, error) {
	config := query.QueryConfig{
		FromBlock: startBlock,
		ToBlock:   endBlock,
		// Addresses: []common.Address{common.HexToAddress("0x1f3134c3f3f8add904b9635acbefc0ea0d0e1ffc")},
		Topics: [][]common.Hash{{common.HexToHash("0x8246cc71ab01533b5bebc672a636df812f10637ad720797319d5741d5ebb3962")}},
	}
	logs, err := query.PaginatedQuery(ctx, client, config)
	if err != nil {
		return nil, err
	}

	return logs, nil
}

func ParseEVKLiquidationRevenue(ctx context.Context, client *ethclient.Client, logItem types.Log) (*big.Int, error) {
	debtToken := logItem.Address
	debtAmount := new(big.Int).SetBytes(logItem.Data[32:64])
	debtValue, err := USDValue(ctx, client, logItem.BlockNumber, logItem.TxIndex, &debtToken, debtAmount)
	if err != nil {
		return nil, fmt.Errorf("failed to get debt value: %v", err)
	}

	collateralToken := common.BytesToAddress(logItem.Data[12:32])
	collateralAmount := new(big.Int).SetBytes(logItem.Data[64:96])
	collateralValue, err := USDValue(ctx, client, logItem.BlockNumber, logItem.TxIndex, &collateralToken, collateralAmount)
	if err != nil {
		return nil, fmt.Errorf("failed to get collateral value: %v", err)
	}

	// log.Println("debtValue: ", debtValue)
	// log.Println("collateralValue: ", collateralValue)
	revenue := new(big.Int).Sub(collateralValue, debtValue)

	return revenue, nil
}

func EVKOracle(ctx context.Context, client *ethclient.Client, evk *common.Address) (common.Address, error) {
	parsedABI, err := abi.JSON(strings.NewReader(evkAbi))
	if err != nil {
		return common.Address{}, err
	}

	data, err := parsedABI.Pack("oracle")
	if err != nil {
		return common.Address{}, err
	}

	msg := ethereum.CallMsg{
		To:   evk,
		Data: data,
	}

	output, err := client.CallContract(ctx, msg, nil)
	if err != nil {
		return common.Address{}, err
	}

	var oracle common.Address
	if err := parsedABI.UnpackIntoInterface(&oracle, "oracle", output); err != nil {
		return common.Address{}, err
	}

	return oracle, nil
}

func GetQuote(
	ctx context.Context, client *ethclient.Client, blockNumber uint64, txIndex uint,
	oracle *common.Address, inAmount *big.Int, base *common.Address, quote *common.Address,
) (*big.Int, error) {
	parsedABI, err := abi.JSON(strings.NewReader(priceOracleAbi))
	if err != nil {
		return nil, fmt.Errorf("failed to parse priceOracleAbi: %v", err)
	}
	data, err := parsedABI.Pack("getQuote", inAmount, base, quote)
	if err != nil {
		return nil, fmt.Errorf("failed to pack getQuote: %v", err)
	}

	msg := ethereum.CallMsg{
		To:   oracle,
		Data: data,
	}

	output, err := client.CallContract(ctx, msg, big.NewInt(int64(blockNumber+1)))
	if err != nil {
		return nil, fmt.Errorf("failed to call contract: %v", err)
	}

	var value *big.Int
	if err := parsedABI.UnpackIntoInterface(&value, "getQuote", output); err != nil {
		return nil, fmt.Errorf("failed to unpack getQuote: %v", err)
	}

	return value, nil
}

func USDValue(ctx context.Context, client *ethclient.Client, blockNumber uint64, txIndex uint, evkToken *common.Address, amount *big.Int) (*big.Int, error) {
	oracle, err := EVKOracle(ctx, client, evkToken)
	if err != nil {
		return nil, fmt.Errorf("failed to get quote: %v", err)
	}

	usd := common.HexToAddress(usdAddress)
	value, err := GetQuote(ctx, client, blockNumber, txIndex, &oracle, amount, evkToken, &usd)
	if err != nil {
		return nil, fmt.Errorf("failed to get quote: %v", err)
	}

	return value, nil
}

func EthPrice(ctx context.Context, client *ethclient.Client, blockNumber uint64) (*big.Int, error) {
	eth := common.HexToAddress(wethAddress)
	usd := common.HexToAddress(usdAddress)
	oracle := common.HexToAddress(eulerRouter)
	amount := big.NewInt(1000000000000000000)
	value, err := GetQuote(ctx, client, blockNumber, 0, &oracle, amount, &eth, &usd)
	if err != nil {
		return nil, fmt.Errorf("failed to get quote: %v", err)
	}
	return value, nil
}

func GetL1Fee(ctx context.Context, client *ethclient.Client, logItem types.Log) (*big.Int, error) {
	parsedABI, err := abi.JSON(strings.NewReader(gasPriceOracleAbi))
	if err != nil {
		return nil, fmt.Errorf("failed to parse gasPriceOracleAbi: %v", err)
	}
	tx, _, err := client.TransactionByHash(ctx, logItem.TxHash)
	if err != nil {
		return nil, fmt.Errorf("failed to get transaction: %v", err)
	}
	bytes, err := rlp.EncodeToBytes(tx)
	if err != nil {
		return nil, fmt.Errorf("failed to encode transaction: %v", err)
	}
	data, err := parsedABI.Pack("getL1Fee", bytes)
	if err != nil {
		return nil, fmt.Errorf("failed to pack getL1Fee: %v", err)
	}
	gasPriceOracleAddress := common.HexToAddress(gasPriceOracle)
	msg := ethereum.CallMsg{
		To:   &gasPriceOracleAddress,
		Data: data,
	}

	output, err := client.CallContract(ctx, msg, big.NewInt(int64(logItem.BlockNumber)))
	if err != nil {
		return nil, fmt.Errorf("failed to call contract: %v", err)
	}

	var value *big.Int
	if err := parsedABI.UnpackIntoInterface(&value, "getL1Fee", output); err != nil {
		return nil, fmt.Errorf("failed to unpack getL1Fee: %v", err)
	}
	return value, nil
}

// return profit in USD (18 decimals)
func ParseEVKLiquidationProfit(ctx context.Context, client *ethclient.Client, logItem types.Log) (*big.Int, error) {
	revenue, err := ParseEVKLiquidationRevenue(ctx, client, logItem)
	if err != nil {
		return nil, fmt.Errorf("failed to get revenue: %v", err)
	}
	receipt, err := client.TransactionReceipt(ctx, logItem.TxHash)
	if err != nil {
		return nil, fmt.Errorf("failed to get transaction: %v", err)
	}
	gasCost := new(big.Int).Mul(big.NewInt(int64(receipt.GasUsed)), receipt.EffectiveGasPrice)
	ethPrice, err := EthPrice(ctx, client, logItem.BlockNumber)
	if err != nil {
		return nil, fmt.Errorf("failed to get eth price: %v", err)
	}
	l1Gas, err := GetL1Fee(ctx, client, logItem)
	if err != nil {
		return nil, fmt.Errorf("failed to get l1 gas: %v", err)
	}
	gasCost = new(big.Int).Add(gasCost, l1Gas)
	gasCost = gasCost.Mul(gasCost, ethPrice)
	gasCost = gasCost.Div(gasCost, big.NewInt(1e18))
	profit := new(big.Int).Sub(revenue, gasCost)
	return profit, nil
}
