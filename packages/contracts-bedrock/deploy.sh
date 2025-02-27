#!/bin/bash

export DEPLOY_CONFIG_PATH="./deploy-config/dsrv-816.json"

forge script -vvv scripts/deploy/Deploy.s.sol:Deploy --ledger --sender 0x3c5A401F7a54eD38304370a3c93dA1D4d9CC01C1 --broadcast --rpc-url $L1_RPC_URL --slow
