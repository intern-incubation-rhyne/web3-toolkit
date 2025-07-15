package trace_test

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"testing"
	"time"
	"toolkit/config"
	"toolkit/trace"

	"github.com/ethereum/go-ethereum/ethclient"
)

const txHash = "0xaa164a696b654ff0556c3e723f951f3de95d246176c7d154cd1360d2a9636dd7"
const txIndex = 1

func TestTraceCall(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 61*time.Second)
	defer cancel()

	client, err := ethclient.Dial(config.TRACE_RPC)
	if err != nil {
		log.Fatalf("Failed to connect to Ethereum node: %v", err)
	}

	traceResult := trace.TraceCall(ctx, client, txHash, txIndex)

	b, err := json.MarshalIndent(traceResult, "", "  ")
	if err != nil {
		log.Fatalf("Failed to marshal traceResult: %v", err)
	}
	fmt.Println(string(b))

	if err := os.WriteFile("traceResult.json", b, 0644); err != nil {
		log.Fatalf("Failed to write traceResult to file: %v", err)
	}
}
