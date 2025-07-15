package trace

import (
	"context"
	"log"

	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethclient"
)

type callLog struct {
	Address  common.Address `json:"address"`
	Topics   []common.Hash  `json:"topics"`
	Data     hexutil.Bytes  `json:"data"`
	Position hexutil.Uint   `json:"position"`
	Decoded  []string       `json:"decodedData,omitempty"`
}

type callTrace struct {
	Type         string         `json:"type"`
	From         common.Address `json:"from"`
	To           common.Address `json:"to"`
	Value        string         `json:"value,omitempty"`   // Hex value, e.g., "0x0"
	Gas          string         `json:"gas,omitempty"`     // Hex value
	GasUsed      string         `json:"gasUsed,omitempty"` // Hex value
	Error        string         `json:"error,omitempty"`
	RevertReason string         `json:"revertReason,omitempty"`
	Calls        []callTrace    `json:"calls,omitempty"` // Recursive call frames
	Logs         []callLog      `json:"logs,omitempty"`
}

func GetSender(tx *types.Transaction) *common.Address {
	signer := types.LatestSignerForChainID(tx.ChainId())

	from, err := types.Sender(signer, tx)
	if err != nil {
		log.Fatalln("failed to get sender: ", err)
		return nil
	}

	return &from
}

func decodeLogData(logEntry *callLog) {
	data := logEntry.Data
	var decoded []string
	for i := 0; i+32 <= len(data); i += 32 {
		word := data[i : i+32]
		u := new(big.Int)
		u.SetBytes(word)
		decoded = append(decoded, u.String())
	}
	logEntry.Decoded = decoded
}

func decodeData(frame *callTrace) {
	for i := range frame.Logs {
		decodeLogData(&frame.Logs[i])
	}
	for i := range frame.Calls {
		decodeData(&frame.Calls[i])
	}
}

func TraceCall(ctx context.Context, client *ethclient.Client, txHash string, txIndex int) callTrace {
	tx, _, err := client.TransactionByHash(ctx, common.HexToHash(txHash))
	if err != nil {
		log.Fatalf("Failed to get transaction: %v", err)
	}

	receipt, err := client.TransactionReceipt(ctx, tx.Hash())
	if err != nil {
		log.Fatalf("Failed to get receipt: %v", err)
	}

	callArgs := map[string]any{
		"from":  GetSender(tx),
		"to":    tx.To(),
		"data":  hexutil.Encode(tx.Data()),
		"value": hexutil.EncodeBig(tx.Value()),
		"gas":   hexutil.EncodeUint64(tx.Gas()),
	}
	blockNumber := hexutil.EncodeBig(receipt.BlockNumber)
	traceConfig := map[string]any{
		"tracer": "callTracer",
		"tracerConfig": map[string]interface{}{
			"withLog": true,
		},
		"txIndex": hexutil.EncodeUint64(uint64(txIndex)),
	}

	var traceResult callTrace
	if err := client.Client().CallContext(ctx, &traceResult, "debug_traceCall", callArgs, blockNumber, traceConfig); err != nil {
		log.Fatalf("BackrunBribe: RPC call to debug_traceCall failed: %v", err)
	}

	decodeData(&traceResult)
	return traceResult
}
