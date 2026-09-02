package opsimulator

import (
	"context"
	"errors"
	"fmt"

	optypes "github.com/ethereum-optimism/optimism/op-core/types"
	"github.com/ethereum-optimism/optimism/op-node/rollup/derive"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"

	"github.com/ethereum/go-ethereum"
)

var _ ethereum.Subscription = &depositTxSubscription{}

type depositTxSubscription struct {
	logSubscription ethereum.Subscription
	logCh           chan types.Log
	errCh           chan error
	doneCh          chan struct{}
}

func (d *depositTxSubscription) Unsubscribe() {
	d.logSubscription.Unsubscribe()
	d.doneCh <- struct{}{}
}

func (d *depositTxSubscription) Err() <-chan error {
	return d.errCh
}

type LogSubscriber interface {
	SubscribeFilterLogs(context.Context, ethereum.FilterQuery, chan<- types.Log) (ethereum.Subscription, error)
}

// transforms Deposit event logs into DepositTx
func SubscribeDepositTx(ctx context.Context, logSub LogSubscriber, depositContractAddr common.Address, ch chan<- *types.DepositTx) (ethereum.Subscription, error) {
	logCh := make(chan types.Log)
	filterQuery := ethereum.FilterQuery{Addresses: []common.Address{depositContractAddr}, Topics: [][]common.Hash{{derive.DepositEventABIHash}}}
	logSubscription, err := logSub.SubscribeFilterLogs(ctx, filterQuery, logCh)
	if err != nil {
		return nil, fmt.Errorf("failed to create log subscription: %w", err)
	}

	errCh := make(chan error)
	doneCh := make(chan struct{})
	logErrCh := logSubscription.Err()

	go func() {
		defer close(logCh)
		defer close(errCh)
		defer close(doneCh)
		for {
			select {
			case log := <-logCh:
				dep, err := logToDepositTx(&log)
				if err != nil {
					errCh <- err
					continue
				}
				ch <- dep
			case err := <-logErrCh:
				errCh <- fmt.Errorf("log subscription error: %w", err)
			case <-ctx.Done():
				return
			case <-doneCh:
				return
			}
		}
	}()

	return &depositTxSubscription{logSubscription, logCh, errCh, doneCh}, nil
}

func logToDepositTx(log *types.Log) (*types.DepositTx, error) {
	if len(log.Topics) > 0 && log.Topics[0] == derive.DepositEventABIHash {
		dep, err := derive.UnmarshalDepositLogEvent(log)
		if err != nil {
			return nil, err
		}
		return toGethDepositTx(dep), nil
	} else {
		return nil, errors.New("log is not a deposit event")
	}
}

// The monorepo moved DepositTx into op-core/types, but op-geth keeps its own
// copy and only that one satisfies types.TxData, so it is still what we send.
func toGethDepositTx(dep *optypes.DepositTx) *types.DepositTx {
	return &types.DepositTx{
		SourceHash:          dep.SourceHash,
		From:                dep.From,
		To:                  dep.To,
		Mint:                dep.Mint,
		Value:               dep.Value,
		Gas:                 dep.Gas,
		IsSystemTransaction: dep.IsSystemTransaction,
		Data:                dep.Data,
	}
}
