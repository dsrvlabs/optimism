#!/bin/bash

./bin/op-node genesis l2 \
  --l1-rpc $L1_RPC_URL \
  --deploy-config ../packages/contracts-bedrock/deploy-config/dsrv-816.json \
  --l2-allocs l2alloc.json \
  --l1-deployments ../packages/contracts-bedrock/deployments/1-deploy.json \
  --outfile.l2 genesis.json \
  --outfile.rollup rollup.json

