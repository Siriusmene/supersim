# First steps

`supersim` allows testing multichain features **locally**. Previously, testing multichain features required complex docker setups or using a testnet.

To see it in practice, let's first try sending some ETH from the L1 to the L2.

## Deposit ETH from the L1 into the L2 (L1 to L2 message passing)

**1. Check initial balance on the L2 (chain 901)**

Grab the balance of the sender account on L2:

```sh
cast balance 0xf39Fd6e51aad88F6F4ce6aB8827279cffFb92266 --rpc-url http://127.0.0.1:9545
```

<br>

**2. Send the Ether** 
<br />
It exists two  different methods to do this action

### First method - `OptimismPortal`
Send the Ether to `OptimismPortal` contract of the respective L2 (on chain 900)**

Initiate a bridge transaction on the L1:<br>
*In the case of the chain 901 the contract is `0x37a418800d0c812A9dE83Bc80e993A6b76511B57`* 

```sh
cast send 0x37a418800d0c812A9dE83Bc80e993A6b76511B57 --value 0.1ether --rpc-url http://127.0.0.1:8545 --private-key 0xac0974bec39a17e36ba4a6b4d238ff944bacb478cbed5efcae784d7bf4f2ff80
```

### Second method - `L1StandardBridge`

**Call `bridgeETH` function on the `L1StandardBridgeProxy` / `L1StandardBridge` contract of the respective L2 on L1 (chain 900)**
*In the case of the chain 901 the contract is `0x8d515eb0e5F293B16B6bBCA8275c060bAe0056B0`* 

Initiate a bridge transaction on the L1:

```sh
cast send 0x8d515eb0e5F293B16B6bBCA8275c060bAe0056B0 "bridgeETH(uint32 _minGasLimit, bytes calldata _extraData)" 50000 0x --value 0.1ether --rpc-url http://127.0.0.1:8545 --private-key 0xac0974bec39a17e36ba4a6b4d238ff944bacb478cbed5efcae784d7bf4f2ff80
```

<br>

**3. Check the balance on the L2 (chain 901)**

Verify that the ETH balance of the sender has increased on the L2:

```sh
cast balance 0xf39Fd6e51aad88F6F4ce6aB8827279cffFb92266 --rpc-url http://127.0.0.1:9545
```

With the steps above, you've now completed an L1 to L2 ETH bridge locally using `supersim`. This approach simplifies multichain testing, allowing you to focus on development without the need for complex setups or relying on external testnets.
