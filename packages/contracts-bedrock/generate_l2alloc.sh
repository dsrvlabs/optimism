export CONTRACT_ADDRESSES_PATH="deployments/1-deploy.json"
export DEPLOY_CONFIG_PATH="deploy-config/dsrv-816.json"
export STATE_DUMP_PATH="../../op-node/l2alloc.json"

forge script scripts/L2Genesis.s.sol:L2Genesis --sig 'runWithStateDump()' --chain-id $L2_CHAIN_ID
