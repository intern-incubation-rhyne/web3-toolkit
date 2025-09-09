package liquidation

import (
	"context"
	"fmt"
	"math/big"
	"toolkit/query"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethclient"
)

const (
	venusLiquidationSignature = "0x298637f684da70674f26509b10f07ec2fbc77a335ab1e7d6215a4b2484d8bb52"
)

func VenusLiquidations(ctx context.Context, client *ethclient.Client, startBlock *big.Int, endBlock *big.Int) ([]types.Log, error) {
	q := query.QueryConfig{
		FromBlock: startBlock,
		ToBlock:   endBlock,
		Topics:    [][]common.Hash{{common.HexToHash(venusLiquidationSignature)}},
	}
	logs, err := query.PaginatedQuery(ctx, client, q)
	if err != nil {
		return nil, fmt.Errorf("failed to parse ABI: %v", err)
	}
	return logs, nil
}
