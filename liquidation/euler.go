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

type EulerLogWithRevenue struct {
	types.Log
	USDRevenue *big.Float `json:"usdRevenue"`
}

const (
	evkLiquidationSignature = "0x8246cc71ab01533b5bebc672a636df812f10637ad720797319d5741d5ebb3962"

	// // unichain
	// gasPriceOracle = "0x420000000000000000000000000000000000000F"
	// eulerRouter    = "0xdEb6135daed5470241843838944631Af12cE464B"
	// wethAddress    = "0x4200000000000000000000000000000000000006"
	// usdAddress     = "0x0000000000000000000000000000000000000348"

	// mainnet
	gasPriceOracle = "0x0000000000000000000000000000000000000000"
	eulerRouter    = "0x83B3b76873D36A28440cF53371dF404c42497136"
	wethAddress    = "0xC02aaA39b223FE8D0A0e5C4F27eAD9083C756Cc2"
	usdAddress     = "0x0000000000000000000000000000000000000348"

	evkAbi            = `[{"inputs":[],"name":"oracle","outputs":[{"internalType":"address","name":"","type":"address"}],"stateMutability":"view","type":"function"}]`
	priceOracleAbi    = `[{"inputs":[{"internalType":"uint256","name":"inAmount","type":"uint256"},{"internalType":"address","name":"base","type":"address"},{"internalType":"address","name":"quote","type":"address"}],"name":"getQuote","outputs":[{"internalType":"uint256","name":"","type":"uint256"}],"stateMutability":"view","type":"function"}]`
	gasPriceOracleAbi = `[
		{"inputs":[{"internalType":"bytes","name":"_data","type":"bytes"}],"name":"getL1Fee","outputs":[{"internalType":"uint256","name":"","type":"uint256"}],"stateMutability":"view","type":"function"},
		{"inputs":[],"name":"name","outputs":[{"internalType":"string","name":"name_","type":"string"}],"stateMutability":"view","type":"function"}
	]`
)

func EVKLiquidations(ctx context.Context, client *ethclient.Client, startBlock *big.Int, endBlock *big.Int) ([]types.Log, error) {
	q := query.QueryConfig{
		FromBlock: startBlock,
		ToBlock:   endBlock,
		// Addresses: []common.Address{common.HexToAddress("0x1f3134c3f3f8add904b9635acbefc0ea0d0e1ffc")},
		Topics: [][]common.Hash{{common.HexToHash(evkLiquidationSignature)}},
		// ChunkSize: 500, // mainnet
		// ChunkSize: 10000, // unichain
	}
	logs, err := query.PaginatedQuery(ctx, client, q)
	if err != nil {
		return nil, err
	}

	return logs, nil
}

func ParseEVKLiquidationRevenue(ctx context.Context, rpcUrl string, client *ethclient.Client, logItem types.Log) (*big.Int, error) {
	if len(logItem.Data) != 96 {
		return nil, fmt.Errorf("invalid log data length: %d", len(logItem.Data))
	}
	debtToken := logItem.Address
	debtAmount := new(big.Int).SetBytes(logItem.Data[32:64])
	// debtValue, err := USDValue(ctx, client, logItem.BlockNumber, logItem.TxIndex, &debtToken, debtAmount)
	// debtValue, err := query.TokenToEthValue(debtToken, debtAmount, big.NewInt(int64(logItem.BlockNumber)), client)
	// if err != nil {
	// 	return nil, fmt.Errorf("failed to get debt value: %v", err)
	// }

	collateralToken := common.BytesToAddress(logItem.Data[12:32])
	collateralAmount := new(big.Int).SetBytes(logItem.Data[64:96])
	// collateralValue, err := USDValue(ctx, client, logItem.BlockNumber, logItem.TxIndex, &collateralToken, collateralAmount)
	// collateralValue, err := query.TokenToEthValue(collateralToken, collateralAmount, big.NewInt(int64(logItem.BlockNumber)), client)
	// if err != nil {
	// 	return nil, fmt.Errorf("failed to get collateral value: %v", err)
	// }

	// log.Println("debtToken: ", debtToken)
	// log.Println("debtAmount: ", debtAmount)
	// log.Println("debtValue: ", debtValue)
	// log.Println("collateralToken: ", collateralToken)
	// log.Println("collateralAmount: ", collateralAmount)
	// log.Println("collateralValue: ", collateralValue)
	// revenue := new(big.Int).Sub(collateralValue, debtValue)

	revenue, debtValue, collateralValue, err := GetEulerRevenue(ctx, rpcUrl, client, debtToken, debtAmount, collateralToken, collateralAmount, big.NewInt(int64(logItem.BlockNumber)), logItem.TxIndex+1)
	if err != nil {
		return nil, fmt.Errorf("failed to get revenue: %v", err)
	}

	_, _ = debtValue, collateralValue

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
func ParseEVKLiquidationProfit(ctx context.Context, rpcUrl string, client *ethclient.Client, txHash common.Hash) (*big.Int, error) {
	receipt, err := client.TransactionReceipt(ctx, txHash)
	if err != nil {
		return nil, fmt.Errorf("failed to get transaction: %v", err)
	}
	var logs []types.Log
	for _, log := range receipt.Logs {
		if len(log.Topics) > 0 && log.Topics[0] == common.HexToHash(evkLiquidationSignature) {
			logs = append(logs, *log)
		}
	}

	revenueSum := big.NewInt(0)
	for i, logItem := range logs {
		revenue, err := ParseEVKLiquidationRevenue(ctx, rpcUrl, client, logItem)
		if err != nil {
			return nil, fmt.Errorf("failed to get revenue: %v", err)
		}
		revenueSum = new(big.Int).Add(revenueSum, revenue)
		fmt.Printf("    liquidation event %d revenue: %v\n", i, revenue)
	}

	// gasCost := new(big.Int).Mul(big.NewInt(int64(receipt.GasUsed)), receipt.EffectiveGasPrice)

	// directBribe, err := query.Bribe(ctx, client, txHash)
	// if err != nil {
	// 	return nil, fmt.Errorf("failed to get bribe: %v", err)
	// }
	// profit := new(big.Int).Sub(revenueSum, new(big.Int).Add(gasCost, directBribe))

	// // fmt.Printf("  %s revenue: %v\n", txHash.Hex(), revenueSum)
	// // fmt.Printf("  %s gasCost: %v\n", txHash.Hex(), gasCost)
	// // fmt.Printf("  %s directBribe: %v\n", txHash.Hex(), directBribe)
	// // fmt.Printf("  %s profit: %v\n", txHash.Hex(), profit)
	// fmt.Printf("  total revenue: %v\n", revenueSum)
	// fmt.Printf("  gasCost: %v\n", gasCost)
	// fmt.Printf("  directBribe: %v\n", directBribe)
	// return profit, nil

	return revenueSum, nil
}

func TokenName(ctx context.Context, client *ethclient.Client, token common.Address, blockNumber uint64) (string, error) {
	parsedABI, err := abi.JSON(strings.NewReader(erc20Abi))
	if err != nil {
		return "", fmt.Errorf("failed to parse erc20Abi: %v", err)
	}
	data, err := parsedABI.Pack("name")
	if err != nil {
		return "", fmt.Errorf("failed to pack name: %v", err)
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
		return "", fmt.Errorf("failed to call contract: %v", err)
	}
	var name string
	if err := parsedABI.UnpackIntoInterface(&name, "name", output); err != nil {
		return "", fmt.Errorf("failed to unpack name: %v", err)
	}
	return name, nil
}

func ParseEVKLiquidationLogs(ctx context.Context, client *ethclient.Client, logs []types.Log) ([]EulerLogWithRevenue, error) {
	return nil, nil
}
