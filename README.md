# metacoin-chaincode

This is Metacoin's Chaincode for Hyperledger Fabric.


# Metacoin network consists of 3 components.

1. Metacoin node
- After receiving the API request, confirming that it is the correct call, it is delivered to the Metacoin bridge.
- It saves the transaction and shows it quickly.
- Shows the transactions generated in the Metacoin network in various ways.

2. Metacoin bridge
- Forwards the request of Metacoin node to Hyperledger fabric.

3. Metacoin chaincode
- Block chain is created by processing the request of Metacoin bridge.


# Build chaincode
docker build . -t inblock/metacoin_chaincode:2.1.0


# Metacoin info
- Metacoin : http://metacoin.network
- API Document : http://api.metacoin.network
- Facebook : https://fb.me/metacoinnetwork
- Twitter : https://twitter.com/MetacoinNetwork
- Telegram : https://t.me/metacoinnetwork
- Youtube : https://www.youtube.com/channel/UCjk4Pc23sC_SfCa_dnkS2Mw
- Medium : https://medium.com/metacoin