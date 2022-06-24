module metacoin

go 1.15

require (
	github.com/hyperledger/fabric-chaincode-go v0.0.0-20200728190242-9b3ae92d8664
	github.com/hyperledger/fabric-protos-go v0.0.0-20201028172056-a3136dde2354
	golang.org/x/tools v0.1.10 // indirect
	inblock/metacoin v0.0.0-00010101000000-000000000000
	inblock/metacoin/mtc v0.0.0-00010101000000-000000000000 // indirect
	inblock/metacoin/util v0.0.0-00010101000000-000000000000
)

replace (
	inblock/metacoin => ./inblock/metacoin
	inblock/metacoin/mtc => ./inblock/metacoin/mtc
	inblock/metacoin/util => ./inblock/metacoin/util
)
