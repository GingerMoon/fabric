#!/usr/bin/env bash
env GOOS=linux GOARCH=amd64 go build
cd /c/goworkspace/src/bitbucket.org/accelorteam/demo/fabric-network/first-network
./byfn.sh down
rm -f /c/goworkspace/src/bitbucket.org/accelorteam/demo/fabric-network/bin/peer
cd /c/goworkspace/src/github.com/hyperledger/fabric/peer
mv peer /c/goworkspace/src/bitbucket.org/accelorteam/demo/fabric-network/bin
cd /c/goworkspace/src/bitbucket.org/accelorteam/demo/fabric-network/first-network
./debug.sh