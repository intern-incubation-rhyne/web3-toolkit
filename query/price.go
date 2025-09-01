package query

import (
	"context"
	"fmt"
	"math"
	"math/big"
	"strings"
	"toolkit/trace"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethclient"
)

const (
	wethAddress                             = "0xC02aaA39b223FE8D0A0e5C4F27eAD9083C756Cc2"
	ethAddress                              = "0xEeeeeEeeeEeEeeEeEeEeeEEEeeeeEeeeeeeeEEeE"
	oneInchSpotPriceAggregatorV1BlockNumber = 12522266
	oneInchSpotPriceAggregatorV1            = "0x07D91f5fb9Bf7798734C3f606dB065549F6893bb"
	oneInchSpotPriceAggregatorV2BlockNumber = 16995101
	oneInchSpotPriceAggregatorV2            = "0x3E1Fe1Bd5a5560972bFa2D393b9aC18aF279fF56"
	oneInchSpotPriceAggregatorV3BlockNumber = 17684577
	oneInchSpotPriceAggregatorV3            = "0x52cbE0f49CcdD4Dc6E9C13BAb024EABD2842045B"
	oneInchSpotPriceAggregatorV4BlockNumber = 18040583
	oneInchSpotPriceAggregatorV4            = "0x0AdDd25a91563696D8567Df78D5A01C9a991F9B8"
	oneInchSpotPriceAggregatorV5BlockNumber = 20227911
	oneInchSpotPriceAggregatorV5            = "0xf224a25453D76A41c4427DD1C05369BC9f498444"
	oneInchSpotPriceAggregatorV6BlockNumber = 20535992
	oneInchSpotPriceAggregatorV6            = "0x00000000000D6FFc74A8feb35aF5827bf57f6786"
)

func ToWei(value *big.Float, unit string) *big.Int {
	newValue := new(big.Float).Set(value)
	switch unit {
	case "eth", "ether":
		newValue.Mul(newValue, big.NewFloat(math.Pow10(18)))
	case "gwei":
		newValue.Mul(newValue, big.NewFloat(math.Pow10(9)))
	}
	valueInt, _ := newValue.Int(nil)
	return valueInt
}

func GetTokenRateToETH(tokenAddress common.Address, blockNumber *big.Int, client *ethclient.Client) (*big.Int, error) {
	if tokenAddress == common.HexToAddress(wethAddress) || tokenAddress == common.HexToAddress(ethAddress) {
		return ToWei(big.NewFloat(1), "eth"), nil
	}

	var oneInchSpotPriceAggregatorAddress common.Address
	if blockNumber.Cmp(big.NewInt(oneInchSpotPriceAggregatorV6BlockNumber)) >= 0 {
		oneInchSpotPriceAggregatorAddress = common.HexToAddress(oneInchSpotPriceAggregatorV6)
	} else if blockNumber.Cmp(big.NewInt(oneInchSpotPriceAggregatorV5BlockNumber)) >= 0 {
		oneInchSpotPriceAggregatorAddress = common.HexToAddress(oneInchSpotPriceAggregatorV5)
	} else if blockNumber.Cmp(big.NewInt(oneInchSpotPriceAggregatorV4BlockNumber)) >= 0 {
		oneInchSpotPriceAggregatorAddress = common.HexToAddress(oneInchSpotPriceAggregatorV4)
	} else if blockNumber.Cmp(big.NewInt(oneInchSpotPriceAggregatorV3BlockNumber)) >= 0 {
		oneInchSpotPriceAggregatorAddress = common.HexToAddress(oneInchSpotPriceAggregatorV3)
	} else if blockNumber.Cmp(big.NewInt(oneInchSpotPriceAggregatorV2BlockNumber)) >= 0 {
		oneInchSpotPriceAggregatorAddress = common.HexToAddress(oneInchSpotPriceAggregatorV2)
	} else if blockNumber.Cmp(big.NewInt(oneInchSpotPriceAggregatorV1BlockNumber)) >= 0 {
		oneInchSpotPriceAggregatorAddress = common.HexToAddress(oneInchSpotPriceAggregatorV1)
	} else {
		return nil, fmt.Errorf("oneinch spot price aggregator is not supported for this block number")
	}

	spotPriceAggregatorContract, err := NewSpotPriceAggregator(oneInchSpotPriceAggregatorAddress, client)
	if err != nil {
		return nil, fmt.Errorf("failed to get oneinch spot price aggregator contract: %w", err)
	}

	rate, err := spotPriceAggregatorContract.GetRateToEth(&bind.CallOpts{
		BlockNumber: blockNumber,
	}, tokenAddress, true)
	if err != nil {
		return nil, fmt.Errorf("failed to get rate: %w", err)
	}

	return rate, nil
}

func TokenToEthValue(tokenAddress common.Address, amount *big.Int, blockNumber *big.Int, client *ethclient.Client) (*big.Int, error) {
	rate, err := GetTokenRateToETH(tokenAddress, blockNumber, client)
	if err != nil {
		return nil, fmt.Errorf("failed to get token rate to eth: %w", err)
	}

	price := big.NewInt(0)
	if amount.Cmp(big.NewInt(0)) > 0 {
		exp := new(big.Int).Exp(big.NewInt(10), big.NewInt(18), nil)
		temp := new(big.Int).Mul(rate, amount)
		price = new(big.Int).Div(temp, exp) // unit: wei
	} else if amount.Cmp(big.NewInt(0)) < 0 {
		amount = new(big.Int).Neg(amount)
		exp := new(big.Int).Exp(big.NewInt(10), big.NewInt(18), nil)
		temp := new(big.Int).Mul(rate, amount)
		price = new(big.Int).Div(temp, exp) // unit: wei
		price = new(big.Int).Neg(price)
	}
	return price, nil
}

// bindSpotPriceAggregator binds a generic wrapper to an already deployed contract.
func bindSpotPriceAggregator(address common.Address, caller bind.ContractCaller, transactor bind.ContractTransactor, filterer bind.ContractFilterer) (*bind.BoundContract, error) {
	parsed, err := SpotPriceAggregatorMetaData.GetAbi()
	if err != nil {
		return nil, err
	}
	return bind.NewBoundContract(address, *parsed, caller, transactor, filterer), nil
}

// NewSpotPriceAggregator creates a new instance of SpotPriceAggregator, bound to a specific deployed contract.
func NewSpotPriceAggregator(address common.Address, backend bind.ContractBackend) (*SpotPriceAggregator, error) {
	contract, err := bindSpotPriceAggregator(address, backend, backend, backend)
	if err != nil {
		return nil, err
	}
	return &SpotPriceAggregator{SpotPriceAggregatorCaller: SpotPriceAggregatorCaller{contract: contract}, SpotPriceAggregatorTransactor: SpotPriceAggregatorTransactor{contract: contract}, SpotPriceAggregatorFilterer: SpotPriceAggregatorFilterer{contract: contract}}, nil
}

// GetRateToEth is a free data retrieval call binding the contract method 0x7de4fd10.
//
// Solidity: function getRateToEth(address srcToken, bool useSrcWrappers) view returns(uint256 weightedRate)
func (_SpotPriceAggregator *SpotPriceAggregatorCaller) GetRateToEth(opts *bind.CallOpts, srcToken common.Address, useSrcWrappers bool) (*big.Int, error) {
	var out []interface{}
	err := _SpotPriceAggregator.contract.Call(opts, &out, "getRateToEth", srcToken, useSrcWrappers)

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

func (m *MetaData) GetAbi() (*abi.ABI, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.ab != nil {
		return m.ab, nil
	}
	if parsed, err := abi.JSON(strings.NewReader(m.ABI)); err != nil {
		return nil, err
	} else {
		m.ab = &parsed
	}
	return m.ab, nil
}

func Bribe(ctx context.Context, client *ethclient.Client, txHash common.Hash) (*big.Int, error) {
	receipt, err := client.TransactionReceipt(ctx, txHash)
	if err != nil {
		return nil, fmt.Errorf("failed to get transaction receipt: %v", err)
	}

	// 1. get base fee
	header, err := client.HeaderByHash(ctx, receipt.BlockHash)
	if err != nil {
		return nil, fmt.Errorf("failed to get block header: %v", err)
	}
	// baseFee := header.BaseFee

	// 2. get tx EffectiveGasPrice and gas used
	// effectiveGasPrice := receipt.EffectiveGasPrice
	// gasUsed := big.NewInt(int64(receipt.GasUsed))

	// 3. trace tx to get the transfer amount to coinbase
	coinbase := header.Coinbase
	traceConfig := map[string]any{"tracer": "callTracer"}
	var traceResult trace.CallTrace
	if err := client.Client().CallContext(ctx, &traceResult, "debug_traceTransaction", receipt.TxHash, traceConfig); err != nil {
		return nil, fmt.Errorf("RPC call to debug_traceTransaction failed: %v", err)
	}
	directBribe := big.NewInt(0)
	getDirectBribe(&traceResult, coinbase, directBribe)

	// 4. bribe = DirectTransfer + (EffectiveGasPrice - BaseFee) * GasUsed
	// priorityFee := new(big.Int).Sub(effectiveGasPrice, baseFee)
	// bribe := new(big.Int).Add(directBribe, priorityFee.Mul(priorityFee, gasUsed))
	return directBribe, nil
}

// sumPayments recursively walks through the call frames and sums up values sent to the coinbase.
func getDirectBribe(frame *trace.CallTrace, coinbase common.Address, total *big.Int) {
	if frame == nil {
		return
	}

	// If this frame's destination is the coinbase and it has value, add it to the total.
	if frame.To != nil && *frame.To == coinbase && frame.Value != nil {
		total.Add(total, frame.Value.ToInt())
	}

	// Recurse into all sub-calls.
	for _, subCall := range frame.Calls {
		getDirectBribe(&subCall, coinbase, total)
	}
}
