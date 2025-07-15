package slot0

import (
	"context"
	"fmt"
	"log"
	"math/big"
	"strings"
	"toolkit/config"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethclient"
)

const slot0Abi = `[{"name":"slot0","outputs":[{"internalType":"uint160","name":"sqrtPriceX96","type":"uint160"},{"internalType":"int24","name":"tick","type":"int24"},{"internalType":"uint16","name":"","type":"uint16"},{"internalType":"uint16","name":"","type":"uint16"},{"internalType":"uint16","name":"","type":"uint16"},{"internalType":"uint8","name":"","type":"uint8"},{"internalType":"bool","name":"","type":"bool"}],"stateMutability":"view","type":"function"}]`

type Slot0 struct {
	SqrtPriceX96               *big.Int
	Tick                       *big.Int
	ObservationIndex           uint16
	ObservationCardinality     uint16
	ObservationCardinalityNext uint16
	FeeProtocol                uint8
	Unlocked                   bool
}

func (s Slot0) String() string {
	return fmt.Sprintf(`SqrtPriceX96: %v
Tick: %v
ObservationIndex: %v
ObservationCardinality: %v
ObservationCardinalityNext: %v
FeeProtocol: %v
Unlocked: %v`,
		s.SqrtPriceX96,
		s.Tick,
		s.ObservationIndex,
		s.ObservationCardinality,
		s.ObservationCardinalityNext,
		s.FeeProtocol,
		s.Unlocked,
	)
}

func slot0FromResult(out []interface{}) Slot0 {
	return Slot0{
		SqrtPriceX96:               out[0].(*big.Int),
		Tick:                       out[1].(*big.Int),
		ObservationIndex:           out[2].(uint16),
		ObservationCardinality:     out[3].(uint16),
		ObservationCardinalityNext: out[4].(uint16),
		FeeProtocol:                out[5].(uint8),
		Unlocked:                   out[6].(bool),
	}
}

// PriceFromSqrtPriceX96 calculates the price from sqrtPriceX96 as (sqrtPriceX96 / 2^96)^2
func PriceFromSqrtPriceX96(sqrtPriceX96 *big.Int) *big.Float {
	// Convert sqrtPriceX96 to big.Float
	sqrtPrice := new(big.Float).SetInt(sqrtPriceX96)
	// 2^96 as big.Float
	twoPow96 := new(big.Float).SetInt(new(big.Int).Lsh(big.NewInt(1), 96))
	// sqrtPriceX96 / 2^96
	ratio := new(big.Float).Quo(sqrtPrice, twoPow96)
	// (sqrtPriceX96 / 2^96)^2
	price := new(big.Float).Mul(ratio, ratio)
	return price
}

// PrepareSlot0Call prepares the CallMsg and parsed ABI for the slot0 call.
func PrepareSlot0Call(ca string) (ethereum.CallMsg, abi.ABI, error) {
	poolAddress := common.HexToAddress(ca)

	parsedABI, err := abi.JSON(strings.NewReader(slot0Abi))
	if err != nil {
		return ethereum.CallMsg{}, abi.ABI{}, err
	}

	data, err := parsedABI.Pack("slot0")
	if err != nil {
		return ethereum.CallMsg{}, abi.ABI{}, err
	}

	msg := ethereum.CallMsg{
		To:   &poolAddress,
		Data: data,
	}

	return msg, parsedABI, nil
}

func Query(ca string) Slot0 {
	msg, parsedABI, err := PrepareSlot0Call(ca)
	if err != nil {
		log.Fatalf("Error preparing slot0 call: %v", err)
	}

	client, err := ethclient.Dial(config.HTTP_RPC)
	if err != nil {
		log.Fatalf("Error connecting RPC: %v", err)
	}

	output, err := client.CallContract(context.Background(), msg, nil)
	if err != nil {
		log.Fatalf("Error eth_call: %v", err)
	}

	result, err := parsedABI.Unpack("slot0", output)
	if err != nil {
		log.Fatalf("Error unpacking return value: %v", err)
	}

	return slot0FromResult(result)
}
