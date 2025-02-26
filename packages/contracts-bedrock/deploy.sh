export DEPLOY_CONFIG_PATH="deploy-config/getting-started.json"
forge script -vvv scripts/deploy/Deploy.s.sol:Deploy --ledger --sender 0x57C96c00E3A3c75F52943D707FC422B0fAfC8a38 --broadcast --rpc-url $L1_RPC_URL --slow
