package worldgen

import (
	"fmt"
	"maps"
	"slices"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"

	"github.com/ethereum-optimism/optimism/op-chain-ops/foundry"
	"github.com/ethereum-optimism/optimism/op-chain-ops/genesis"
	"github.com/ethereum-optimism/optimism/op-chain-ops/interopgen"
	"github.com/ethereum-optimism/optimism/op-chain-ops/script"
	"github.com/ethereum-optimism/optimism/op-service/eth"
)

// Mirrors the unexported value in op-chain-ops/interopgen.
var sysGenesisDeployer = common.Address(crypto.Keccak256([]byte("System genesis deployer"))[12:])

// checkStackGenesis runs interopgen's alloc validation while the state still holds
// only the OP Stack genesis. interopgen.CompleteL2 runs it after our periphery
// deploys instead, where it rejects every one of them as a stray account.
func checkStackGenesis(l2Host *script.Host, cfg *interopgen.L2Config) error {
	allocs, err := l2Host.StateDump()
	if err != nil {
		return fmt.Errorf("failed to dump L2 state: %w", err)
	}
	if err := ensureNoDeployed(allocs, sysGenesisDeployer); err != nil {
		return fmt.Errorf("unexpected deployed account content by L2 genesis deployer: %w", err)
	}
	return genesis.CheckL2GenesisAllocs(allocs, genesis.CheckL2AllocsOpts{
		FundDevAccounts: cfg.FundDevAccounts,
		AllowedEOAs:     slices.Collect(maps.Keys(cfg.Prefund)),
	})
}

// completeL2 is interopgen.CompleteL2 without the alloc validation that
// checkStackGenesis has already run. Drop both once CheckL2AllocsOpts can be told
// about contracts that were deployed on purpose.
func completeL2(l2Host *script.Host, cfg *interopgen.L2Config, l1Block *types.Block, deployment *interopgen.L2Deployment) (*interopgen.L2Output, error) {
	deployCfg := &genesis.DeployConfig{
		L2InitializationConfig: cfg.L2InitializationConfig,
		L1DependenciesConfig: genesis.L1DependenciesConfig{
			L1StandardBridgeProxy:       deployment.L1StandardBridgeProxy,
			L1CrossDomainMessengerProxy: deployment.L1CrossDomainMessengerProxy,
			L1ERC721BridgeProxy:         deployment.L1ERC721BridgeProxy,
			SystemConfigProxy:           deployment.SystemConfigProxy,
			OptimismPortalProxy:         deployment.OptimismPortalProxy,
			DAChallengeProxy:            common.Address{}, // unsupported for now
		},
	}
	// l1Block is used to determine genesis time.
	l2Genesis, err := genesis.NewL2Genesis(deployCfg, eth.BlockRefFromHeader(l1Block.Header()))
	if err != nil {
		return nil, fmt.Errorf("failed to build L2 genesis config: %w", err)
	}

	allocs, err := l2Host.StateDump()
	if err != nil {
		return nil, fmt.Errorf("failed to dump L2 state: %w", err)
	}

	if err := ensureNoDeployed(allocs, sysGenesisDeployer); err != nil {
		return nil, fmt.Errorf("unexpected deployed account content by L2 genesis deployer: %w", err)
	}

	for addr, amount := range cfg.Prefund {
		acc := allocs.Accounts[addr]
		acc.Balance = amount
		allocs.Accounts[addr] = acc
	}

	for addr, account := range allocs.Accounts {
		l2Genesis.Alloc[addr] = account
	}
	l2GenesisBlock := l2Genesis.ToBlock()

	rollupCfg, err := deployCfg.RollupConfig(eth.BlockRefFromHeader(l1Block.Header()), l2GenesisBlock.Hash(), l2GenesisBlock.NumberU64())
	if err != nil {
		return nil, fmt.Errorf("failed to build L2 rollup config: %w", err)
	}
	return &interopgen.L2Output{
		Genesis:   l2Genesis,
		RollupCfg: rollupCfg,
	}, nil
}

// Mirrors the unexported helper in op-chain-ops/interopgen.
func ensureNoDeployed(allocs *foundry.ForgeAllocs, deployer common.Address) error {
	for i := uint64(0); i <= allocs.Accounts[deployer].Nonce; i++ {
		addr := crypto.CreateAddress(deployer, i)
		if _, ok := allocs.Accounts[addr]; ok {
			return fmt.Errorf("system deployer output %s (deployed with nonce %d) was not cleaned up", addr, i)
		}
	}
	delete(allocs.Accounts, deployer)
	return nil
}
