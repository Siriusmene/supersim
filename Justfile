set positional-arguments

build-book:
    mdbook build ./docs

build-contracts:
    forge --version
    forge build --sizes --root ./contracts src lib/optimism/packages/contracts-bedrock/src/universal/Proxy.sol lib/optimism/packages/contracts-bedrock/src/L2/SuperchainETHBridge.sol lib/optimism/packages/contracts-bedrock/src/L2/ETHLiquidity.sol

# op-core/superchain //go:embeds superchain-configs.zip, which the monorepo
# gitignores, so it is absent from the published module and nothing that reaches
# that package compiles. See ethereum-optimism/optimism#22678. Build it from the
# monorepo submodule instead, which is why go.mod replaces the module with it.
build-superchain-bundle:
    git -C contracts/lib/optimism submodule update --init --force --depth 1 -- superchain-registry
    cd contracts/lib/optimism && bash op-core/superchain/sync-superchain.sh

build-go: build-superchain-bundle
    go build ./...

lint-go: build-superchain-bundle
    golangci-lint run --timeout 5m ./...

test-contracts:
    forge test -vvv --root ./contracts

test-go: build-superchain-bundle
    go test ./... -v

start:
    go run ./...

version-monorepo-contracts:
    cd contracts/lib/optimism && \
    git rev-parse HEAD

version-monorepo-go:
    go list -m -f '{{"{{"}}.Version{{"}}"}}' github.com/ethereum-optimism/optimism

check-monorepo-versions:
    #!/usr/bin/env bash
    ./scripts/check-versions.sh $(just version-monorepo-contracts) $(just version-monorepo-go)

fetch-monorepo-contracts version:
    cd contracts/lib/optimism && \
    git fetch origin {{version}}

install-monorepo-go version:
    go get github.com/ethereum-optimism/optimism@{{version}} && go mod tidy

install-monorepo-contracts version: (fetch-monorepo-contracts version)
    cd contracts && \
    forge install ethereum-optimism/optimism@{{version}}

install-monorepo version: (install-monorepo-go version) (install-monorepo-contracts version)

# The published artifact tarballs stopped being produced, so the checksum-keyed
# bucket URL 404s for any recent monorepo pin. See ethereum-optimism/optimism#22679.
# Build the artifacts from the submodule and hand op-deployer a file:// locator instead.
build-monorepo-contracts:
    cd contracts/lib/optimism/packages/contracts-bedrock && just build-no-tests

monorepo-artifacts-url:
    @echo "file://$(pwd)/contracts/lib/optimism/packages/contracts-bedrock"

vendor-superchain-registry:
    #!/usr/bin/env bash
    ./scripts/vendor-superchain-registry.sh

update-superchain-registry:
    ./scripts/update-superchain-registry-commit-hash.sh

update-and-vendor-superchain-registry: update-superchain-registry vendor-superchain-registry

generate-genesis: build-superchain-bundle build-monorepo-contracts build-contracts
    go run ./genesis/cmd/main.go --monorepo-artifacts $(just monorepo-artifacts-url) --periphery-artifacts ./contracts/out --outdir ./genesis/generated

generate-all version: (install-monorepo version) generate-genesis
