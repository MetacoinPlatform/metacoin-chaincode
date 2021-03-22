module inblock.co/metacoin

go 1.15

require (
	github.com/hyperledger/fabric-chaincode-go v0.0.0-20200728190242-9b3ae92d8664
	github.com/hyperledger/fabric-contract-api-go v1.1.1
	github.com/hyperledger/fabric-protos-go v0.0.0-20201028172056-a3136dde2354
	github.com/shopspring/decimal v1.2.0 // indirect
	inblock/metacoin v0.0.0-00010101000000-000000000000
	inblock/metacoin/mtc v0.0.0-00010101000000-000000000000 // indirect
	inblock/metacoin/util v0.0.0-00010101000000-000000000000
)

replace inblock/metacoin => ./inblock/metacoin

replace inblock/metacoin/mtc => ./inblock/metacoin/mtc

replace inblock/metacoin/util => ./inblock/metacoin/util
