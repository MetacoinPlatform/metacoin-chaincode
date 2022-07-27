module metacoin

go 1.15

require (
        github.com/hyperledger/fabric-chaincode-go v0.0.0-20220131132609-1476cf1d3206
        github.com/hyperledger/fabric-protos-go v0.0.0-20220613214546-bf864f01d75e // indirect
        github.com/shopspring/decimal v1.2.0
        inblock/metacoin v0.0.0-00010101000000-000000000000
        inblock/metacoin/mtc v0.0.0-00010101000000-000000000000
        inblock/metacoin/util v0.0.0-00010101000000-000000000000
)

replace (
        inblock/metacoin => ./inblock/metacoin
        inblock/metacoin/mtc => ./inblock/metacoin/mtc
        inblock/metacoin/util => ./inblock/metacoin/util
)
