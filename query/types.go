package query

import (
	"encoding/json"
	"sync"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core/types"
)

type BlockOverrides struct {
	Number        *hexutil.Big       `json:"number,omitempty"`
	Time          *hexutil.Uint64    `json:"time,omitempty"`
	GasLimit      *hexutil.Uint64    `json:"gasLimit,omitempty"`
	FeeRecipient  *common.Address    `json:"feeRecipient,omitempty"`
	PrevRandao    *common.Hash       `json:"prevRandao,omitempty"`
	BaseFeePerGas *hexutil.Big       `json:"baseFeePerGas,omitempty"`
	BlobBaseFee   *hexutil.Big       `json:"blobBaseFee,omitempty"`
	BeaconRoot    *common.Hash       `json:"beaconRoot,omitempty"`
	Withdrawals   *types.Withdrawals `json:"withdrawals,omitempty"`
}

// RPCRequest represents a JSON-RPC request
type RPCRequest struct {
	JSONRPC string      `json:"jsonrpc"`
	Method  string      `json:"method"`
	Params  interface{} `json:"params"`
	ID      int         `json:"id"`
}

type RPCResponse struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      int             `json:"id"`
	Result  json.RawMessage `json:"result"`
	Error   *RPCError       `json:"error,omitempty"`
}

type RPCError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

// SpotPriceAggregator is an auto generated Go binding around an Ethereum contract.
type SpotPriceAggregator struct {
	SpotPriceAggregatorCaller     // Read-only binding to the contract
	SpotPriceAggregatorTransactor // Write-only binding to the contract
	SpotPriceAggregatorFilterer   // Log filterer for contract events
}

// SpotPriceAggregatorCaller is an auto generated read-only Go binding around an Ethereum contract.
type SpotPriceAggregatorCaller struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// SpotPriceAggregatorTransactor is an auto generated write-only Go binding around an Ethereum contract.
type SpotPriceAggregatorTransactor struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// SpotPriceAggregatorFilterer is an auto generated log filtering Go binding around an Ethereum contract events.
type SpotPriceAggregatorFilterer struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// MetaData collects all metadata for a bound contract.
type MetaData struct {
	mu   sync.Mutex
	Sigs map[string]string
	Bin  string
	ABI  string
	ab   *abi.ABI
}

// SpotPriceAggregatorMetaData contains all meta data concerning the SpotPriceAggregator contract.
var SpotPriceAggregatorMetaData = &bind.MetaData{
	ABI: "[{\"inputs\":[{\"internalType\":\"contractMultiWrapper\",\"name\":\"_multiWrapper\",\"type\":\"address\"},{\"internalType\":\"contractIOracle[]\",\"name\":\"existingOracles\",\"type\":\"address[]\"},{\"internalType\":\"enumOffchainOracle.OracleType[]\",\"name\":\"oracleTypes\",\"type\":\"uint8[]\"},{\"internalType\":\"contractIERC20[]\",\"name\":\"existingConnectors\",\"type\":\"address[]\"},{\"internalType\":\"contractIERC20\",\"name\":\"wBase\",\"type\":\"address\"},{\"internalType\":\"address\",\"name\":\"owner_\",\"type\":\"address\"}],\"stateMutability\":\"nonpayable\",\"type\":\"constructor\"},{\"inputs\":[],\"name\":\"ArraysLengthMismatch\",\"type\":\"error\"},{\"inputs\":[],\"name\":\"ConnectorAlreadyAdded\",\"type\":\"error\"},{\"inputs\":[],\"name\":\"InvalidOracleTokenKind\",\"type\":\"error\"},{\"inputs\":[],\"name\":\"MathOverflowedMulDiv\",\"type\":\"error\"},{\"inputs\":[],\"name\":\"OracleAlreadyAdded\",\"type\":\"error\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"owner\",\"type\":\"address\"}],\"name\":\"OwnableInvalidOwner\",\"type\":\"error\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"account\",\"type\":\"address\"}],\"name\":\"OwnableUnauthorizedAccount\",\"type\":\"error\"},{\"inputs\":[],\"name\":\"SameTokens\",\"type\":\"error\"},{\"inputs\":[],\"name\":\"TooBigThreshold\",\"type\":\"error\"},{\"inputs\":[],\"name\":\"UnknownConnector\",\"type\":\"error\"},{\"inputs\":[],\"name\":\"UnknownOracle\",\"type\":\"error\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":false,\"internalType\":\"contractIERC20\",\"name\":\"connector\",\"type\":\"address\"}],\"name\":\"ConnectorAdded\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":false,\"internalType\":\"contractIERC20\",\"name\":\"connector\",\"type\":\"address\"}],\"name\":\"ConnectorRemoved\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":false,\"internalType\":\"contractMultiWrapper\",\"name\":\"multiWrapper\",\"type\":\"address\"}],\"name\":\"MultiWrapperUpdated\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":false,\"internalType\":\"contractIOracle\",\"name\":\"oracle\",\"type\":\"address\"},{\"indexed\":false,\"internalType\":\"enumOffchainOracle.OracleType\",\"name\":\"oracleType\",\"type\":\"uint8\"}],\"name\":\"OracleAdded\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":false,\"internalType\":\"contractIOracle\",\"name\":\"oracle\",\"type\":\"address\"},{\"indexed\":false,\"internalType\":\"enumOffchainOracle.OracleType\",\"name\":\"oracleType\",\"type\":\"uint8\"}],\"name\":\"OracleRemoved\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"internalType\":\"address\",\"name\":\"previousOwner\",\"type\":\"address\"},{\"indexed\":true,\"internalType\":\"address\",\"name\":\"newOwner\",\"type\":\"address\"}],\"name\":\"OwnershipTransferred\",\"type\":\"event\"},{\"inputs\":[{\"internalType\":\"contractIERC20\",\"name\":\"connector\",\"type\":\"address\"}],\"name\":\"addConnector\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"contractIOracle\",\"name\":\"oracle\",\"type\":\"address\"},{\"internalType\":\"enumOffchainOracle.OracleType\",\"name\":\"oracleKind\",\"type\":\"uint8\"}],\"name\":\"addOracle\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"connectors\",\"outputs\":[{\"internalType\":\"contractIERC20[]\",\"name\":\"allConnectors\",\"type\":\"address[]\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"contractIERC20\",\"name\":\"srcToken\",\"type\":\"address\"},{\"internalType\":\"contractIERC20\",\"name\":\"dstToken\",\"type\":\"address\"},{\"internalType\":\"bool\",\"name\":\"useWrappers\",\"type\":\"bool\"}],\"name\":\"getRate\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"weightedRate\",\"type\":\"uint256\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"contractIERC20\",\"name\":\"srcToken\",\"type\":\"address\"},{\"internalType\":\"bool\",\"name\":\"useSrcWrappers\",\"type\":\"bool\"}],\"name\":\"getRateToEth\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"weightedRate\",\"type\":\"uint256\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"contractIERC20\",\"name\":\"srcToken\",\"type\":\"address\"},{\"internalType\":\"bool\",\"name\":\"useSrcWrappers\",\"type\":\"bool\"},{\"internalType\":\"contractIERC20[]\",\"name\":\"customConnectors\",\"type\":\"address[]\"},{\"internalType\":\"uint256\",\"name\":\"thresholdFilter\",\"type\":\"uint256\"}],\"name\":\"getRateToEthWithCustomConnectors\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"weightedRate\",\"type\":\"uint256\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"contractIERC20\",\"name\":\"srcToken\",\"type\":\"address\"},{\"internalType\":\"bool\",\"name\":\"useSrcWrappers\",\"type\":\"bool\"},{\"internalType\":\"uint256\",\"name\":\"thresholdFilter\",\"type\":\"uint256\"}],\"name\":\"getRateToEthWithThreshold\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"weightedRate\",\"type\":\"uint256\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"contractIERC20\",\"name\":\"srcToken\",\"type\":\"address\"},{\"internalType\":\"contractIERC20\",\"name\":\"dstToken\",\"type\":\"address\"},{\"internalType\":\"bool\",\"name\":\"useWrappers\",\"type\":\"bool\"},{\"internalType\":\"contractIERC20[]\",\"name\":\"customConnectors\",\"type\":\"address[]\"},{\"internalType\":\"uint256\",\"name\":\"thresholdFilter\",\"type\":\"uint256\"}],\"name\":\"getRateWithCustomConnectors\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"weightedRate\",\"type\":\"uint256\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"contractIERC20\",\"name\":\"srcToken\",\"type\":\"address\"},{\"internalType\":\"contractIERC20\",\"name\":\"dstToken\",\"type\":\"address\"},{\"internalType\":\"bool\",\"name\":\"useWrappers\",\"type\":\"bool\"},{\"internalType\":\"uint256\",\"name\":\"thresholdFilter\",\"type\":\"uint256\"}],\"name\":\"getRateWithThreshold\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"weightedRate\",\"type\":\"uint256\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"contractIERC20\",\"name\":\"srcToken\",\"type\":\"address\"},{\"internalType\":\"bool\",\"name\":\"useSrcWrappers\",\"type\":\"bool\"},{\"internalType\":\"contractIERC20[]\",\"name\":\"customConnectors\",\"type\":\"address[]\"},{\"internalType\":\"uint256\",\"name\":\"thresholdFilter\",\"type\":\"uint256\"}],\"name\":\"getRatesAndWeightsToEthWithCustomConnectors\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"wrappedPrice\",\"type\":\"uint256\"},{\"components\":[{\"internalType\":\"uint256\",\"name\":\"maxOracleWeight\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"size\",\"type\":\"uint256\"},{\"components\":[{\"internalType\":\"uint256\",\"name\":\"rate\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"weight\",\"type\":\"uint256\"}],\"internalType\":\"structOraclePrices.OraclePrice[]\",\"name\":\"oraclePrices\",\"type\":\"tuple[]\"}],\"internalType\":\"structOraclePrices.Data\",\"name\":\"ratesAndWeights\",\"type\":\"tuple\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"contractIERC20\",\"name\":\"srcToken\",\"type\":\"address\"},{\"internalType\":\"contractIERC20\",\"name\":\"dstToken\",\"type\":\"address\"},{\"internalType\":\"bool\",\"name\":\"useWrappers\",\"type\":\"bool\"},{\"internalType\":\"contractIERC20[]\",\"name\":\"customConnectors\",\"type\":\"address[]\"},{\"internalType\":\"uint256\",\"name\":\"thresholdFilter\",\"type\":\"uint256\"}],\"name\":\"getRatesAndWeightsWithCustomConnectors\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"wrappedPrice\",\"type\":\"uint256\"},{\"components\":[{\"internalType\":\"uint256\",\"name\":\"maxOracleWeight\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"size\",\"type\":\"uint256\"},{\"components\":[{\"internalType\":\"uint256\",\"name\":\"rate\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"weight\",\"type\":\"uint256\"}],\"internalType\":\"structOraclePrices.OraclePrice[]\",\"name\":\"oraclePrices\",\"type\":\"tuple[]\"}],\"internalType\":\"structOraclePrices.Data\",\"name\":\"ratesAndWeights\",\"type\":\"tuple\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"multiWrapper\",\"outputs\":[{\"internalType\":\"contractMultiWrapper\",\"name\":\"\",\"type\":\"address\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"oracles\",\"outputs\":[{\"internalType\":\"contractIOracle[]\",\"name\":\"allOracles\",\"type\":\"address[]\"},{\"internalType\":\"enumOffchainOracle.OracleType[]\",\"name\":\"oracleTypes\",\"type\":\"uint8[]\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"owner\",\"outputs\":[{\"internalType\":\"address\",\"name\":\"\",\"type\":\"address\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"contractIERC20\",\"name\":\"connector\",\"type\":\"address\"}],\"name\":\"removeConnector\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"contractIOracle\",\"name\":\"oracle\",\"type\":\"address\"},{\"internalType\":\"enumOffchainOracle.OracleType\",\"name\":\"oracleKind\",\"type\":\"uint8\"}],\"name\":\"removeOracle\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"renounceOwnership\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"contractMultiWrapper\",\"name\":\"_multiWrapper\",\"type\":\"address\"}],\"name\":\"setMultiWrapper\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"newOwner\",\"type\":\"address\"}],\"name\":\"transferOwnership\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"}]",
}
