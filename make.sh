#!/bin/bash
docker build . --force-rm  -t inblock/testnet:latest
cd ../docker
docker-compose  -f docker-compose-cc.yaml  up -d
cd -
#docker logs -f --tail 20 cc.metacoin.network
