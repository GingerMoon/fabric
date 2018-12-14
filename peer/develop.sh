env GOOS=linux GOARCH=amd64 go build
mv peer /c/goworkspace/src/bitbucket.org/accelorteam/demo/fabric-network/bin
cd /c/goworkspace/src/bitbucket.org/accelorteam/demo/fabric-network/first-network
./debug.sh